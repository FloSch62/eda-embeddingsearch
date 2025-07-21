package embedding

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/eda-labs/eda-embeddingsearch/internal/cache"
	"github.com/eda-labs/eda-embeddingsearch/pkg/models"
)

// Loader handles loading of embedding databases
type Loader struct {
	cacheManager cache.CacheManager
}

// NewLoader creates a new loader with the specified cache manager
func NewLoader(cacheManager cache.CacheManager) *Loader {
	return &Loader{
		cacheManager: cacheManager,
	}
}

// Load loads an embedding database from disk with caching
func (l *Loader) Load(path string) (*models.EmbeddingDB, error) {
	// Check memory cache first
	if db := l.loadFromMemoryCache(path); db != nil {
		return db, nil
	}

	// Check binary cache
	cachePath := l.cacheManager.GetBinaryCachePath(path)
	if db := l.loadFromBinaryCache(path, cachePath); db != nil {
		return db, nil
	}

	// Load from JSON file
	return l.loadFromJSON(path, cachePath)
}

func (l *Loader) loadFromMemoryCache(path string) *models.EmbeddingDB {
	if cached, exists := l.cacheManager.GetFromMemory(path); exists {
		return cached
	}
	return nil
}

func (l *Loader) loadFromBinaryCache(path, cachePath string) *models.EmbeddingDB {
	if !l.cacheManager.IsBinaryCacheValid(path, cachePath) {
		return nil
	}

	db, err := l.cacheManager.LoadBinaryCache(cachePath)
	if err == nil {
		l.cacheManager.StoreInMemory(path, db)
		return db
	}

	return nil
}

func (l *Loader) loadFromJSON(path, cachePath string) (*models.EmbeddingDB, error) {

	// Load JSON data
	db, err := l.loadJSONFile(path)
	if err != nil {
		return nil, err
	}

	// Build index and save cache
	l.postProcessDatabase(db, cachePath)

	// Cache in memory
	l.cacheManager.StoreInMemory(path, db)

	return db, nil
}

func (l *Loader) loadJSONFile(path string) (*models.EmbeddingDB, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open embedding file %s: %w", path, err)
	}
	defer func() {
		_ = file.Close()
	}()

	var db models.EmbeddingDB
	dec := json.NewDecoder(file)
	if err := dec.Decode(&db); err != nil {
		return nil, fmt.Errorf("failed to decode embedding JSON from %s: %w", path, err)
	}

	return &db, nil
}

func (l *Loader) postProcessDatabase(db *models.EmbeddingDB, cachePath string) {
	// Build inverted index
	BuildInvertedIndex(db)

	// Save binary cache
	l.saveBinaryCache(db, cachePath)
}

func (l *Loader) saveBinaryCache(db *models.EmbeddingDB, cachePath string) {
	_ = l.cacheManager.SaveBinaryCache(db, cachePath)
}
