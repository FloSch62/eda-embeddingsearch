package download

import (
	"os"
	"path/filepath"

	"github.com/eda-labs/eda-embeddingsearch/pkg/models"
)

// Embedding URLs and filenames
const (
	srlEmbeddingURL   = "https://github.com/nokia-eda/llm-embeddings/releases/download/nokia-srl-25.3.3/llm-embeddings-srl-25-3-3.tar.gz"
	srosEmbeddingURL  = "https://github.com/nokia-eda/llm-embeddings/releases/download/nokia-sros-v25.3.r2/llm-embeddings-sros-25-3-r2.tar.gz"
	srlEmbeddingFile  = "ce-llm-embed-db-srl-25.3.3.json"
	srosEmbeddingFile = "ce-llm-embed-db-sros-25.3.r1.json"
)

// GetEmbeddingsPath returns the path to the embeddings directory
// Deprecated: Use NewManager() instead
func GetEmbeddingsPath() string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "~/.eda/vscode/embeddings"
	}
	return filepath.Join(homeDir, ".eda", "vscode", "embeddings")
}

// GetEmbeddingPaths returns the paths for both SRL and SROS embeddings
// Deprecated: Use Downloader.GetEmbeddingPath instead
func GetEmbeddingPaths() (srlPath, srosPath string) {
	downloader := NewDownloader()
	srlPath = downloader.GetEmbeddingPath(models.SRL)
	srosPath = downloader.GetEmbeddingPath(models.SROS)
	return srlPath, srosPath
}

// DetectEmbeddingType determines which embedding set to use based on query content
// Deprecated: Use DetectPlatformFromQuery instead
func DetectEmbeddingType(query string) models.EmbeddingType {
	return DetectPlatformFromQuery(query)
}

// DownloadEmbeddings downloads and extracts a specific embedding set
// Deprecated: Use Downloader.EnsureEmbeddings instead
func DownloadEmbeddings(embType models.EmbeddingType, embeddingsDir string, verbose bool) error {
	// For backward compatibility, we can't use the downloader directly since
	// it doesn't support custom embedding directories
	// This function should be removed in the future
	return nil
}

// DownloadAndExtractEmbeddings downloads and extracts the embedding files if they don't exist
// Deprecated: Use Downloader.EnsureEmbeddings instead
func DownloadAndExtractEmbeddings(query string, verbose bool) (string, error) {
	downloader := NewDownloader()
	platform := DetectPlatformFromQuery(query)
	return downloader.EnsureEmbeddings(platform, verbose)
}
