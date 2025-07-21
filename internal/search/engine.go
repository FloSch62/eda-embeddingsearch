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

	isSROSDB := e.detectSROSDatabase()
	candidateKeys := e.getCandidateKeys(words, query, isSROSDB)

	// If no candidates from index, fall back to full search
	if len(candidateKeys) == 0 {
		return e.Search(query)
	}

	// Score candidates and generate results
	candidates := e.scoreCandidates(candidateKeys, query, words)
	return e.generateVectorSearchResults(candidates, query)
}

func (e *Engine) detectSROSDatabase() bool {
	for key := range e.db.Table {
		if strings.Contains(key, ".sros.") {
			return true
		}
	}
	return false
}

func (e *Engine) getCandidateKeys(words []string, query string, isSROSDB bool) map[string]int {
	candidateKeys := make(map[string]int)

	// Use inverted index to get candidate keys
	e.addIndexedCandidates(words, candidateKeys)

	// For SROS database or queries, ensure we get interface-related entries
	if shouldAddInterfaceCandidates(words, query, isSROSDB) {
		e.addInterfaceCandidates(candidateKeys)
	}

	return candidateKeys
}

func (e *Engine) addIndexedCandidates(words []string, candidateKeys map[string]int) {
	for _, word := range words {
		if keys, exists := e.db.InvertedIndex[word]; exists {
			for _, key := range keys {
				candidateKeys[key]++
			}
		}
	}
}

func shouldAddInterfaceCandidates(words []string, query string, isSROSDB bool) bool {
	if !isSROSDB && download.DetectEmbeddingType(query) != models.SROS {
		return false
	}

	for _, word := range words {
		if word == "interface" || word == "interfaces" {
			return true
		}
	}
	return false
}

func (e *Engine) addInterfaceCandidates(candidateKeys map[string]int) {
	for indexWord, keys := range e.db.InvertedIndex {
		if strings.Contains(indexWord, "interface") {
			for _, key := range keys {
				candidateKeys[key]++
			}
		}
	}
}

type scoredCandidate struct {
	key   string
	score float64
}

func (e *Engine) scoreCandidates(candidateKeys map[string]int, query string, words []string) []scoredCandidate {
	bigrams := generateBigrams(words)
	candidates := make([]scoredCandidate, 0, len(candidateKeys))

	for key, matchCount := range candidateKeys {
		score := e.calculateCandidateScore(key, matchCount, query, words, bigrams)
		threshold := getScoreThreshold(key)

		if score > threshold {
			candidates = append(candidates, scoredCandidate{
				key:   key,
				score: score,
			})
		}
	}

	// Sort candidates by score
	sort.Slice(candidates, func(i, j int) bool {
		return candidates[i].score > candidates[j].score
	})

	return candidates
}

func (e *Engine) calculateCandidateScore(key string, matchCount int, query string, words, bigrams []string) float64 {
	entry := e.db.Table[key]

	// Base score from inverted index matches
	baseScore := float64(matchCount) * 10

	// Bonus for having all query words in the key
	if hasAllWords(key, words) {
		baseScore += float64(len(words)) * 20
	}

	// Additional scoring
	additionalScore := e.scoreEntry(key, entry, query, words, bigrams)

	return baseScore + additionalScore
}

func hasAllWords(key string, words []string) bool {
	keyLower := strings.ToLower(key)
	for _, word := range words {
		if !strings.Contains(keyLower, word) {
			return false
		}
	}
	return true
}

func getScoreThreshold(key string) float64 {
	if strings.Contains(key, ".sros.") {
		return 8.0 // Lower threshold for SROS
	}
	return 10.0
}

func (e *Engine) generateVectorSearchResults(candidates []scoredCandidate, query string) []models.SearchResult {
	results := make([]models.SearchResult, 0, 10)

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
	bigrams := generateBigrams(words)

	results := make([]models.SearchResult, 0)

	// Check for alarm queries first
	if alarmResult := e.checkAlarmQuery(query, words); alarmResult != nil {
		results = append(results, *alarmResult)
	}

	// Find best candidates using parallel search
	candidates := e.findTopCandidates(query, words, bigrams)

	// Convert candidates to search results
	results = e.convertCandidatesToResults(candidates, query, results)

	sort.Slice(results, func(i, j int) bool {
		return results[i].Score > results[j].Score
	})

	return results
}

func generateBigrams(words []string) []string {
	bigrams := make([]string, 0, len(words)-1)
	for i := 0; i < len(words)-1; i++ {
		bigrams = append(bigrams, words[i]+" "+words[i+1])
	}
	return bigrams
}

func (e *Engine) checkAlarmQuery(query string, words []string) *models.SearchResult {
	alarmScore := calculateAlarmScore(words)
	if alarmScore == 0 {
		return nil
	}

	alarmPath := ".namespace.alarms.v1.alarm"
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

	return &models.SearchResult{
		Key:             alarmPath,
		Score:           alarmScore,
		EQLQuery:        eqlQuery,
		Description:     "Active alarms in the system",
		AvailableFields: []string{"severity", "text", "time-created", "acknowledged"},
	}
}

func calculateAlarmScore(words []string) float64 {
	score := 0.0
	for _, w := range words {
		if w == "alarm" || w == "alarms" {
			score += 10
		}
		if w == "critical" || w == "major" || w == "minor" {
			score += 5
		}
	}
	return score
}

type candidate struct {
	key   string
	score float64
}

func (e *Engine) findTopCandidates(query string, words, bigrams []string) []candidate {
	const maxWorkers = 4
	const chunkSize = 2000
	const scoreThreshold = 5.0
	const maxCandidates = 20

	keys := e.getAllKeys()
	candidateChan := make(chan candidate, 50)

	// Process chunks in parallel
	e.processChunksParallel(keys, query, words, bigrams, candidateChan, maxWorkers, chunkSize, scoreThreshold)

	// Collect top candidates
	return collectTopCandidates(candidateChan, maxCandidates, scoreThreshold)
}

func (e *Engine) getAllKeys() []string {
	keys := make([]string, 0, len(e.db.Table))
	for key := range e.db.Table {
		keys = append(keys, key)
	}
	return keys
}

func (e *Engine) processChunksParallel(keys []string, query string, words, bigrams []string,
	candidateChan chan<- candidate, maxWorkers, chunkSize int, scoreThreshold float64) {

	var wg sync.WaitGroup
	semaphore := make(chan struct{}, maxWorkers)

	for i := 0; i < len(keys); i += chunkSize {
		end := minInt(i+chunkSize, len(keys))

		wg.Add(1)
		semaphore <- struct{}{}

		go func(start, end int) {
			defer wg.Done()
			defer func() { <-semaphore }()

			e.processChunk(keys[start:end], query, words, bigrams, candidateChan, scoreThreshold)
		}(i, end)
	}

	go func() {
		wg.Wait()
		close(candidateChan)
	}()
}

func (e *Engine) processChunk(keys []string, query string, words, bigrams []string,
	candidateChan chan<- candidate, scoreThreshold float64) {

	for _, key := range keys {
		entry := e.db.Table[key]
		score := e.scoreEntry(key, entry, query, words, bigrams)

		if score > scoreThreshold {
			candidateChan <- candidate{key: key, score: score}
		}
	}
}

func collectTopCandidates(candidateChan <-chan candidate, maxCandidates int, scoreThreshold float64) []candidate {
	candidates := make([]candidate, 0, maxCandidates)
	minScore := scoreThreshold

	for cand := range candidateChan {
		candidates = updateTopCandidates(candidates, cand, maxCandidates, &minScore)
	}

	return candidates
}

func updateTopCandidates(candidates []candidate, newCand candidate, maxCandidates int, minScore *float64) []candidate {
	if len(candidates) < maxCandidates {
		candidates = append(candidates, newCand)
		if len(candidates) == maxCandidates {
			sort.Slice(candidates, func(i, j int) bool {
				return candidates[i].score > candidates[j].score
			})
			*minScore = candidates[maxCandidates-1].score
		}
	} else if newCand.score > *minScore {
		candidates[maxCandidates-1] = newCand
		sort.Slice(candidates, func(i, j int) bool {
			return candidates[i].score > candidates[j].score
		})
		*minScore = candidates[maxCandidates-1].score
	}
	return candidates
}

func (e *Engine) convertCandidatesToResults(candidates []candidate, query string, results []models.SearchResult) []models.SearchResult {
	for _, cand := range candidates {
		if len(results) >= 10 {
			break
		}

		result := e.createSearchResult(cand, query)
		results = append(results, result)
	}
	return results
}

func (e *Engine) createSearchResult(cand candidate, query string) models.SearchResult {
	entry := e.db.Table[cand.key]

	description, availableFields := parseEmbeddingInfo(entry.Text)

	eqlQuery := models.EQLQuery{
		Table:       cand.key,
		Fields:      eql.ExtractFields(query, cand.key, &entry),
		WhereClause: eql.GenerateWhereClause(cand.key, query),
		OrderBy:     eql.ExtractOrderBy(query, cand.key, &entry),
		Limit:       eql.ExtractLimit(query),
		Delta:       eql.ExtractDelta(query),
	}

	return models.SearchResult{
		Key:             cand.key,
		Score:           cand.score,
		EQLQuery:        eqlQuery,
		Description:     description,
		AvailableFields: availableFields,
	}
}

func parseEmbeddingInfo(text string) (description string, fields []string) {
	var embeddingInfo struct {
		Description string   `json:"Description"`
		Fields      []string `json:"Fields"`
	}

	if err := json.Unmarshal([]byte(text), &embeddingInfo); err == nil {
		return embeddingInfo.Description, embeddingInfo.Fields
	}

	return "", []string{}
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
