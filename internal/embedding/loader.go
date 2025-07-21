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
	if cached, exists := cache.GetFromMemory(path); exists {
		if verbose {
			fmt.Printf("Using in-memory cached embeddings\n")
		}
		return cached, nil
	}

	// Check binary cache
	cachePath := cache.GetBinaryCachePath(path)
	if cache.IsBinaryCacheValid(path, cachePath) {
		if verbose {
			fmt.Printf("Loading from binary cache...\n")
		}
		start := time.Now()

		db, err := cache.LoadBinaryCache(cachePath)
		if err == nil {
			if verbose {
				fmt.Printf("Loaded binary cache in %.2f seconds\n", time.Since(start).Seconds())
			}

			// Cache in memory
			cache.StoreInMemory(path, db)
			return db, nil
		}
		if verbose {
			fmt.Printf("Binary cache failed, falling back to JSON: %v\n", err)
		}
	}

	// Load from JSON file
	if verbose {
		fmt.Printf("Loading embeddings from %s...\n", filepath.Base(path))
	}
	start := time.Now()

	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var db models.EmbeddingDB
	dec := json.NewDecoder(file)
	if err := dec.Decode(&db); err != nil {
		return nil, err
	}

	jsonLoadTime := time.Since(start).Seconds()
	if verbose {
		fmt.Printf("JSON loaded in %.2f seconds\n", jsonLoadTime)
	}

	// Build inverted index
	if verbose {
		fmt.Println("Building search index...")
	}
	indexStart := time.Now()
	BuildInvertedIndex(&db)
	if verbose {
		fmt.Printf("Index built in %.2f seconds\n", time.Since(indexStart).Seconds())
	}

	// Save binary cache for next time
	if verbose {
		fmt.Println("Saving binary cache for faster future loads...")
	}
	cacheStart := time.Now()
	if err := cache.SaveBinaryCache(&db, cachePath); err != nil {
		if verbose {
			fmt.Printf("Warning: Failed to save binary cache: %v\n", err)
		}
	} else {
		if verbose {
			fmt.Printf("Binary cache saved in %.2f seconds\n", time.Since(cacheStart).Seconds())
		}
	}

	// Cache in memory
	cache.StoreInMemory(path, &db)

	if verbose {
		fmt.Printf("Total load time: %.2f seconds\n", time.Since(start).Seconds())
		fmt.Printf("Loaded %d embeddings with %d indexed terms\n", len(db.Table), len(db.InvertedIndex))
	}
	return &db, nil
}