package download

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"

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
func GetEmbeddingPaths() (srlPath, srosPath string) {
	embeddingsDir := GetEmbeddingsPath()
	srlPath = filepath.Join(embeddingsDir, srlEmbeddingFile)
	srosPath = filepath.Join(embeddingsDir, srosEmbeddingFile)
	return srlPath, srosPath
}

// DetectEmbeddingType determines which embedding set to use based on query content
// Deprecated: Use DetectPlatformFromQuery instead
func DetectEmbeddingType(query string) models.EmbeddingType {
	return DetectPlatformFromQuery(query)
}

// DownloadEmbeddings downloads and extracts a specific embedding set
func DownloadEmbeddings(embType models.EmbeddingType, embeddingsDir string, verbose bool) error {
	url, expectedFile := getEmbeddingURLAndFile(embType, verbose)

	// Download the tar.gz file
	resp, err := downloadFile(url)
	if err != nil {
		return err
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if verbose {
		fmt.Println("Extracting embeddings...")
	}

	// Extract the tar.gz archive
	if err := extractTarGz(resp.Body, embeddingsDir); err != nil {
		return err
	}

	if verbose {
		fmt.Println("Embeddings extracted successfully!")
	}

	// Verify the expected file exists
	return verifyExtractedFile(embeddingsDir, expectedFile)
}

func getEmbeddingURLAndFile(embType models.EmbeddingType, verbose bool) (url, file string) {
	switch embType {
	case models.SRL:
		if verbose {
			fmt.Println("Downloading SRL embeddings from GitHub...")
		}
		return srlEmbeddingURL, srlEmbeddingFile
	case models.SROS:
		if verbose {
			fmt.Println("Downloading SROS embeddings from GitHub...")
		}
		return srosEmbeddingURL, srosEmbeddingFile
	default:
		return "", ""
	}
}

func downloadFile(url string) (*http.Response, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to download embeddings: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		_ = resp.Body.Close()
		return nil, fmt.Errorf("failed to download embeddings: HTTP %d", resp.StatusCode)
	}

	return resp, nil
}

func extractTarGz(reader io.Reader, destDir string) error {
	// Create gzip reader
	gzipReader, err := gzip.NewReader(reader)
	if err != nil {
		return fmt.Errorf("failed to create gzip reader: %v", err)
	}
	defer func() {
		_ = gzipReader.Close()
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

		if err := extractTarEntry(tarReader, header, destDir); err != nil {
			return err
		}
	}

	return nil
}

func extractTarEntry(tarReader *tar.Reader, header *tar.Header, destDir string) error {
	// Skip directories
	if header.Typeflag == tar.TypeDir {
		return nil
	}

	// Create the file path
	filePath := filepath.Join(destDir, header.Name)

	// Create directory if needed
	if err := os.MkdirAll(filepath.Dir(filePath), 0o755); err != nil {
		return fmt.Errorf("failed to create directory: %v", err)
	}

	// Create and write the file
	file, err := os.Create(filePath)
	if err != nil {
		return fmt.Errorf("failed to create file %s: %v", filePath, err)
	}
	defer func() {
		_ = file.Close()
	}()

	// Copy file contents
	if _, err := io.Copy(file, tarReader); err != nil {
		return fmt.Errorf("failed to write file %s: %v", filePath, err)
	}

	return nil
}

func verifyExtractedFile(embeddingsDir, expectedFile string) error {
	expectedPath := filepath.Join(embeddingsDir, expectedFile)
	if _, err := os.Stat(expectedPath); err != nil {
		return fmt.Errorf("expected embedding file not found after extraction: %s", expectedPath)
	}
	return nil
}

// DownloadAndExtractEmbeddings downloads and extracts the embedding files if they don't exist
// Deprecated: Use Manager.EnsureEmbeddings instead
func DownloadAndExtractEmbeddings(query string, verbose bool) (string, error) {
	manager := NewManager()
	platform := DetectPlatformFromQuery(query)
	return manager.EnsureEmbeddings(platform, verbose)
}
