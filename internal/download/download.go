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

	"github.com/eda-labs/eda-embeddingsearch/pkg/models"
)

const (
	srlEmbeddingURL   = "https://github.com/nokia-eda/llm-embeddings/releases/download/nokia-srl-25.3.3/llm-embeddings-srl-25-3-3.tar.gz"
	srosEmbeddingURL  = "https://github.com/nokia-eda/llm-embeddings/releases/download/nokia-sros-v25.3.r2/llm-embeddings-sros-25-3-r2.tar.gz"
	srlEmbeddingFile  = "ce-llm-embed-db-srl-25.3.3.json"
	srosEmbeddingFile = "ce-llm-embed-db-sros-25.3.r1.json"
)

// GetEmbeddingsPath returns the path to the embeddings directory
func GetEmbeddingsPath() string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "~/.eda/vscode/embeddings"
	}
	return filepath.Join(homeDir, ".eda", "vscode", "embeddings")
}

// GetEmbeddingPaths returns the paths for both SRL and SROS embeddings
func GetEmbeddingPaths() (string, string) {
	embeddingsDir := GetEmbeddingsPath()
	srlPath := filepath.Join(embeddingsDir, srlEmbeddingFile)
	srosPath := filepath.Join(embeddingsDir, srosEmbeddingFile)
	return srlPath, srosPath
}

// DetectEmbeddingType determines which embedding set to use based on query content
func DetectEmbeddingType(query string) models.EmbeddingType {
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

// DownloadEmbeddings downloads and extracts a specific embedding set
func DownloadEmbeddings(embType models.EmbeddingType, embeddingsDir string, verbose bool) (err error) {
	var url, expectedFile string

	switch embType {
	case models.SRL:
		url = srlEmbeddingURL
		expectedFile = srlEmbeddingFile
		if verbose {
			fmt.Println("Downloading SRL embeddings from GitHub...")
		}
	case models.SROS:
		url = srosEmbeddingURL
		expectedFile = srosEmbeddingFile
		if verbose {
			fmt.Println("Downloading SROS embeddings from GitHub...")
		}
	}

	// Download the tar.gz file
	resp, err := http.Get(url)
	if err != nil {
		return fmt.Errorf("failed to download embeddings: %v", err)
	}
	defer func() {
		if cerr := resp.Body.Close(); err == nil {
			err = cerr
		}
	}()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to download embeddings: HTTP %d", resp.StatusCode)
	}

	if verbose {
		fmt.Println("Extracting embeddings...")
	}

	// Create gzip reader
	gzipReader, err := gzip.NewReader(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to create gzip reader: %v", err)
	}
	defer func() {
		if cerr := gzipReader.Close(); err == nil {
			err = cerr
		}
	}()

	// Create tar reader
	tarReader := tar.NewReader(gzipReader)

	// Extract files
	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("failed to read tar entry: %v", err)
		}

		// Skip directories
		if header.Typeflag == tar.TypeDir {
			continue
		}

		// Create the file path
		filePath := filepath.Join(embeddingsDir, header.Name)

		// Create directory if needed
		if err := os.MkdirAll(filepath.Dir(filePath), 0755); err != nil {
			return fmt.Errorf("failed to create directory: %v", err)
		}

		// Create the file
		var file *os.File
		file, err = os.Create(filePath)
		if err != nil {
			return fmt.Errorf("failed to create file %s: %v", filePath, err)
		}

		// Copy file contents
		if _, err := io.Copy(file, tarReader); err != nil {
			_ = file.Close()
			return fmt.Errorf("failed to write file %s: %v", filePath, err)
		}
		_ = file.Close()
	}

	if verbose {
		fmt.Println("Embeddings extracted successfully!")
	}

	// Verify the expected file exists
	expectedPath := filepath.Join(embeddingsDir, expectedFile)
	if _, err := os.Stat(expectedPath); err != nil {
		return fmt.Errorf("expected embedding file not found after extraction: %s", expectedPath)
	}

	return err
}

// DownloadAndExtractEmbeddings downloads and extracts the embedding files if they don't exist
func DownloadAndExtractEmbeddings(query string, verbose bool) (string, error) {
	embeddingsDir := GetEmbeddingsPath()
	srlPath, srosPath := GetEmbeddingPaths()

	// Create embeddings directory
	if err := os.MkdirAll(embeddingsDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create embeddings directory: %v", err)
	}

	// Determine which embedding type to use
	embType := DetectEmbeddingType(query)

	var targetPath string
	switch embType {
	case models.SRL:
		targetPath = srlPath
		// Check if SRL embeddings exist, download if not
		if _, err := os.Stat(srlPath); err != nil {
			if err := DownloadEmbeddings(models.SRL, embeddingsDir, verbose); err != nil {
				return "", err
			}
		}
	case models.SROS:
		targetPath = srosPath
		// Check if SROS embeddings exist, download if not
		if _, err := os.Stat(srosPath); err != nil {
			if err := DownloadEmbeddings(models.SROS, embeddingsDir, verbose); err != nil {
				return "", err
			}
		}
	}

	return targetPath, nil
}
