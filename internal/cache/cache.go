package cache

import (
	"github.com/eda-labs/eda-embeddingsearch/pkg/models"
)

// Global cache instance for backward compatibility
var globalCache = NewCacheManager()

// GetFromMemory retrieves a database from memory cache (deprecated: use CacheManager)
func GetFromMemory(path string) (*models.EmbeddingDB, bool) {
	return globalCache.GetFromMemory(path)
}

// StoreInMemory stores a database in memory cache (deprecated: use CacheManager)
func StoreInMemory(path string, db *models.EmbeddingDB) {
	globalCache.StoreInMemory(path, db)
}

// GetBinaryCachePath returns the path for the binary cache file (deprecated: use CacheManager)
func GetBinaryCachePath(jsonPath string) string {
	return globalCache.GetBinaryCachePath(jsonPath)
}

// SaveBinaryCache saves the database to a binary cache file (deprecated: use CacheManager)
func SaveBinaryCache(db *models.EmbeddingDB, cachePath string) error {
	return globalCache.SaveBinaryCache(db, cachePath)
}

// LoadBinaryCache loads the database from a binary cache file (deprecated: use CacheManager)
func LoadBinaryCache(cachePath string) (*models.EmbeddingDB, error) {
	return globalCache.LoadBinaryCache(cachePath)
}

// IsBinaryCacheValid checks if binary cache exists and is newer than JSON (deprecated: use CacheManager)
func IsBinaryCacheValid(jsonPath, cachePath string) bool {
	return globalCache.IsBinaryCacheValid(jsonPath, cachePath)
}
