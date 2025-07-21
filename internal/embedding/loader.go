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

// LoadDB loads an embedding database from disk with caching
func LoadDB(path string, verbose bool) (*models.EmbeddingDB, error) {
	// Check memory cache first
	if db := loadFromMemoryCache(path, verbose); db != nil {
		return db, nil
	}

	// Check binary cache
	cachePath := cache.GetBinaryCachePath(path)
	if db := loadFromBinaryCache(path, cachePath, verbose); db != nil {
		return db, nil
	}

	// Load from JSON file
	return loadFromJSON(path, cachePath, verbose)
}

func loadFromMemoryCache(path string, verbose bool) *models.EmbeddingDB {
	if cached, exists := cache.GetFromMemory(path); exists {
		if verbose {
			fmt.Printf("Using in-memory cached embeddings\n")
		}
		return cached
	}
	return nil
}

func loadFromBinaryCache(path, cachePath string, verbose bool) *models.EmbeddingDB {
	if !cache.IsBinaryCacheValid(path, cachePath) {
		return nil
	}

	if verbose {
		fmt.Printf("Loading from binary cache...\n")
	}
	start := time.Now()

	db, err := cache.LoadBinaryCache(cachePath)
	if err == nil {
		if verbose {
			fmt.Printf("Loaded binary cache in %.2f seconds\n", time.Since(start).Seconds())
		}
		cache.StoreInMemory(path, db)
		return db
	}

	if verbose {
		fmt.Printf("Binary cache failed, falling back to JSON: %v\n", err)
	}
	return nil
}

func loadFromJSON(path, cachePath string, verbose bool) (*models.EmbeddingDB, error) {
	if verbose {
		fmt.Printf("Loading embeddings from %s...\n", filepath.Base(path))
	}
	start := time.Now()

	// Load JSON data
	db, err := loadJSONFile(path)
	if err != nil {
		return nil, err
	}

	if verbose {
		fmt.Printf("JSON loaded in %.2f seconds\n", time.Since(start).Seconds())
	}

	// Build index and save cache
	postProcessDatabase(db, cachePath, verbose)

	// Cache in memory
	cache.StoreInMemory(path, db)

	if verbose {
		printLoadStats(db, start)
	}

	return db, nil
}

func loadJSONFile(path string) (*models.EmbeddingDB, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = file.Close()
	}()

	var db models.EmbeddingDB
	dec := json.NewDecoder(file)
	if err := dec.Decode(&db); err != nil {
		return nil, err
	}

	return &db, nil
}

func postProcessDatabase(db *models.EmbeddingDB, cachePath string, verbose bool) {
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
	saveBinaryCache(db, cachePath, verbose)
}

func saveBinaryCache(db *models.EmbeddingDB, cachePath string, verbose bool) {
	if verbose {
		fmt.Println("Saving binary cache for faster future loads...")
	}
	cacheStart := time.Now()

	if err := cache.SaveBinaryCache(db, cachePath); err != nil {
		if verbose {
			fmt.Printf("Warning: Failed to save binary cache: %v\n", err)
		}
	} else if verbose {
		fmt.Printf("Binary cache saved in %.2f seconds\n", time.Since(cacheStart).Seconds())
	}
}

func printLoadStats(db *models.EmbeddingDB, start time.Time) {
	fmt.Printf("Total load time: %.2f seconds\n", time.Since(start).Seconds())
	fmt.Printf("Loaded %d embeddings with %d indexed terms\n", len(db.Table), len(db.InvertedIndex))
}
