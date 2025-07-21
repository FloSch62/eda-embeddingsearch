package cache

import (
	"encoding/gob"
	"os"
	"path/filepath"
	"sync"

	"github.com/eda-labs/eda-embeddingsearch/pkg/models"
)

// Global cache for loaded databases
var (
	dbCache    = make(map[string]*models.EmbeddingDB)
	cacheMutex sync.RWMutex
)

// GetFromMemory retrieves a database from memory cache
func GetFromMemory(path string) (*models.EmbeddingDB, bool) {
	cacheMutex.RLock()
	defer cacheMutex.RUnlock()
	db, exists := dbCache[path]
	return db, exists
}

// StoreInMemory stores a database in memory cache
func StoreInMemory(path string, db *models.EmbeddingDB) {
	cacheMutex.Lock()
	defer cacheMutex.Unlock()
	dbCache[path] = db
}

// GetBinaryCachePath returns the path for the binary cache file
func GetBinaryCachePath(jsonPath string) string {
	dir := filepath.Dir(jsonPath)
	base := filepath.Base(jsonPath)
	return filepath.Join(dir, "."+base+".cache")
}

// SaveBinaryCache saves the database to a binary cache file
func SaveBinaryCache(db *models.EmbeddingDB, cachePath string) error {
	file, err := os.Create(cachePath)
	if err != nil {
		return err
	}
	defer file.Close()

	enc := gob.NewEncoder(file)
	return enc.Encode(db)
}

// LoadBinaryCache loads the database from a binary cache file
func LoadBinaryCache(cachePath string) (*models.EmbeddingDB, error) {
	file, err := os.Open(cachePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var db models.EmbeddingDB
	dec := gob.NewDecoder(file)
	if err := dec.Decode(&db); err != nil {
		return nil, err
	}

	return &db, nil
}

// IsBinaryCacheValid checks if binary cache exists and is newer than JSON
func IsBinaryCacheValid(jsonPath, cachePath string) bool {
	jsonInfo, err := os.Stat(jsonPath)
	if err != nil {
		return false
	}

	cacheInfo, err := os.Stat(cachePath)
	if err != nil {
		return false
	}

	return cacheInfo.ModTime().After(jsonInfo.ModTime())
}