package search

import (
	"encoding/json"
	"sort"
	"strings"
	"sync"

	"github.com/eda-labs/eda-embeddingsearch/internal/download"
	"github.com/eda-labs/eda-embeddingsearch/internal/eql"
	"github.com/eda-labs/eda-embeddingsearch/pkg/models"
)

// Engine represents the search engine
type Engine struct {
	db *models.EmbeddingDB
}

// NewEngine creates a new search engine
func NewEngine(db *models.EmbeddingDB) *Engine {
	return &Engine{db: db}
}

// VectorSearch performs fast indexed search with pre-filtering
func (e *Engine) VectorSearch(query string) []models.SearchResult {
	words := ExpandSynonyms(Tokenize(query))

	// Detect if this is an SROS database by checking a few entries
	isSROSDB := false
	for key := range e.db.Table {
		if strings.Contains(key, ".sros.") {
			isSROSDB = true
			break
		}
	}

	// Use inverted index to get candidate keys
	candidateKeys := make(map[string]int)
	for _, word := range words {
		if keys, exists := e.db.InvertedIndex[word]; exists {
			for _, key := range keys {
				candidateKeys[key]++
			}
		}
	}

	// For SROS database or queries, ensure we get interface-related entries
	if isSROSDB || download.DetectEmbeddingType(query) == models.SROS {
		// Add all entries containing "interface" if that's in the query
		for _, word := range words {
			if word == "interface" || word == "interfaces" {
				// Also check for variations
				for indexWord, keys := range e.db.InvertedIndex {
					if strings.Contains(indexWord, "interface") {
						for _, key := range keys {
							candidateKeys[key]++
						}
					}
				}
			}
		}
	}

	// If no candidates from index, fall back to full search
	if len(candidateKeys) == 0 {
		return e.Search(query)
	}

	// Score only the candidates
	results := make([]models.SearchResult, 0)
	bigrams := make([]string, 0, len(words)-1)
	for i := 0; i < len(words)-1; i++ {
		bigrams = append(bigrams, words[i]+" "+words[i+1])
	}

	// Process candidates with scoring
	type scoredCandidate struct {
		key   string
		score float64
	}

	candidates := make([]scoredCandidate, 0, len(candidateKeys))

	for key, matchCount := range candidateKeys {
		entry := e.db.Table[key]

		// Base score from inverted index matches
		baseScore := float64(matchCount) * 10

		// Bonus for having all query words in the key
		allWordsInKey := true
		for _, word := range words {
			if !strings.Contains(strings.ToLower(key), word) {
				allWordsInKey = false
				break
			}
		}
		if allWordsInKey {
			baseScore += float64(len(words)) * 20 // Big bonus for complete matches
		}

		// Additional scoring
		additionalScore := e.scoreEntry(key, entry, query, words, bigrams)

		totalScore := baseScore + additionalScore

		// Adjust threshold based on platform
		threshold := 10.0
		if strings.Contains(key, ".sros.") {
			threshold = 8.0 // Lower threshold for SROS to get better results
		}

		if totalScore > threshold {
			candidates = append(candidates, scoredCandidate{
				key:   key,
				score: totalScore,
			})
		}
	}

	// Sort candidates
	sort.Slice(candidates, func(i, j int) bool {
		return candidates[i].score > candidates[j].score
	})

	// Generate EQL for top 10
	for i, cand := range candidates {
		if i >= 10 {
			break
		}

		entry := e.db.Table[cand.key]
		eqlQuery := models.EQLQuery{
			Table:       cand.key,
			Fields:      eql.ExtractFields(query, cand.key, &entry),
			WhereClause: eql.GenerateWhereClause(cand.key, query),
			OrderBy:     eql.ExtractOrderBy(query, cand.key, &entry),
			Limit:       eql.ExtractLimit(query),
			Delta:       eql.ExtractDelta(query),
		}
		results = append(results, models.SearchResult{
			Key:      cand.key,
			Score:    cand.score,
			EQLQuery: eqlQuery,
		})
	}

	return results
}

// Search performs a full search across all embeddings
func (e *Engine) Search(query string) []models.SearchResult {
	words := ExpandSynonyms(Tokenize(query))
	bigrams := make([]string, 0, len(words)-1)
	for i := 0; i < len(words)-1; i++ {
		bigrams = append(bigrams, words[i]+" "+words[i+1])
	}

	results := make([]models.SearchResult, 0)

	// Check for alarm queries first (not in embeddings)
	alarmScore := 0.0
	for _, w := range words {
		if w == "alarm" || w == "alarms" {
			alarmScore += 10
		}
		if w == "critical" || w == "major" || w == "minor" {
			alarmScore += 5
		}
	}
	if alarmScore > 0 {
		alarmPath := ".namespace.alarms.v1.alarm"
		// Create a dummy embedding entry for alarms (not in actual embeddings)
		alarmEntry := &models.EmbeddingEntry{
			Text: `{"Description":"Active alarms in the system","Fields":["severity","text","time-created","acknowledged"]}`,
		}
		eqlQuery := models.EQLQuery{
			Table:       alarmPath,
			Fields:      eql.ExtractFields(query, alarmPath, alarmEntry),
			WhereClause: eql.GenerateWhereClause(alarmPath, query),
			OrderBy:     eql.ExtractOrderBy(query, alarmPath, alarmEntry),
			Limit:       eql.ExtractLimit(query),
			Delta:       eql.ExtractDelta(query),
		}
		results = append(results, models.SearchResult{
			Key:             alarmPath,
			Score:           alarmScore,
			EQLQuery:        eqlQuery,
			Description:     "Active alarms in the system",
			AvailableFields: []string{"severity", "text", "time-created", "acknowledged"},
		})
	}

	// Aggressive optimization for top-10 results only
	const maxWorkers = 4
	const chunkSize = 2000     // Larger chunks for better throughput
	const scoreThreshold = 5.0 // Higher threshold for faster filtering
	const maxCandidates = 20   // Only keep top 20 candidates during processing

	keys := make([]string, 0, len(e.db.Table))
	for key := range e.db.Table {
		keys = append(keys, key)
	}

	// Use a simpler structure for intermediate results - just score and key
	type candidate struct {
		key   string
		score float64
	}

	candidateChan := make(chan candidate, 50)
	var wg sync.WaitGroup

	// Create a semaphore to limit concurrent workers
	semaphore := make(chan struct{}, maxWorkers)

	// Process in chunks with early termination
	for i := 0; i < len(keys); i += chunkSize {
		end := i + chunkSize
		if end > len(keys) {
			end = len(keys)
		}

		wg.Add(1)
		// Acquire semaphore slot
		semaphore <- struct{}{}

		go func(start, end int) {
			defer wg.Done()
			defer func() { <-semaphore }() // Release semaphore slot

			for j := start; j < end; j++ {
				key := keys[j]
				entry := e.db.Table[key]

				score := e.scoreEntry(key, entry, query, words, bigrams)

				// Only process high-scoring entries
				if score > scoreThreshold {
					candidateChan <- candidate{key: key, score: score}
				}
			}
		}(i, end)
	}

	// Close channel when all workers are done
	go func() {
		wg.Wait()
		close(candidateChan)
	}()

	// Collect only the best candidates - maintain sorted list of top candidates
	candidates := make([]candidate, 0, maxCandidates)
	minScore := scoreThreshold

	for cand := range candidateChan {
		if len(candidates) < maxCandidates {
			candidates = append(candidates, cand)
			if len(candidates) == maxCandidates {
				// Sort and find minimum score
				sort.Slice(candidates, func(i, j int) bool {
					return candidates[i].score > candidates[j].score
				})
				minScore = candidates[maxCandidates-1].score
			}
		} else if cand.score > minScore {
			// Replace worst candidate
			candidates[maxCandidates-1] = cand
			// Re-sort to maintain order
			sort.Slice(candidates, func(i, j int) bool {
				return candidates[i].score > candidates[j].score
			})
			minScore = candidates[maxCandidates-1].score
		}
	}

	// Now generate EQL only for the top candidates
	for _, cand := range candidates {
		if len(results) >= 10 { // Only need 10 results maximum
			break
		}

		entry := e.db.Table[cand.key]

		// Parse description and fields from entry
		var embeddingInfo struct {
			Description string   `json:"Description"`
			Fields      []string `json:"Fields"`
		}
		description := ""
		availableFields := []string{}
		if err := json.Unmarshal([]byte(entry.Text), &embeddingInfo); err == nil {
			description = embeddingInfo.Description
			availableFields = embeddingInfo.Fields
		}

		eqlQuery := models.EQLQuery{
			Table:       cand.key,
			Fields:      eql.ExtractFields(query, cand.key, &entry),
			WhereClause: eql.GenerateWhereClause(cand.key, query),
			OrderBy:     eql.ExtractOrderBy(query, cand.key, &entry),
			Limit:       eql.ExtractLimit(query),
			Delta:       eql.ExtractDelta(query),
		}
		results = append(results, models.SearchResult{
			Key:             cand.key,
			Score:           cand.score,
			EQLQuery:        eqlQuery,
			Description:     description,
			AvailableFields: availableFields,
		})
	}

	sort.Slice(results, func(i, j int) bool {
		return results[i].Score > results[j].Score
	})

	// Limit to maximum 10 results for better performance
	if len(results) > 10 {
		results = results[:10]
	}

	return results
}
