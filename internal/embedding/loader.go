package embedding

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

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
func (l *Loader) Load(path string, verbose bool) (*models.EmbeddingDB, error) {
	// Check memory cache first
	if db := l.loadFromMemoryCache(path, verbose); db != nil {
		return db, nil
	}

	// Check binary cache
	cachePath := l.cacheManager.GetBinaryCachePath(path)
	if db := l.loadFromBinaryCache(path, cachePath, verbose); db != nil {
		return db, nil
	}

	// Load from JSON file
	return l.loadFromJSON(path, cachePath, verbose)
}

func (l *Loader) loadFromMemoryCache(path string, verbose bool) *models.EmbeddingDB {
	if cached, exists := l.cacheManager.GetFromMemory(path); exists {
		if verbose {
			fmt.Printf("Using in-memory cached embeddings\n")
		}
		return cached
	}
	return nil
}

func (l *Loader) loadFromBinaryCache(path, cachePath string, verbose bool) *models.EmbeddingDB {
	if !l.cacheManager.IsBinaryCacheValid(path, cachePath) {
		return nil
	}

	if verbose {
		fmt.Printf("Loading from binary cache...\n")
	}
	start := time.Now()

	db, err := l.cacheManager.LoadBinaryCache(cachePath)
	if err == nil {
		if verbose {
			fmt.Printf("Loaded binary cache in %.2f seconds\n", time.Since(start).Seconds())
		}
		l.cacheManager.StoreInMemory(path, db)
		return db
	}

	if verbose {
		fmt.Printf("Binary cache failed, falling back to JSON: %v\n", err)
	}
	return nil
}

func (l *Loader) loadFromJSON(path, cachePath string, verbose bool) (*models.EmbeddingDB, error) {
	if verbose {
		fmt.Printf("Loading embeddings from %s...\n", filepath.Base(path))
	}
	start := time.Now()

	// Load JSON data
	db, err := l.loadJSONFile(path)
	if err != nil {
		return nil, err
	}

	if verbose {
		fmt.Printf("JSON loaded in %.2f seconds\n", time.Since(start).Seconds())
	}

	// Build index and save cache
	l.postProcessDatabase(db, cachePath, verbose)

	// Cache in memory
	l.cacheManager.StoreInMemory(path, db)

	if verbose {
		l.printLoadStats(db, start)
	}

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

func (l *Loader) postProcessDatabase(db *models.EmbeddingDB, cachePath string, verbose bool) {
	// Build inverted index
	if verbose {
		fmt.Println("Building search index...")
	}
	indexStart := time.Now()
	BuildInvertedIndex(db)
	if verbose {
		fmt.Printf("Index built in %.2f seconds\n", time.Since(indexStart).Seconds())
	}

	// Save binary cache
	l.saveBinaryCache(db, cachePath, verbose)
}

func (l *Loader) saveBinaryCache(db *models.EmbeddingDB, cachePath string, verbose bool) {
	if verbose {
		fmt.Println("Saving binary cache for faster future loads...")
	}
	cacheStart := time.Now()

	if err := l.cacheManager.SaveBinaryCache(db, cachePath); err != nil {
		if verbose {
			fmt.Printf("Warning: Failed to save binary cache: %v\n", err)
		}
	} else if verbose {
		fmt.Printf("Binary cache saved in %.2f seconds\n", time.Since(cacheStart).Seconds())
	}
}

func (l *Loader) printLoadStats(db *models.EmbeddingDB, start time.Time) {
	fmt.Printf("Total load time: %.2f seconds\n", time.Since(start).Seconds())
	fmt.Printf("Loaded %d embeddings with %d indexed terms\n", len(db.Table), len(db.InvertedIndex))
}

// LoadDB loads an embedding database from disk with caching
// Deprecated: Use NewLoader with dependency injection instead
func LoadDB(path string, verbose bool) (*models.EmbeddingDB, error) {
	loader := NewLoader(cache.NewCacheManager())
	return loader.Load(path, verbose)
}
