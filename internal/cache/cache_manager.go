// Package cache provides utilities to load and store embedding databases
// using an on-disk binary cache and an in-memory cache.
package cache

import (
	"encoding/gob"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/eda-labs/eda-embeddingsearch/pkg/models"
)

// CacheManager interface defines cache operations
type CacheManager interface {
	GetFromMemory(path string) (*models.EmbeddingDB, bool)
	StoreInMemory(path string, db *models.EmbeddingDB)
	GetBinaryCachePath(jsonPath string) string
	SaveBinaryCache(db *models.EmbeddingDB, cachePath string) error
	LoadBinaryCache(cachePath string) (*models.EmbeddingDB, error)
	IsBinaryCacheValid(jsonPath, cachePath string) bool
}

// DefaultCacheManager implements the CacheManager interface
type DefaultCacheManager struct {
	dbCache    map[string]*models.EmbeddingDB
	cacheMutex sync.RWMutex
}

// NewCacheManager creates a new cache manager
func NewCacheManager() CacheManager {
	return &DefaultCacheManager{
		dbCache: make(map[string]*models.EmbeddingDB),
	}
}

// GetFromMemory retrieves a database from memory cache
func (m *DefaultCacheManager) GetFromMemory(path string) (*models.EmbeddingDB, bool) {
	m.cacheMutex.RLock()
	defer m.cacheMutex.RUnlock()
	db, exists := m.dbCache[path]
	return db, exists
}

// StoreInMemory stores a database in memory cache
func (m *DefaultCacheManager) StoreInMemory(path string, db *models.EmbeddingDB) {
	m.cacheMutex.Lock()
	defer m.cacheMutex.Unlock()
	m.dbCache[path] = db
}

// GetBinaryCachePath returns the path for the binary cache file
func (m *DefaultCacheManager) GetBinaryCachePath(jsonPath string) string {
	dir := filepath.Dir(jsonPath)
	base := filepath.Base(jsonPath)
	return filepath.Join(dir, "."+base+".cache")
}

// SaveBinaryCache saves the database to a binary cache file
func (m *DefaultCacheManager) SaveBinaryCache(db *models.EmbeddingDB, cachePath string) error {
	file, err := os.Create(cachePath)
	if err != nil {
		return fmt.Errorf("failed to create cache file %s: %w", cachePath, err)
	}

	enc := gob.NewEncoder(file)
	if err = enc.Encode(db); err != nil {
		_ = file.Close()
		return fmt.Errorf("failed to encode cache data: %w", err)
	}
	if cerr := file.Close(); cerr != nil {
		return fmt.Errorf("failed to close cache file: %w", cerr)
	}
	return nil
}

// LoadBinaryCache loads the database from a binary cache file
func (m *DefaultCacheManager) LoadBinaryCache(cachePath string) (*models.EmbeddingDB, error) {
	file, err := os.Open(cachePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open cache file %s: %w", cachePath, err)
	}

	var db models.EmbeddingDB
	dec := gob.NewDecoder(file)
	if err = dec.Decode(&db); err != nil {
		_ = file.Close()
		return nil, fmt.Errorf("failed to decode cache data from %s: %w", cachePath, err)
	}
	if cerr := file.Close(); cerr != nil {
		return nil, fmt.Errorf("failed to close cache file: %w", cerr)
	}

	return &db, nil
}

// IsBinaryCacheValid checks if binary cache exists and is newer than JSON
func (m *DefaultCacheManager) IsBinaryCacheValid(jsonPath, cachePath string) bool {
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
