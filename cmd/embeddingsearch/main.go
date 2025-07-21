package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/eda-labs/eda-embeddingsearch/internal/download"
	"github.com/eda-labs/eda-embeddingsearch/internal/embedding"
	"github.com/eda-labs/eda-embeddingsearch/internal/search"
	"github.com/eda-labs/eda-embeddingsearch/pkg/models"
)

func main() {
	dbPath := flag.String("db", "", "path to embedding db (auto-downloads if not specified)")
	verbose := flag.Bool("v", false, "verbose output showing all query components")
	jsonOutput := flag.Bool("json", false, "output results as JSON")
	platformStr := flag.String("platform", "", "force platform type (srl or sros)")
	flag.Parse()

	if flag.NArg() == 0 {
		fmt.Println("usage: embeddingsearch [-v] [-json] [-platform srl|sros] <query>")
		fmt.Println("\nExamples:")
		fmt.Println("  embeddingsearch 'show interface statistics for leaf1'")
		fmt.Println("  embeddingsearch 'get top 5 processes by memory usage'")
		fmt.Println("  embeddingsearch 'critical alarms from the last hour'")
		fmt.Println("  embeddingsearch 'interface traffic on spine1 every 5 seconds'")
		fmt.Println("  embeddingsearch -json 'show interfaces'  # Output as JSON")
		return
	}

	query := strings.Join(flag.Args(), " ")

	// Override platform detection if specified
	if *platformStr != "" {
		if *platformStr == "sros" {
			// Prepend SROS to query to force SROS detection
			query = "SROS " + query
		}
		// For SRL, no need to prepend anything as it's the default
	}

	// Determine the database path
	var finalDBPath string
	if *dbPath != "" {
		finalDBPath = *dbPath
	} else {
		// Auto-download embeddings if not specified (based on query content)
		var err error
		finalDBPath, err = download.DownloadAndExtractEmbeddings(query, !*jsonOutput)
		if err != nil {
			fmt.Fprintf(os.Stderr, "failed to download embeddings: %v\n", err)
			os.Exit(1)
		}
	}

	db, err := embedding.LoadDB(finalDBPath, !*jsonOutput)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to load db: %v\n", err)
		os.Exit(1)
	}

	// Create search engine and perform search
	engine := search.NewEngine(db)
	results := engine.VectorSearch(query)

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
		outputText(results, *verbose)
	}
}

func outputJSON(results []models.SearchResult) {
	type JSONResult struct {
		Score           float64  `json:"score"`
		Query           string   `json:"query"`
		Table           string   `json:"table"`
		Description     string   `json:"description,omitempty"`
		AvailableFields []string `json:"availableFields,omitempty"`
		Fields          []string `json:"fields,omitempty"`
		Where           string   `json:"where,omitempty"`
		OrderBy         []struct {
			Field     string `json:"field"`
			Direction string `json:"direction"`
			Algorithm string `json:"algorithm,omitempty"`
		} `json:"orderBy,omitempty"`
		Limit int `json:"limit,omitempty"`
		Delta *struct {
			Unit  string `json:"unit"`
			Value int    `json:"value"`
		} `json:"delta,omitempty"`
	}

	type JSONOutput struct {
		TopMatch JSONResult   `json:"topMatch"`
		Others   []JSONResult `json:"others,omitempty"`
	}

	// Convert top match
	top := results[0]
	topMatch := JSONResult{
		Score:           top.Score,
		Query:           top.EQLQuery.String(),
		Table:           top.EQLQuery.Table,
		Description:     top.Description,
		AvailableFields: top.AvailableFields,
		Fields:          top.EQLQuery.Fields,
		Where:           top.EQLQuery.WhereClause,
		Limit:           top.EQLQuery.Limit,
	}

	if len(top.EQLQuery.OrderBy) > 0 {
		for _, ob := range top.EQLQuery.OrderBy {
			topMatch.OrderBy = append(topMatch.OrderBy, struct {
				Field     string `json:"field"`
				Direction string `json:"direction"`
				Algorithm string `json:"algorithm,omitempty"`
			}{
				Field:     ob.Field,
				Direction: ob.Direction,
				Algorithm: ob.Algorithm,
			})
		}
	}

	if top.EQLQuery.Delta != nil {
		topMatch.Delta = &struct {
			Unit  string `json:"unit"`
			Value int    `json:"value"`
		}{
			Unit:  top.EQLQuery.Delta.Unit,
			Value: top.EQLQuery.Delta.Value,
		}
	}

	output := JSONOutput{TopMatch: topMatch}

	// Add other matches
	maxOthers := 9
	if len(results)-1 < maxOthers {
		maxOthers = len(results) - 1
	}
	for i := 1; i <= maxOthers; i++ {
		r := results[i]
		other := JSONResult{
			Score:           r.Score,
			Query:           r.EQLQuery.String(),
			Table:           r.EQLQuery.Table,
			Description:     r.Description,
			AvailableFields: r.AvailableFields,
			Fields:          r.EQLQuery.Fields,
			Where:           r.EQLQuery.WhereClause,
			Limit:           r.EQLQuery.Limit,
		}

		if len(r.EQLQuery.OrderBy) > 0 {
			for _, ob := range r.EQLQuery.OrderBy {
				other.OrderBy = append(other.OrderBy, struct {
					Field     string `json:"field"`
					Direction string `json:"direction"`
					Algorithm string `json:"algorithm,omitempty"`
				}{
					Field:     ob.Field,
					Direction: ob.Direction,
					Algorithm: ob.Algorithm,
				})
			}
		}

		if r.EQLQuery.Delta != nil {
			other.Delta = &struct {
				Unit  string `json:"unit"`
				Value int    `json:"value"`
			}{
				Unit:  r.EQLQuery.Delta.Unit,
				Value: r.EQLQuery.Delta.Value,
			}
		}

		output.Others = append(output.Others, other)
	}

	jsonData, err := json.MarshalIndent(output, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to marshal JSON: %v\n", err)
		os.Exit(1)
	}
	fmt.Println(string(jsonData))
}

func outputText(results []models.SearchResult, verbose bool) {
	// Display top match
	top := results[0]
	fmt.Printf("Top match (score: %.2f):\n%s\n", top.Score, top.EQLQuery.String())

	if verbose {
		fmt.Println("\nQuery components:")
		fmt.Printf("  Table: %s\n", top.EQLQuery.Table)
		if len(top.EQLQuery.Fields) > 0 {
			fmt.Printf("  Fields: %s\n", strings.Join(top.EQLQuery.Fields, ", "))
		}
		if top.EQLQuery.WhereClause != "" {
			fmt.Printf("  Where: %s\n", top.EQLQuery.WhereClause)
		}
		if len(top.EQLQuery.OrderBy) > 0 {
			fmt.Print("  Order by: ")
			for i, ob := range top.EQLQuery.OrderBy {
				if i > 0 {
					fmt.Print(", ")
				}
				fmt.Printf("%s %s", ob.Field, ob.Direction)
				if ob.Algorithm != "" {
					fmt.Printf(" %s", ob.Algorithm)
				}
			}
			fmt.Println()
		}
		if top.EQLQuery.Limit > 0 {
			fmt.Printf("  Limit: %d\n", top.EQLQuery.Limit)
		}
		if top.EQLQuery.Delta != nil {
			fmt.Printf("  Delta: %s %d\n", top.EQLQuery.Delta.Unit, top.EQLQuery.Delta.Value)
		}
	}

	// Show other matches (limit to 9 more for total of 10)
	if len(results) > 1 {
		fmt.Println("\nOther possible matches:")
		maxOthers := 9
		if len(results)-1 < maxOthers {
			maxOthers = len(results) - 1
		}
		for i := 1; i <= maxOthers; i++ {
			fmt.Printf("%d. %s (score: %.2f)\n", i, results[i].EQLQuery.String(), results[i].Score)
		}
	}
}
