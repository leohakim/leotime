package backup

import (
	"context"
	"database/sql"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/leotime/leotime/apps/api/internal/backup/crypto"
	"github.com/leotime/leotime/apps/api/internal/backup/snapshot"
	"github.com/leotime/leotime/apps/api/internal/backup/storage"
	"github.com/leotime/leotime/apps/api/internal/config"
	"github.com/leotime/leotime/apps/api/internal/db"
	"github.com/leotime/leotime/apps/api/internal/maintenance"
	"github.com/leotime/leotime/apps/api/internal/metrics"
	"github.com/leotime/leotime/apps/api/internal/store"
	sqlite "modernc.org/sqlite"
)

type ClientFactory func(ctx context.Context, cfg storage.S3Config) (storage.Client, error)

type EmailNotifier interface {
	EnqueueBackupResult(ctx context.Context, userID, objectKey, errMsg string, success bool, finishedAt time.Time)
	EnqueueRestoreResult(ctx context.Context, userID, objectKey, errMsg string, success bool, finishedAt time.Time)
}

type Service struct {
	cfg           config.Config
	store         *store.Store
	db            *sql.DB
	clientFactory ClientFactory
	notifier      EmailNotifier
	mu            sync.Mutex
}

type RunResult struct {
	Status     string `json:"status"`
	ObjectKey  string `json:"objectKey"`
	SizeBytes  int64  `json:"sizeBytes"`
	StartedAt  string `json:"startedAt"`
	FinishedAt string `json:"finishedAt"`
	Error      string `json:"error,omitempty"`
}

type RestoreResult struct {
	Status             string `json:"status"`
	ObjectKey          string `json:"objectKey"`
	SafetySnapshotPath string `json:"-"`
	StartedAt          string `json:"startedAt"`
	FinishedAt         string `json:"finishedAt"`
	RequiresRestart    bool   `json:"requiresRestart"`
	Error              string `json:"error,omitempty"`
}

type ObjectResponse struct {
	Key          string `json:"key"`
	SizeBytes    int64  `json:"sizeBytes"`
	LastModified string `json:"lastModified"`
}

func NewService(cfg config.Config, st *store.Store, db *sql.DB, notifier EmailNotifier) *Service {
	return &Service{
		cfg:           cfg,
		store:         st,
		db:            db,
		clientFactory: defaultClientFactory,
		notifier:      notifier,
	}
}

// SetClientFactory replaces the default remote storage client factory.
func (s *Service) SetClientFactory(factory ClientFactory) {
	if factory != nil {
		s.clientFactory = factory
	}
}

func defaultClientFactory(ctx context.Context, cfg storage.S3Config) (storage.Client, error) {
	return storage.NewS3Client(ctx, cfg)
}

func (s *Service) GetSettings(ctx context.Context, userID string) (*store.BackupSettings, error) {
	return s.store.EnsureBackupSettingsDefaults(ctx, userID)
}

func (s *Service) SaveSettings(ctx context.Context, userID string, input store.BackupSettingsInput) (*store.BackupSettings, error) {
	secretEnc := ""
	if input.SecretAccessKey != "" {
		key, err := s.secretsKey()
		if err != nil {
			return nil, err
		}
		encoded, err := crypto.Encrypt([]byte(input.SecretAccessKey), key)
		if err != nil {
			return nil, fmt.Errorf("encrypt secret access key: %w", err)
		}
		secretEnc = encoded
	}

	return s.store.UpsertBackupSettings(ctx, userID, input, secretEnc)
}

func (s *Service) ListObjects(ctx context.Context, userID string) ([]ObjectResponse, error) {
	resolved, client, err := s.loadClient(ctx, userID)
	if err != nil {
		return nil, err
	}

	objects, err := client.List(ctx, resolved.prefix)
	if err != nil {
		return nil, wrapRemoteError(err)
	}
	sortStorageObjectsDesc(objects)

	response := make([]ObjectResponse, 0, len(objects))
	for _, object := range objects {
		if !isBackupArchiveKey(object.Key) {
			continue
		}
		response = append(response, ObjectResponse{
			Key:          object.Key,
			SizeBytes:    object.SizeBytes,
			LastModified: object.LastModified.UTC().Format(time.RFC3339),
		})
	}
	sortObjectResponsesDesc(response)
	return response, nil
}

func (s *Service) Run(ctx context.Context, userID string, force bool) (*RunResult, error) {
	if !s.mu.TryLock() {
		return nil, ErrBusy
	}
	defer s.mu.Unlock()

	started := time.Now().UTC()
	result := &RunResult{
		StartedAt: started.Format(time.RFC3339),
	}

	var notifySuccess bool
	var notifyObjectKey string
	var notifyError string
	var shouldNotify bool
	defer func() {
		if shouldNotify && s.notifier != nil {
			s.notifier.EnqueueBackupResult(ctx, userID, notifyObjectKey, notifyError, notifySuccess, time.Now().UTC())
		}
	}()

	profile, err := s.store.ProfileByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}

	settings, err := s.store.EnsureBackupSettingsDefaults(ctx, userID)
	if err != nil {
		return nil, err
	}

	due, err := IsDue(EnabledSettings{
		Enabled:      settings.Enabled,
		ScheduleHour: settings.ScheduleHour,
		LastRunAt:    settings.LastRunAt,
		LastStatus:   settings.LastStatus,
	}, profile.Settings.Timezone, started, force)
	if err != nil {
		return nil, err
	}
	if !due {
		result.Status = "skipped"
		result.FinishedAt = time.Now().UTC().Format(time.RFC3339)
		return result, nil
	}

	resolved, client, err := s.loadClient(ctx, userID)
	if err != nil {
		_ = s.store.UpdateBackupRunStatus(ctx, userID, "failed", err.Error(), "")
		result.Status = "failed"
		result.Error = err.Error()
		result.FinishedAt = time.Now().UTC().Format(time.RFC3339)
		metrics.BackupFailuresTotal.Inc()
		shouldNotify = true
		notifySuccess = false
		notifyError = err.Error()
		return result, err
	}

	timer := metrics.NewBackupTimer()
	defer timer.ObserveDuration()

	dir, err := os.MkdirTemp("", "leotime-backup-*")
	if err != nil {
		return nil, fmt.Errorf("create temp dir: %w", err)
	}
	defer os.RemoveAll(dir)

	snapshotPath := filepath.Join(dir, "snapshot.db")
	archivePath := filepath.Join(dir, "snapshot.tar.gz")
	if err := snapshot.SnapshotToFile(ctx, s.cfg.DBPath, snapshotPath); err != nil {
		_ = s.store.UpdateBackupRunStatus(ctx, userID, "failed", err.Error(), "")
		metrics.BackupFailuresTotal.Inc()
		shouldNotify = true
		notifySuccess = false
		notifyError = err.Error()
		return nil, err
	}
	if err := CreateArchive(snapshotPath, s.cfg.DocumentRoot, archivePath); err != nil {
		_ = s.store.UpdateBackupRunStatus(ctx, userID, "failed", err.Error(), "")
		metrics.BackupFailuresTotal.Inc()
		shouldNotify = true
		notifySuccess = false
		notifyError = err.Error()
		return nil, err
	}

	info, err := os.Stat(archivePath)
	if err != nil {
		return nil, err
	}

	objectKey := resolved.prefix + "leotime-" + started.Format("20060102T150405Z") + ".tar.gz"
	file, err := os.Open(archivePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	if err := client.Put(ctx, objectKey, file, "application/gzip"); err != nil {
		_ = s.store.UpdateBackupRunStatus(ctx, userID, "failed", err.Error(), "")
		metrics.BackupFailuresTotal.Inc()
		shouldNotify = true
		notifySuccess = false
		notifyObjectKey = objectKey
		notifyError = err.Error()
		return nil, wrapRemoteError(err)
	}

	if err := s.pruneOldBackups(ctx, client, resolved.prefix, settings.RetentionDays, started); err != nil {
		log.Printf("backup prune warning (upload succeeded): %v", err)
	}

	_ = s.store.UpdateBackupRunStatus(ctx, userID, "success", "", objectKey)
	metrics.BackupLastSuccessTimestamp.Set(float64(time.Now().Unix()))

	result.Status = "success"
	result.ObjectKey = objectKey
	result.SizeBytes = info.Size()
	result.FinishedAt = time.Now().UTC().Format(time.RFC3339)
	shouldNotify = true
	notifySuccess = true
	notifyObjectKey = objectKey
	return result, nil
}

func (s *Service) Restore(ctx context.Context, userID, objectKey string, latest bool) (*RestoreResult, error) {
	if !s.mu.TryLock() {
		return nil, ErrBusy
	}
	defer s.mu.Unlock()

	maintenance.Enter()
	defer maintenance.Leave()

	started := time.Now().UTC()
	result := &RestoreResult{StartedAt: started.Format(time.RFC3339)}

	var notifySuccess bool
	var notifyObjectKey string
	var notifyError string
	var shouldNotify bool
	defer func() {
		if shouldNotify && s.notifier != nil {
			s.notifier.EnqueueRestoreResult(ctx, userID, notifyObjectKey, notifyError, notifySuccess, time.Now().UTC())
		}
	}()

	resolved, client, err := s.loadClient(ctx, userID)
	if err != nil {
		return nil, err
	}

	if latest {
		objects, err := s.ListObjects(ctx, userID)
		if err != nil {
			return nil, err
		}
		if len(objects) == 0 {
			err = fmt.Errorf("no backup objects found")
			_ = s.store.UpdateBackupRestoreStatus(ctx, userID, "failed", err.Error(), "")
			metrics.BackupRestoreFailuresTotal.Inc()
			shouldNotify = true
			notifySuccess = false
			notifyError = err.Error()
			return nil, err
		}
		objectKey = objects[0].Key
	}

	if objectKey == "" {
		err = fmt.Errorf("objectKey is required")
		_ = s.store.UpdateBackupRestoreStatus(ctx, userID, "failed", err.Error(), "")
		metrics.BackupRestoreFailuresTotal.Inc()
		shouldNotify = true
		notifySuccess = false
		notifyError = err.Error()
		return nil, err
	}
	if !strings.HasPrefix(objectKey, resolved.prefix) {
		err = fmt.Errorf("object key outside configured prefix")
		_ = s.store.UpdateBackupRestoreStatus(ctx, userID, "failed", err.Error(), objectKey)
		metrics.BackupRestoreFailuresTotal.Inc()
		shouldNotify = true
		notifySuccess = false
		notifyObjectKey = objectKey
		notifyError = err.Error()
		return nil, err
	}

	dir, err := os.MkdirTemp("", "leotime-restore-*")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(dir)

	downloadPath := filepath.Join(dir, "restore-download")
	restoreDBPath := filepath.Join(dir, "restore.db")

	reader, err := client.Get(ctx, objectKey)
	if err != nil {
		_ = s.store.UpdateBackupRestoreStatus(ctx, userID, "failed", err.Error(), objectKey)
		metrics.BackupRestoreFailuresTotal.Inc()
		shouldNotify = true
		notifySuccess = false
		notifyObjectKey = objectKey
		notifyError = err.Error()
		return nil, wrapRemoteError(err)
	}
	file, err := os.Create(downloadPath)
	if err != nil {
		reader.Close()
		return nil, err
	}
	if _, err := io.Copy(file, reader); err != nil {
		file.Close()
		reader.Close()
		return nil, err
	}
	file.Close()
	reader.Close()

	extractDir := filepath.Join(dir, "extracted")
	var restoreDocumentsDir string
	if strings.HasSuffix(objectKey, ".tar.gz") {
		if _, err := ExtractArchive(downloadPath, extractDir); err != nil {
			_ = s.store.UpdateBackupRestoreStatus(ctx, userID, "failed", err.Error(), objectKey)
			metrics.BackupRestoreFailuresTotal.Inc()
			shouldNotify = true
			notifySuccess = false
			notifyObjectKey = objectKey
			notifyError = err.Error()
			return nil, err
		}
		restoreDBPath = filepath.Join(extractDir, archiveDBFileName)
		restoreDocumentsDir = filepath.Join(extractDir, archiveDocumentsDir)
	} else {
		if err := snapshot.GunzipToFile(downloadPath, restoreDBPath); err != nil {
			_ = s.store.UpdateBackupRestoreStatus(ctx, userID, "failed", err.Error(), objectKey)
			metrics.BackupRestoreFailuresTotal.Inc()
			shouldNotify = true
			notifySuccess = false
			notifyObjectKey = objectKey
			notifyError = err.Error()
			return nil, err
		}
	}
	minMigrationVersion, err := db.LatestMigrationVersion()
	if err != nil {
		return nil, err
	}
	if err := snapshot.ValidateDatabase(ctx, restoreDBPath, minMigrationVersion); err != nil {
		_ = s.store.UpdateBackupRestoreStatus(ctx, userID, "failed", err.Error(), objectKey)
		metrics.BackupRestoreFailuresTotal.Inc()
		shouldNotify = true
		notifySuccess = false
		notifyObjectKey = objectKey
		notifyError = err.Error()
		return nil, err
	}

	safetyPath := filepath.Join(filepath.Dir(s.cfg.DBPath), "leotime-pre-restore-"+started.Format("20060102T150405Z")+".db.gz")
	safetySnapshot := filepath.Join(dir, "safety.db")
	if err := snapshot.SnapshotToFile(ctx, s.cfg.DBPath, safetySnapshot); err != nil {
		_ = s.store.UpdateBackupRestoreStatus(ctx, userID, "failed", err.Error(), objectKey)
		metrics.BackupRestoreFailuresTotal.Inc()
		shouldNotify = true
		notifySuccess = false
		notifyObjectKey = objectKey
		notifyError = err.Error()
		return nil, err
	}
	if err := snapshot.GzipFile(safetySnapshot, safetyPath); err != nil {
		_ = s.store.UpdateBackupRestoreStatus(ctx, userID, "failed", err.Error(), objectKey)
		metrics.BackupRestoreFailuresTotal.Inc()
		shouldNotify = true
		notifySuccess = false
		notifyObjectKey = objectKey
		notifyError = err.Error()
		return nil, err
	}

	if err := copyDatabaseInto(ctx, restoreDBPath, s.db); err != nil {
		_ = s.store.UpdateBackupRestoreStatus(ctx, userID, "failed", err.Error(), objectKey)
		metrics.BackupRestoreFailuresTotal.Inc()
		shouldNotify = true
		notifySuccess = false
		notifyObjectKey = objectKey
		notifyError = err.Error()
		return nil, err
	}

	if restoreDocumentsDir != "" {
		if err := replaceDocumentRoot(restoreDocumentsDir, s.cfg.DocumentRoot); err != nil {
			_ = s.store.UpdateBackupRestoreStatus(ctx, userID, "failed", err.Error(), objectKey)
			metrics.BackupRestoreFailuresTotal.Inc()
			shouldNotify = true
			notifySuccess = false
			notifyObjectKey = objectKey
			notifyError = err.Error()
			return nil, err
		}
	}

	_ = s.store.UpdateBackupRestoreStatus(ctx, userID, "success", "", objectKey)
	metrics.BackupRestoreSuccessTotal.Inc()

	result.Status = "success"
	result.ObjectKey = objectKey
	result.SafetySnapshotPath = safetyPath
	result.FinishedAt = time.Now().UTC().Format(time.RFC3339)
	result.RequiresRestart = true
	shouldNotify = true
	notifySuccess = true
	notifyObjectKey = objectKey
	return result, nil
}

func (s *Service) RunScheduled(ctx context.Context) error {
	if !s.cfg.BackupSchedulerEnabled {
		return nil
	}

	user, err := s.store.UserByEmail(ctx, s.cfg.BootstrapEmail)
	if err != nil {
		return err
	}

	settings, err := s.store.EnsureBackupSettingsDefaults(ctx, user.ID)
	if err != nil || !settings.Enabled {
		return err
	}

	_, err = s.Run(ctx, user.ID, false)
	if err != nil {
		return err
	}
	return nil
}

func (s *Service) loadClient(ctx context.Context, userID string) (*resolvedS3Config, storage.Client, error) {
	resolved, err := s.resolveS3Config(ctx, userID, nil, true)
	if err != nil {
		return nil, nil, err
	}

	client, err := s.clientFactory(ctx, resolved.cfg)
	if err != nil {
		return nil, nil, wrapRemoteError(err)
	}

	return resolved, client, nil
}

func (s *Service) secretsKey() ([]byte, error) {
	if strings.TrimSpace(s.cfg.SecretsKey) == "" {
		return nil, store.ErrBackupSecretsKeyMissing
	}
	return crypto.ParseKey(s.cfg.SecretsKey)
}

func (s *Service) decryptSecret(encoded string) (string, error) {
	key, err := s.secretsKey()
	if err != nil {
		return "", err
	}
	plain, err := crypto.Decrypt(encoded, key)
	if err != nil {
		return "", err
	}
	return string(plain), nil
}

func (s *Service) pruneOldBackups(ctx context.Context, client storage.Client, prefix string, retentionDays int, now time.Time) error {
	objects, err := client.List(ctx, prefix)
	if err != nil {
		return err
	}

	cutoff := now.AddDate(0, 0, -retentionDays)
	for _, object := range objects {
		if !isBackupArchiveKey(object.Key) {
			continue
		}
		if object.LastModified.Before(cutoff) {
			if err := client.Delete(ctx, object.Key); err != nil {
				return wrapRemoteError(err)
			}
		}
	}
	return nil
}

func copyDatabaseInto(ctx context.Context, srcPath string, destDB *sql.DB) error {
	destConn, err := destDB.Conn(ctx)
	if err != nil {
		return fmt.Errorf("dest conn: %w", err)
	}
	defer destConn.Close()

	return destConn.Raw(func(raw any) error {
		restorer, ok := raw.(interface {
			NewRestore(string) (*sqlite.Backup, error)
		})
		if !ok {
			return fmt.Errorf("sqlite driver does not support restore")
		}

		backup, err := restorer.NewRestore(srcPath)
		if err != nil {
			return fmt.Errorf("begin restore: %w", err)
		}

		for {
			more, err := backup.Step(-1)
			if err != nil {
				_ = backup.Finish()
				return fmt.Errorf("restore step: %w", err)
			}
			if !more {
				break
			}
		}

		remoteConn, err := backup.Commit()
		if remoteConn != nil {
			_ = remoteConn.Close()
		}
		if err != nil {
			return fmt.Errorf("commit restore: %w", err)
		}
		return nil
	})
}
