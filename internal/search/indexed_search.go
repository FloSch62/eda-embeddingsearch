package search

import (
	"strings"

	"github.com/eda-labs/eda-embeddingsearch/internal/constants"
	"github.com/eda-labs/eda-embeddingsearch/internal/download"
	"github.com/eda-labs/eda-embeddingsearch/internal/eql"
	"github.com/eda-labs/eda-embeddingsearch/pkg/models"
)

// IndexedSearch performs fast search using the prebuilt inverted index
// and falls back to the full search if no candidates are found.
func (e *Engine) IndexedSearch(query string) []models.SearchResult {
	words := ExpandSynonyms(Tokenize(query))

	isSROSDB := e.detectSROSDatabase()
	candidateKeys := e.getCandidateKeys(words, query, isSROSDB)

	// If no candidates from index, fall back to full search
	if len(candidateKeys) == 0 {
		return e.Search(query)
	}

	// Score candidates and generate results
	candidates := e.scoreCandidates(candidateKeys, query, words)
	return e.generateIndexedSearchResults(candidates, query)
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
	if !isSROSDB && download.DetectPlatformFromQuery(query) != models.SROS {
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

func (e *Engine) generateIndexedSearchResults(candidates []scoredCandidate, query string) []models.SearchResult {
	results := make([]models.SearchResult, 0, constants.MaxSearchResults)

	for i, cand := range candidates {
		if i >= constants.MaxSearchResults {
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
			Key:         cand.key,
			Score:       cand.score,
			EQLQuery:    eqlQuery,
			Explanation: e.generateExplanation(cand.key, query, cand.score),
		})
	}

	return results
}
