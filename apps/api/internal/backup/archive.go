package backup

import (
	"archive/tar"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

func isBackupArchiveKey(key string) bool {
	return strings.HasSuffix(key, ".tar.gz") || strings.HasSuffix(key, ".db.gz")
}

func CreateArchive(dbPath, documentRoot, destPath string) error {
	if err := os.MkdirAll(filepath.Dir(destPath), 0o755); err != nil {
		return fmt.Errorf("create archive directory: %w", err)
	}

	manifest, err := BuildManifest(dbPath, documentRoot)
	if err != nil {
		return err
	}

	destination, err := os.Create(destPath)
	if err != nil {
		return fmt.Errorf("create archive: %w", err)
	}
	defer destination.Close()

	gzipWriter := gzip.NewWriter(destination)
	tarWriter := tar.NewWriter(gzipWriter)

	manifestData, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal manifest: %w", err)
	}
	if err := writeTarBytes(tarWriter, manifestFileName, manifestData, 0o644); err != nil {
		return err
	}
	if err := addFileToTar(tarWriter, dbPath, archiveDBFileName); err != nil {
		return fmt.Errorf("add database to archive: %w", err)
	}
	root := strings.TrimSpace(documentRoot)
	if root != "" {
		if err := addDirectoryToTar(tarWriter, root, archiveDocumentsDir); err != nil {
			return fmt.Errorf("add documents to archive: %w", err)
		}
	}
	if err := tarWriter.Close(); err != nil {
		return fmt.Errorf("close tar writer: %w", err)
	}
	if err := gzipWriter.Close(); err != nil {
		return fmt.Errorf("close gzip writer: %w", err)
	}
	return destination.Close()
}

func ExtractArchive(archivePath, destDir string) (*Manifest, error) {
	if err := os.MkdirAll(destDir, 0o755); err != nil {
		return nil, fmt.Errorf("create extract directory: %w", err)
	}

	source, err := os.Open(archivePath)
	if err != nil {
		return nil, fmt.Errorf("open archive: %w", err)
	}
	defer source.Close()

	gzipReader, err := gzip.NewReader(source)
	if err != nil {
		return nil, fmt.Errorf("open gzip reader: %w", err)
	}
	defer gzipReader.Close()

	tarReader := tar.NewReader(gzipReader)
	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("read tar entry: %w", err)
		}
		if header.Typeflag != tar.TypeReg && header.Typeflag != tar.TypeDir {
			continue
		}

		targetPath := filepath.Join(destDir, filepath.FromSlash(header.Name))
		rel, err := filepath.Rel(destDir, targetPath)
		if err != nil || strings.HasPrefix(rel, "..") {
			return nil, fmt.Errorf("archive path escapes extract dir: %s", header.Name)
		}

		if header.Typeflag == tar.TypeDir {
			if err := os.MkdirAll(targetPath, 0o755); err != nil {
				return nil, fmt.Errorf("create archive directory: %w", err)
			}
			continue
		}

		if err := os.MkdirAll(filepath.Dir(targetPath), 0o755); err != nil {
			return nil, fmt.Errorf("create archive parent directory: %w", err)
		}

		destination, err := os.OpenFile(targetPath, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, os.FileMode(header.Mode))
		if err != nil {
			return nil, fmt.Errorf("create archive file: %w", err)
		}
		if _, err := io.Copy(destination, tarReader); err != nil {
			destination.Close()
			return nil, fmt.Errorf("extract archive file: %w", err)
		}
		if err := destination.Close(); err != nil {
			return nil, fmt.Errorf("close archive file: %w", err)
		}
	}

	manifest, err := ReadManifest(filepath.Join(destDir, manifestFileName))
	if err != nil {
		return nil, fmt.Errorf("read manifest: %w", err)
	}
	if err := ValidateManifest(manifest, destDir); err != nil {
		return nil, err
	}
	return manifest, nil
}

const documentRestoreStagingSuffix = ".restore-staging"
const documentRestoreBackupSuffix = ".restore-backup"

func siblingDocumentPath(documentRoot, suffix string) string {
	root := strings.TrimSpace(documentRoot)
	return filepath.Join(filepath.Dir(root), filepath.Base(root)+suffix)
}

func stageRestoredDocuments(manifest *Manifest, sourceDir, documentRoot string) (stagingDir string, err error) {
	root := strings.TrimSpace(documentRoot)
	if root == "" {
		return "", nil
	}

	stagingDir = siblingDocumentPath(root, documentRestoreStagingSuffix)
	if err := os.RemoveAll(stagingDir); err != nil {
		return "", fmt.Errorf("clear document staging: %w", err)
	}

	switch _, statErr := os.Stat(sourceDir); {
	case statErr == nil:
		if err := copyDir(sourceDir, stagingDir); err != nil {
			return "", fmt.Errorf("copy documents to staging: %w", err)
		}
	case os.IsNotExist(statErr):
		if err := os.MkdirAll(stagingDir, 0o755); err != nil {
			return "", fmt.Errorf("create empty document staging: %w", err)
		}
	default:
		return "", fmt.Errorf("stat restored documents: %w", statErr)
	}

	if err := ValidateDocumentManifest(manifest, stagingDir); err != nil {
		_ = os.RemoveAll(stagingDir)
		return "", err
	}
	return stagingDir, nil
}

func backupLiveDocuments(documentRoot string) (backupDir string, err error) {
	root := strings.TrimSpace(documentRoot)
	if root == "" {
		return "", nil
	}

	backupDir = siblingDocumentPath(root, documentRestoreBackupSuffix)
	if err := os.RemoveAll(backupDir); err != nil {
		return "", fmt.Errorf("clear document backup: %w", err)
	}

	switch _, statErr := os.Stat(root); {
	case statErr == nil:
		if err := copyDir(root, backupDir); err != nil {
			return "", fmt.Errorf("backup live documents: %w", err)
		}
	case os.IsNotExist(statErr):
		if err := os.MkdirAll(backupDir, 0o755); err != nil {
			return "", fmt.Errorf("create empty document backup: %w", err)
		}
	default:
		return "", fmt.Errorf("stat live documents: %w", statErr)
	}
	return backupDir, nil
}

func promoteStagedDocuments(stagingDir, documentRoot string) error {
	root := strings.TrimSpace(documentRoot)
	if root == "" || stagingDir == "" {
		return nil
	}
	if err := os.RemoveAll(root); err != nil {
		return fmt.Errorf("clear document root: %w", err)
	}
	if err := os.Rename(stagingDir, root); err != nil {
		return fmt.Errorf("promote staged documents: %w", err)
	}
	return nil
}

func restoreBackedUpDocuments(backupDir, documentRoot string) error {
	root := strings.TrimSpace(documentRoot)
	if root == "" || backupDir == "" {
		return nil
	}
	if _, err := os.Stat(backupDir); os.IsNotExist(err) {
		return nil
	}
	if err := os.RemoveAll(root); err != nil {
		return fmt.Errorf("clear document root for rollback: %w", err)
	}
	if err := os.Rename(backupDir, root); err != nil {
		return fmt.Errorf("restore backed up documents: %w", err)
	}
	return nil
}

func cleanupRestoreDocumentArtifacts(documentRoot, stagingDir, backupDir string) {
	root := strings.TrimSpace(documentRoot)
	if root == "" {
		return
	}
	if stagingDir != "" {
		_ = os.RemoveAll(stagingDir)
	}
	if backupDir != "" {
		_ = os.RemoveAll(backupDir)
	}
	_ = os.RemoveAll(siblingDocumentPath(root, documentRestoreStagingSuffix))
	_ = os.RemoveAll(siblingDocumentPath(root, documentRestoreBackupSuffix))
}

func addFileToTar(tarWriter *tar.Writer, sourcePath, tarName string) error {
	file, err := os.Open(sourcePath)
	if err != nil {
		return err
	}
	defer file.Close()

	info, err := file.Stat()
	if err != nil {
		return err
	}

	header, err := tar.FileInfoHeader(info, "")
	if err != nil {
		return err
	}
	header.Name = tarName
	if err := tarWriter.WriteHeader(header); err != nil {
		return err
	}
	_, err = io.Copy(tarWriter, file)
	return err
}

func addDirectoryToTar(tarWriter *tar.Writer, sourceDir, tarPrefix string) error {
	if _, err := os.Stat(sourceDir); os.IsNotExist(err) {
		return nil
	}

	return filepath.Walk(sourceDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(sourceDir, path)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		if rel == "." {
			return nil
		}

		tarName := filepath.ToSlash(filepath.Join(tarPrefix, rel))
		if info.IsDir() {
			header := &tar.Header{
				Name:     tarName + "/",
				Typeflag: tar.TypeDir,
				Mode:     0o755,
			}
			return tarWriter.WriteHeader(header)
		}
		return addFileToTar(tarWriter, path, tarName)
	})
}

func writeTarBytes(tarWriter *tar.Writer, name string, data []byte, mode int64) error {
	header := &tar.Header{
		Name: name,
		Mode: mode,
		Size: int64(len(data)),
	}
	if err := tarWriter.WriteHeader(header); err != nil {
		return err
	}
	_, err := tarWriter.Write(data)
	return err
}

func copyDir(sourceDir, destDir string) error {
	return filepath.Walk(sourceDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(sourceDir, path)
		if err != nil {
			return err
		}
		target := filepath.Join(destDir, rel)
		if info.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return err
		}
		return copyFile(path, target)
	})
}

func copyFile(sourcePath, destPath string) error {
	source, err := os.Open(sourcePath)
	if err != nil {
		return err
	}
	defer source.Close()

	if err := os.MkdirAll(filepath.Dir(destPath), 0o755); err != nil {
		return err
	}

	destination, err := os.Create(destPath)
	if err != nil {
		return err
	}
	defer destination.Close()

	if _, err := io.Copy(destination, source); err != nil {
		return err
	}
	return destination.Close()
}
