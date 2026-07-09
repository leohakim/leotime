package billing

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

type StoredDocument struct {
	RelativePath string
	SHA256       string
	ByteSize     int64
	MIMEType     string
}

type DocumentStore struct {
	root string
}

func NewDocumentStore(root string) (*DocumentStore, error) {
	cleanRoot := strings.TrimSpace(root)
	if cleanRoot == "" {
		return nil, fmt.Errorf("document root is required")
	}
	absRoot, err := filepath.Abs(cleanRoot)
	if err != nil {
		return nil, fmt.Errorf("resolve document root: %w", err)
	}
	if err := os.MkdirAll(absRoot, 0o755); err != nil {
		return nil, fmt.Errorf("create document root: %w", err)
	}
	return &DocumentStore{root: absRoot}, nil
}

func (s *DocumentStore) Root() string {
	return s.root
}

func (s *DocumentStore) WriteOfficial(ctx context.Context, relativePath string, sourcePath string) (StoredDocument, error) {
	_ = ctx
	if err := validateDocumentRelativePath(relativePath); err != nil {
		return StoredDocument{}, err
	}

	targetPath, err := s.resolvePath(relativePath)
	if err != nil {
		return StoredDocument{}, err
	}
	if err := os.MkdirAll(filepath.Dir(targetPath), 0o755); err != nil {
		return StoredDocument{}, fmt.Errorf("create document directory: %w", err)
	}

	source, err := os.Open(sourcePath)
	if err != nil {
		return StoredDocument{}, fmt.Errorf("open source document: %w", err)
	}
	defer source.Close()

	tempPath := targetPath + ".tmp"
	tempFile, err := os.OpenFile(tempPath, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o644)
	if err != nil {
		return StoredDocument{}, fmt.Errorf("create temp document: %w", err)
	}

	hasher := sha256.New()
	writer := io.MultiWriter(tempFile, hasher)
	size, err := io.Copy(writer, source)
	if err != nil {
		tempFile.Close()
		os.Remove(tempPath)
		return StoredDocument{}, fmt.Errorf("write temp document: %w", err)
	}
	if err := tempFile.Sync(); err != nil {
		tempFile.Close()
		os.Remove(tempPath)
		return StoredDocument{}, fmt.Errorf("sync temp document: %w", err)
	}
	if err := tempFile.Close(); err != nil {
		os.Remove(tempPath)
		return StoredDocument{}, fmt.Errorf("close temp document: %w", err)
	}

	header := make([]byte, 4)
	file, err := os.Open(tempPath)
	if err != nil {
		os.Remove(tempPath)
		return StoredDocument{}, fmt.Errorf("verify temp document: %w", err)
	}
	if _, err := io.ReadFull(file, header); err != nil {
		file.Close()
		os.Remove(tempPath)
		return StoredDocument{}, fmt.Errorf("read temp document header: %w", err)
	}
	file.Close()
	if string(header) != "%PDF" {
		os.Remove(tempPath)
		return StoredDocument{}, fmt.Errorf("document is not a PDF")
	}

	if err := os.Rename(tempPath, targetPath); err != nil {
		os.Remove(tempPath)
		return StoredDocument{}, fmt.Errorf("finalize document: %w", err)
	}

	return StoredDocument{
		RelativePath: filepath.ToSlash(relativePath),
		SHA256:       hex.EncodeToString(hasher.Sum(nil)),
		ByteSize:     size,
		MIMEType:     "application/pdf",
	}, nil
}

func (s *DocumentStore) Open(relativePath string) (*os.File, StoredDocument, error) {
	if err := validateDocumentRelativePath(relativePath); err != nil {
		return nil, StoredDocument{}, err
	}
	targetPath, err := s.resolvePath(relativePath)
	if err != nil {
		return nil, StoredDocument{}, err
	}

	file, err := os.Open(targetPath)
	if err != nil {
		return nil, StoredDocument{}, fmt.Errorf("open document: %w", err)
	}
	info, err := file.Stat()
	if err != nil {
		file.Close()
		return nil, StoredDocument{}, fmt.Errorf("stat document: %w", err)
	}

	hasher := sha256.New()
	if _, err := io.Copy(hasher, file); err != nil {
		file.Close()
		return nil, StoredDocument{}, fmt.Errorf("hash document: %w", err)
	}
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		file.Close()
		return nil, StoredDocument{}, fmt.Errorf("rewind document: %w", err)
	}

	return file, StoredDocument{
		RelativePath: filepath.ToSlash(relativePath),
		SHA256:       hex.EncodeToString(hasher.Sum(nil)),
		ByteSize:     info.Size(),
		MIMEType:     "application/pdf",
	}, nil
}

func (s *DocumentStore) resolvePath(relativePath string) (string, error) {
	cleanRelative := filepath.ToSlash(strings.TrimPrefix(filepath.Clean(relativePath), string(filepath.Separator)))
	fullPath, err := filepath.Abs(filepath.Join(s.root, filepath.FromSlash(cleanRelative)))
	if err != nil {
		return "", fmt.Errorf("resolve document path: %w", err)
	}
	rel, err := filepath.Rel(s.root, fullPath)
	if err != nil || strings.HasPrefix(rel, "..") {
		return "", fmt.Errorf("document path escapes root")
	}
	return fullPath, nil
}

func validateDocumentRelativePath(relativePath string) error {
	clean := strings.TrimSpace(strings.ReplaceAll(relativePath, "\\", "/"))
	if clean == "" {
		return fmt.Errorf("document path is required")
	}
	if strings.Contains(clean, "..") || filepath.IsAbs(clean) {
		return fmt.Errorf("document path is invalid")
	}
	if !strings.HasSuffix(strings.ToLower(clean), ".pdf") {
		return fmt.Errorf("document path must end with .pdf")
	}
	return nil
}

func DocumentRelativePath(year int, seriesCode, invoiceNumber, fileName string) string {
	safeSeries := safePathSegment(seriesCode)
	safeNumber := safePathSegment(invoiceNumber)
	return fmt.Sprintf("invoices/%d/%s/%s/%s", year, safeSeries, safeNumber, fileName)
}

func safePathSegment(value string) string {
	replacer := strings.NewReplacer("/", "-", "\\", "-", "..", "")
	return replacer.Replace(strings.TrimSpace(value))
}
