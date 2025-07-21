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

// Manager handles downloading and managing embeddings
type Manager struct {
	embedDir     string
	srlURL       string
	srosURL      string
	srlFileName  string
	srosFileName string
}

// NewManager creates a new download manager
func NewManager() *Manager {
	homeDir, _ := os.UserHomeDir()
	embedDir := filepath.Join(homeDir, ".eda", "vscode", "embeddings")

	return &Manager{
		embedDir:     embedDir,
		srlURL:       "https://github.com/nokia-eda/llm-embeddings/releases/download/nokia-srl-25.3.3/llm-embeddings-srl-25-3-3.tar.gz",
		srosURL:      "https://github.com/nokia-eda/llm-embeddings/releases/download/nokia-sros-v25.3.r2/llm-embeddings-sros-25-3-r2.tar.gz",
		srlFileName:  "ce-llm-embed-db-srl-25.3.3.json",
		srosFileName: "ce-llm-embed-db-sros-25.3.r1.json",
	}
}

// GetEmbeddingPath returns the path for the specified platform
func (m *Manager) GetEmbeddingPath(platform models.EmbeddingType) string {
	switch platform {
	case models.SROS:
		return filepath.Join(m.embedDir, m.srosFileName)
	default:
		return filepath.Join(m.embedDir, m.srlFileName)
	}
}

// EnsureEmbeddings ensures embeddings are downloaded for the specified platform
func (m *Manager) EnsureEmbeddings(platform models.EmbeddingType, verbose bool) (string, error) {
	// Create embeddings directory
	if err := os.MkdirAll(m.embedDir, constants.DirPermissions); err != nil {
		return "", fmt.Errorf("failed to create embeddings directory: %v", err)
	}

	path := m.GetEmbeddingPath(platform)

	// Check if embeddings already exist
	if _, err := os.Stat(path); err == nil {
		return path, nil
	}

	// Download embeddings
	if err := m.downloadEmbeddings(platform, verbose); err != nil {
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

func (m *Manager) downloadEmbeddings(platform models.EmbeddingType, verbose bool) error {
	url, expectedFile := m.getURLAndFile(platform)

	if verbose {
		platformName := "SRL"
		if platform == models.SROS {
			platformName = "SROS"
		}
		fmt.Printf("Downloading %s embeddings from GitHub...\n", platformName)
	}

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

	if verbose {
		fmt.Println("Extracting embeddings...")
	}

	// Extract the tar.gz archive
	if err := m.extractTarGz(resp.Body); err != nil {
		return err
	}

	if verbose {
		fmt.Println("Embeddings extracted successfully!")
	}

	// Verify the expected file exists
	expectedPath := filepath.Join(m.embedDir, expectedFile)
	if _, err := os.Stat(expectedPath); err != nil {
		return fmt.Errorf("expected embedding file not found after extraction: %s", expectedPath)
	}

	return nil
}

func (m *Manager) getURLAndFile(platform models.EmbeddingType) (url, file string) {
	switch platform {
	case models.SROS:
		return m.srosURL, m.srosFileName
	default:
		return m.srlURL, m.srlFileName
	}
}

func (m *Manager) extractTarGz(r io.Reader) error {
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

		target := filepath.Join(m.embedDir, header.Name)

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
