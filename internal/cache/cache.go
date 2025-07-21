package cache

import (
	"github.com/eda-labs/eda-embeddingsearch/pkg/models"
)

// Global cache instance for backward compatibility
var globalCache = NewManager()

// GetFromMemory retrieves a database from memory cache (deprecated: use Manager)
func GetFromMemory(path string) (*models.EmbeddingDB, bool) {
	return globalCache.GetFromMemory(path)
}

// StoreInMemory stores a database in memory cache (deprecated: use Manager)
func StoreInMemory(path string, db *models.EmbeddingDB) {
	globalCache.StoreInMemory(path, db)
}

// GetBinaryCachePath returns the path for the binary cache file (deprecated: use Manager)
func GetBinaryCachePath(jsonPath string) string {
	return globalCache.GetBinaryCachePath(jsonPath)
}

// SaveBinaryCache saves the database to a binary cache file (deprecated: use Manager)
func SaveBinaryCache(db *models.EmbeddingDB, cachePath string) error {
	return globalCache.SaveBinaryCache(db, cachePath)
}

// LoadBinaryCache loads the database from a binary cache file (deprecated: use Manager)
func LoadBinaryCache(cachePath string) (*models.EmbeddingDB, error) {
	return globalCache.LoadBinaryCache(cachePath)
}

// IsBinaryCacheValid checks if binary cache exists and is newer than JSON (deprecated: use Manager)
func IsBinaryCacheValid(jsonPath, cachePath string) bool {
	return globalCache.IsBinaryCacheValid(jsonPath, cachePath)
}
