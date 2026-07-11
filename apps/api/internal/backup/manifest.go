package backup

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const manifestFileName = "manifest.json"
const archiveDBFileName = "leotime.db"
const archiveDocumentsDir = "documents"

type ManifestDocument struct {
	Path     string `json:"path"`
	SHA256   string `json:"sha256"`
	ByteSize int64  `json:"byteSize"`
}

type Manifest struct {
	GeneratedAt    string             `json:"generatedAt"`
	DatabaseSHA256 string             `json:"databaseSha256"`
	Documents      []ManifestDocument `json:"documents"`
}

func hashFile(path string) (string, int64, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", 0, err
	}
	defer file.Close()

	info, err := file.Stat()
	if err != nil {
		return "", 0, err
	}

	hasher := sha256.New()
	if _, err := io.Copy(hasher, file); err != nil {
		return "", 0, err
	}

	return hex.EncodeToString(hasher.Sum(nil)), info.Size(), nil
}

func BuildManifest(dbPath, documentRoot string) (*Manifest, error) {
	dbHash, _, err := hashFile(dbPath)
	if err != nil {
		return nil, fmt.Errorf("hash database: %w", err)
	}

	manifest := &Manifest{
		GeneratedAt:    time.Now().UTC().Format(time.RFC3339),
		DatabaseSHA256: dbHash,
		Documents:      nil,
	}

	root := strings.TrimSpace(documentRoot)
	if root == "" {
		return manifest, nil
	}

	if _, err := os.Stat(root); os.IsNotExist(err) {
		return manifest, nil
	} else if err != nil {
		return nil, fmt.Errorf("stat document root: %w", err)
	}

	err = filepath.Walk(root, func(path string, info os.FileInfo, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if info.IsDir() {
			return nil
		}

		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		if strings.Contains(rel, "..") {
			return fmt.Errorf("invalid document path: %s", rel)
		}

		hash, size, err := hashFile(path)
		if err != nil {
			return err
		}
		if size < 0 {
			return fmt.Errorf("negative byte size for %s", rel)
		}

		manifest.Documents = append(manifest.Documents, ManifestDocument{
			Path:     rel,
			SHA256:   hash,
			ByteSize: size,
		})
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("walk documents: %w", err)
	}

	return manifest, nil
}

func ValidateManifest(manifest *Manifest, extractDir string) error {
	if manifest == nil {
		return fmt.Errorf("manifest is required")
	}
	if strings.TrimSpace(manifest.DatabaseSHA256) == "" {
		return fmt.Errorf("database sha256 is required")
	}

	dbPath := filepath.Join(extractDir, archiveDBFileName)
	hash, _, err := hashFile(dbPath)
	if err != nil {
		return fmt.Errorf("hash archive database: %w", err)
	}
	if hash != manifest.DatabaseSHA256 {
		return fmt.Errorf("database hash mismatch")
	}

	return ValidateDocumentManifest(manifest, filepath.Join(extractDir, archiveDocumentsDir))
}

func ValidateDocumentManifest(manifest *Manifest, documentRoot string) error {
	if manifest == nil {
		return fmt.Errorf("manifest is required")
	}

	for _, doc := range manifest.Documents {
		if doc.ByteSize < 0 {
			return fmt.Errorf("negative byte size for %s", doc.Path)
		}
		if strings.Contains(doc.Path, "..") || filepath.IsAbs(doc.Path) {
			return fmt.Errorf("invalid document path: %s", doc.Path)
		}

		docPath := filepath.Join(documentRoot, filepath.FromSlash(doc.Path))
		hash, size, err := hashFile(docPath)
		if err != nil {
			return fmt.Errorf("missing document %s: %w", doc.Path, err)
		}
		if hash != doc.SHA256 {
			return fmt.Errorf("document hash mismatch for %s", doc.Path)
		}
		if size != doc.ByteSize {
			return fmt.Errorf("document size mismatch for %s", doc.Path)
		}
	}

	return nil
}

func WriteManifest(path string, manifest *Manifest) error {
	data, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal manifest: %w", err)
	}
	return os.WriteFile(path, data, 0o644)
}

func ReadManifest(path string) (*Manifest, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var manifest Manifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		return nil, fmt.Errorf("parse manifest: %w", err)
	}
	return &manifest, nil
}
