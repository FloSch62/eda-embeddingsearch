package cache

import (
	"encoding/gob"
	"os"
	"path/filepath"
	"sync"

	"github.com/eda-labs/eda-embeddingsearch/pkg/models"
)

// Manager interface defines cache operations
type Manager interface {
	GetFromMemory(path string) (*models.EmbeddingDB, bool)
	StoreInMemory(path string, db *models.EmbeddingDB)
	GetBinaryCachePath(jsonPath string) string
	SaveBinaryCache(db *models.EmbeddingDB, cachePath string) error
	LoadBinaryCache(cachePath string) (*models.EmbeddingDB, error)
	IsBinaryCacheValid(jsonPath, cachePath string) bool
}

// DefaultManager implements the Manager interface
type DefaultManager struct {
	dbCache    map[string]*models.EmbeddingDB
	cacheMutex sync.RWMutex
}

// NewManager creates a new cache manager
func NewManager() Manager {
	return &DefaultManager{
		dbCache: make(map[string]*models.EmbeddingDB),
	}
}

// GetFromMemory retrieves a database from memory cache
func (m *DefaultManager) GetFromMemory(path string) (*models.EmbeddingDB, bool) {
	m.cacheMutex.RLock()
	defer m.cacheMutex.RUnlock()
	db, exists := m.dbCache[path]
	return db, exists
}

// StoreInMemory stores a database in memory cache
func (m *DefaultManager) StoreInMemory(path string, db *models.EmbeddingDB) {
	m.cacheMutex.Lock()
	defer m.cacheMutex.Unlock()
	m.dbCache[path] = db
}

// GetBinaryCachePath returns the path for the binary cache file
func (m *DefaultManager) GetBinaryCachePath(jsonPath string) string {
	dir := filepath.Dir(jsonPath)
	base := filepath.Base(jsonPath)
	return filepath.Join(dir, "."+base+".cache")
}

// SaveBinaryCache saves the database to a binary cache file
func (m *DefaultManager) SaveBinaryCache(db *models.EmbeddingDB, cachePath string) error {
	file, err := os.Create(cachePath)
	if err != nil {
		return err
	}

	enc := gob.NewEncoder(file)
	if err = enc.Encode(db); err != nil {
		_ = file.Close()
		return err
	}
	if cerr := file.Close(); cerr != nil {
		return cerr
	}
	return nil
}

// LoadBinaryCache loads the database from a binary cache file
func (m *DefaultManager) LoadBinaryCache(cachePath string) (*models.EmbeddingDB, error) {
	file, err := os.Open(cachePath)
	if err != nil {
		return nil, err
	}

	var db models.EmbeddingDB
	dec := gob.NewDecoder(file)
	if err = dec.Decode(&db); err != nil {
		_ = file.Close()
		return nil, err
	}
	if cerr := file.Close(); cerr != nil {
		return nil, cerr
	}

	return &db, nil
}

// IsBinaryCacheValid checks if binary cache exists and is newer than JSON
func (m *DefaultManager) IsBinaryCacheValid(jsonPath, cachePath string) bool {
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
