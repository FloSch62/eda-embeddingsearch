// Command embeddingsearch is the entry point for the embedding search CLI.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/eda-labs/eda-embeddingsearch/internal/cache"
	"github.com/eda-labs/eda-embeddingsearch/internal/download"
	"github.com/eda-labs/eda-embeddingsearch/internal/embedding"
	"github.com/eda-labs/eda-embeddingsearch/internal/search"
	"github.com/eda-labs/eda-embeddingsearch/pkg/models"
)

func main() {
	dbPath := flag.String("db", "", "path to embedding db (auto-downloads if not specified)")
	jsonOutput := flag.Bool("json", false, "output results as JSON")
	platformStr := flag.String("platform", "", "force platform type (srl or sros)")
	setup := flag.Bool("setup", false, "download all embeddings and build caches")
	flag.Parse()

	if *setup || (flag.NArg() > 0 && flag.Arg(0) == "setup") {
		if err := runSetup(); err != nil {
			fmt.Fprintf(os.Stderr, "setup failed: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("setup completed")
		return
	}

	if flag.NArg() == 0 {
		fmt.Println("usage: embeddingsearch [-json] [-platform srl|sros] <query>")
		fmt.Println("\nExamples:")
		fmt.Println("  embeddingsearch 'show interface statistics for leaf1'")
		fmt.Println("  embeddingsearch 'get top 5 processes by memory usage'")
		fmt.Println("  embeddingsearch 'critical alarms from the last hour'")
		fmt.Println("  embeddingsearch 'interface traffic on spine1 every 5 seconds'")
		fmt.Println("  embeddingsearch -json 'show interfaces'  # Output as JSON")
		return
	}

	query := strings.Join(flag.Args(), " ")

	// Determine platform
	var platform models.EmbeddingType
	if *platformStr != "" {
		switch strings.ToLower(*platformStr) {
		case "sros":
			platform = models.SROS
		case "srl":
			platform = models.SRL
		default:
			fmt.Fprintf(os.Stderr, "Invalid platform: %s (must be 'srl' or 'sros')\n", *platformStr)
			os.Exit(1)
		}
	} else {
		// Auto-detect from query if not specified
		platform = download.DetectPlatformFromQuery(query)
	}

	// Determine the database path
	var finalDBPath string
	if *dbPath != "" {
		finalDBPath = *dbPath
	} else {
		// Auto-download embeddings if not specified
		downloader := download.NewDownloader()
		var err error
		finalDBPath, err = downloader.EnsureEmbeddings(platform)
		if err != nil {
			fmt.Fprintf(os.Stderr, "failed to download embeddings: %v\n", err)
			os.Exit(1)
		}
	}

	loader := embedding.NewLoader(cache.NewCacheManager())
	db, err := loader.Load(finalDBPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to load db: %v\n", err)
		os.Exit(1)
	}

	// Create search engine and perform search
	engine := search.NewEngine(db)
	results := engine.IndexedSearch(query)

	if len(results) == 0 {
		if *jsonOutput {
			fmt.Println(`{"error": "No matches found", "results": []}`)
		} else {
			fmt.Println("No matches found")
		}
		return
	}

	if *jsonOutput {
		outputJSON(results)
	} else {
		outputText(results)
	}
}

func outputJSON(results []models.SearchResult) {
	type JSONOutput struct {
		TopMatch *models.SearchResult   `json:"topMatch"`
		Others   []*models.SearchResult `json:"others,omitempty"`
	}

	output := JSONOutput{TopMatch: &results[0]}

	// Add other matches (limit to 9 more for total of 10)
	maxOthers := 9
	if len(results)-1 < maxOthers {
		maxOthers = len(results) - 1
	}
	if maxOthers > 0 {
		output.Others = make([]*models.SearchResult, maxOthers)
		for i := 0; i < maxOthers; i++ {
			output.Others[i] = &results[i+1]
		}
	}

	jsonData, err := json.MarshalIndent(output, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to marshal JSON: %v\n", err)
		os.Exit(1)
	}
	fmt.Println(string(jsonData))
}

func outputText(results []models.SearchResult) {
	// Display top match
	top := results[0]
	fmt.Printf("Top match (score: %.2f):\n%s\n", top.Score, top.EQLQuery.String())

	if top.Description != "" {
		fmt.Printf("\nDescription: %s\n", top.Description)
	}
	if len(top.AvailableFields) > 0 {
		fmt.Printf("Available fields: %s\n", strings.Join(top.AvailableFields, ", "))
	}

	// Show other matches (limit to 9 more for total of 10)
	if len(results) > 1 {
		fmt.Println("\nOther possible matches:")
		maxOthers := 9
		if len(results)-1 < maxOthers {
			maxOthers = len(results) - 1
		}
		for i := 1; i <= maxOthers; i++ {
			other := results[i]
			fmt.Printf("%d. %s (score: %.2f)\n", i, other.EQLQuery.String(), other.Score)
			if other.Description != "" {
				fmt.Printf("   Description: %s\n", other.Description)
			}
			if len(other.AvailableFields) > 0 {
				fmt.Printf("   Available fields: %s\n", strings.Join(other.AvailableFields, ", "))
			}
		}
	}
}

func runSetup() error {
	downloader := download.NewDownloader()
	loader := embedding.NewLoader(cache.NewCacheManager())

	platforms := []models.EmbeddingType{models.SRL, models.SROS}
	for _, platform := range platforms {
		name := ""
		if platform == models.SROS {
			name = "SROS"
		} else {
			name = "SRL"
		}

		fmt.Printf("Downloading embeddings for %s...\n", name)
		path, err := downloader.EnsureEmbeddings(platform)
		if err != nil {
			return err
		}
		fmt.Printf("Downloaded to %s\n", path)

		fmt.Printf("Loading embeddings for %s into cache...\n", name)
		if _, err := loader.Load(path); err != nil {
			return err
		}
		fmt.Printf("Loaded embeddings for %s\n", name)
	}

	return nil
}
