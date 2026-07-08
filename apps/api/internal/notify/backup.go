package notify

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/leotime/leotime/apps/api/internal/config"
	"github.com/leotime/leotime/apps/api/internal/outbox"
	"github.com/leotime/leotime/apps/api/internal/store"
)

type BackupNotifier struct {
	store         *store.Store
	outbox        *outbox.Store
	publicBaseURL string
	maxAttempts   int
}

func NewBackupNotifier(st *store.Store, outboxStore *outbox.Store, cfg config.Config) *BackupNotifier {
	return &BackupNotifier{
		store:         st,
		outbox:        outboxStore,
		publicBaseURL: cfg.PublicBaseURL,
		maxAttempts:   cfg.MailMaxAttempts,
	}
}

func (n *BackupNotifier) EnqueueBackupResult(ctx context.Context, userID, objectKey, errMsg string, success bool, finishedAt time.Time) {
	if n == nil {
		return
	}
	if err := n.enqueueBackupResult(ctx, userID, objectKey, errMsg, success, finishedAt); err != nil {
		log.Printf("backup email notification failed: %v", err)
	}
}

func (n *BackupNotifier) EnqueueRestoreResult(ctx context.Context, userID, objectKey, errMsg string, success bool, finishedAt time.Time) {
	if n == nil {
		return
	}
	if err := n.enqueueRestoreResult(ctx, userID, objectKey, errMsg, success, finishedAt); err != nil {
		log.Printf("restore email notification failed: %v", err)
	}
}

func (n *BackupNotifier) enqueueBackupResult(ctx context.Context, userID, objectKey, errMsg string, success bool, finishedAt time.Time) error {
	profile, err := n.store.ProfileByUserID(ctx, userID)
	if err != nil {
		return fmt.Errorf("load profile: %w", err)
	}

	if success && !profile.Settings.BackupEmailOnSuccess {
		return nil
	}
	if !success && !profile.Settings.BackupEmailOnFailure {
		return nil
	}

	kind := outbox.KindBackupFailure
	if success {
		kind = outbox.KindBackupSuccess
	}

	_, err = n.outbox.Enqueue(ctx, outbox.EnqueueInput{
		UserID:      userID,
		Kind:        kind,
		ToAddress:   profile.Email,
		Subject:     backupEmailSubject(profile.Locale, success),
		BodyText:    backupEmailBody(profile, n.publicBaseURL, objectKey, errMsg, success, finishedAt),
		MaxAttempts: n.maxAttempts,
	}, finishedAt)
	return err
}

func (n *BackupNotifier) enqueueRestoreResult(ctx context.Context, userID, objectKey, errMsg string, success bool, finishedAt time.Time) error {
	profile, err := n.store.ProfileByUserID(ctx, userID)
	if err != nil {
		return fmt.Errorf("load profile: %w", err)
	}

	if success && !profile.Settings.RestoreEmailOnSuccess {
		return nil
	}
	if !success && !profile.Settings.RestoreEmailOnFailure {
		return nil
	}

	kind := outbox.KindRestoreFailure
	if success {
		kind = outbox.KindRestoreSuccess
	}

	_, err = n.outbox.Enqueue(ctx, outbox.EnqueueInput{
		UserID:      userID,
		Kind:        kind,
		ToAddress:   profile.Email,
		Subject:     restoreEmailSubject(profile.Locale, success),
		BodyText:    restoreEmailBody(profile, n.publicBaseURL, objectKey, errMsg, success, finishedAt),
		MaxAttempts: n.maxAttempts,
	}, finishedAt)
	return err
}

func backupEmailSubject(locale string, success bool) string {
	if strings.EqualFold(strings.TrimSpace(locale), "en") {
		if success {
			return "leotime backup completed"
		}
		return "leotime backup failed"
	}
	if success {
		return "Copia de seguridad de leotime completada"
	}
	return "Copia de seguridad de leotime fallida"
}

func restoreEmailSubject(locale string, success bool) string {
	if strings.EqualFold(strings.TrimSpace(locale), "en") {
		if success {
			return "leotime restore completed"
		}
		return "leotime restore failed"
	}
	if success {
		return "Restauracion de leotime completada"
	}
	return "Restauracion de leotime fallida"
}

func backupEmailBody(profile *store.Profile, publicBaseURL, objectKey, errMsg string, success bool, finishedAt time.Time) string {
	dashboardURL := strings.TrimRight(strings.TrimSpace(publicBaseURL), "/")
	if dashboardURL == "" {
		dashboardURL = "http://127.0.0.1:8080"
	}

	name := displayName(profile.Name, profile.Email)
	when := formatTimestamp(finishedAt)
	object := displayValue(objectKey, "—")
	errorText := displayValue(errMsg, "—")

	if strings.EqualFold(strings.TrimSpace(profile.Locale), "en") {
		if success {
			return fmt.Sprintf(
				"Hi %s,\n\nYour leotime database backup finished successfully.\n\nObject: %s\nFinished: %s\n\nOpen leotime: %s\n\nYou can change backup email notifications in Settings.",
				name, object, when, dashboardURL,
			)
		}
		return fmt.Sprintf(
			"Hi %s,\n\nYour leotime database backup failed.\n\nError: %s\nObject: %s\nFinished: %s\n\nOpen leotime: %s\n\nYou can change backup email notifications in Settings.",
			name, errorText, object, when, dashboardURL,
		)
	}

	if success {
		return fmt.Sprintf(
			"Hola %s,\n\nLa copia de seguridad de tu base de datos leotime se completo correctamente.\n\nObjeto: %s\nFinalizada: %s\n\nAbrir leotime: %s\n\nPuedes cambiar las notificaciones de backup en Ajustes.",
			name, object, when, dashboardURL,
		)
	}
	return fmt.Sprintf(
		"Hola %s,\n\nLa copia de seguridad de tu base de datos leotime fallo.\n\nError: %s\nObjeto: %s\nFinalizada: %s\n\nAbrir leotime: %s\n\nPuedes cambiar las notificaciones de backup en Ajustes.",
		name, errorText, object, when, dashboardURL,
	)
}

func restoreEmailBody(profile *store.Profile, publicBaseURL, objectKey, errMsg string, success bool, finishedAt time.Time) string {
	dashboardURL := strings.TrimRight(strings.TrimSpace(publicBaseURL), "/")
	if dashboardURL == "" {
		dashboardURL = "http://127.0.0.1:8080"
	}

	name := displayName(profile.Name, profile.Email)
	when := formatTimestamp(finishedAt)
	object := displayValue(objectKey, "—")
	errorText := displayValue(errMsg, "—")

	if strings.EqualFold(strings.TrimSpace(profile.Locale), "en") {
		if success {
			return fmt.Sprintf(
				"Hi %s,\n\nYour leotime database restore finished successfully.\n\nObject: %s\nFinished: %s\n\nOpen leotime: %s\n\nYou can change restore email notifications in Settings.",
				name, object, when, dashboardURL,
			)
		}
		return fmt.Sprintf(
			"Hi %s,\n\nYour leotime database restore failed.\n\nError: %s\nObject: %s\nFinished: %s\n\nOpen leotime: %s\n\nYou can change restore email notifications in Settings.",
			name, errorText, object, when, dashboardURL,
		)
	}

	if success {
		return fmt.Sprintf(
			"Hola %s,\n\nLa restauracion de tu base de datos leotime se completo correctamente.\n\nObjeto: %s\nFinalizada: %s\n\nAbrir leotime: %s\n\nPuedes cambiar las notificaciones de restore en Ajustes.",
			name, object, when, dashboardURL,
		)
	}
	return fmt.Sprintf(
		"Hola %s,\n\nLa restauracion de tu base de datos leotime fallo.\n\nError: %s\nObjeto: %s\nFinalizada: %s\n\nAbrir leotime: %s\n\nPuedes cambiar las notificaciones de restore en Ajustes.",
		name, errorText, object, when, dashboardURL,
	)
}
