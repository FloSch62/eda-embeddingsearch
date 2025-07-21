// Package download handles retrieval and extraction of embedding databases from
// remote release archives.
package download

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/eda-labs/eda-embeddingsearch/internal/constants"
	"github.com/eda-labs/eda-embeddingsearch/pkg/models"
)

// Embedding URLs and filenames
const (
	srlEmbeddingURL   = "https://github.com/nokia-eda/llm-embeddings/releases/download/nokia-srl-25.3.3/llm-embeddings-srl-25-3-3.tar.gz"
	srosEmbeddingURL  = "https://github.com/nokia-eda/llm-embeddings/releases/download/nokia-sros-v25.3.r2/llm-embeddings-sros-25-3-r2.tar.gz"
	srlEmbeddingFile  = "ce-llm-embed-db-srl-25.3.3.json"
	srosEmbeddingFile = "ce-llm-embed-db-sros-25.3.r1.json"
)

// Downloader handles downloading and managing embeddings
type Downloader struct {
	embedDir     string
	srlURL       string
	srosURL      string
	srlFileName  string
	srosFileName string
}

// NewDownloader creates a new embeddings downloader
func NewDownloader() *Downloader {
	homeDir, _ := os.UserHomeDir()
	embedDir := filepath.Join(homeDir, ".eda", "vscode", "embeddings")

	return &Downloader{
		embedDir:     embedDir,
		srlURL:       srlEmbeddingURL,
		srosURL:      srosEmbeddingURL,
		srlFileName:  srlEmbeddingFile,
		srosFileName: srosEmbeddingFile,
	}
}

// GetEmbeddingPath returns the path for the specified platform
func (d *Downloader) GetEmbeddingPath(platform models.EmbeddingType) string {
	switch platform {
	case models.SROS:
		return filepath.Join(d.embedDir, d.srosFileName)
	default:
		return filepath.Join(d.embedDir, d.srlFileName)
	}
}

// EnsureEmbeddings ensures embeddings are downloaded for the specified platform
func (d *Downloader) EnsureEmbeddings(platform models.EmbeddingType) (string, error) {
	// Create embeddings directory
	if err := os.MkdirAll(d.embedDir, constants.DirPermissions); err != nil {
		return "", fmt.Errorf("failed to create embeddings directory: %v", err)
	}

	path := d.GetEmbeddingPath(platform)

	// Check if embeddings already exist
	if _, err := os.Stat(path); err == nil {
		return path, nil
	}

	// Download embeddings
	if err := d.downloadEmbeddings(platform); err != nil {
		return "", err
	}

	return path, nil
}

// DetectPlatformFromQuery detects platform based on query content
// This is only used when platform is not explicitly specified
func DetectPlatformFromQuery(query string) models.EmbeddingType {
	queryLower := strings.ToLower(query)

	// Check for SROS-specific keywords
	srosKeywords := []string{"sros", "sr os", "service router", "7750", "7450", "7250", "7950"}
	for _, keyword := range srosKeywords {
		if strings.Contains(queryLower, keyword) {
			return models.SROS
		}
	}

	// Default to SRL
	return models.SRL
}

func (d *Downloader) downloadEmbeddings(platform models.EmbeddingType) error {
	url, expectedFile := d.getURLAndFile(platform)

	// Download the tar.gz file
	resp, err := http.Get(url)
	if err != nil {
		return fmt.Errorf("failed to download embeddings: %v", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to download embeddings: HTTP %d", resp.StatusCode)
	}

	// Extract the tar.gz archive
	if err := d.extractTarGz(resp.Body); err != nil {
		return err
	}
	// Embeddings extracted successfully
	// Verify the expected file exists
	expectedPath := filepath.Join(d.embedDir, expectedFile)
	if _, err := os.Stat(expectedPath); err != nil {
		return fmt.Errorf("expected embedding file not found after extraction: %s", expectedPath)
	}

	return nil
}

func (d *Downloader) getURLAndFile(platform models.EmbeddingType) (url, file string) {
	switch platform {
	case models.SROS:
		return d.srosURL, d.srosFileName
	default:
		return d.srlURL, d.srlFileName
	}
}

func (d *Downloader) extractTarGz(r io.Reader) error {
	gzr, err := gzip.NewReader(r)
	if err != nil {
		return fmt.Errorf("failed to create gzip reader: %v", err)
	}
	defer func() {
		_ = gzr.Close()
	}()

	tr := tar.NewReader(gzr)

	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("tar reading error: %v", err)
		}

		target := filepath.Join(d.embedDir, header.Name)

		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, constants.DirPermissions); err != nil {
				return fmt.Errorf("failed to create directory: %v", err)
			}
		case tar.TypeReg:
			outFile, err := os.Create(target)
			if err != nil {
				return fmt.Errorf("failed to create file: %v", err)
			}
			if _, err := io.Copy(outFile, tr); err != nil {
				_ = outFile.Close()
				return fmt.Errorf("failed to write file: %v", err)
			}
			_ = outFile.Close()
		}
	}

	return nil
}
