package search

import (
	"sort"

	"github.com/eda-labs/eda-embeddingsearch/internal/constants"
	"github.com/eda-labs/eda-embeddingsearch/internal/eql"
	"github.com/eda-labs/eda-embeddingsearch/pkg/models"
)

// Search performs a full search across all embeddings
func (e *Engine) Search(query string) []models.SearchResult {
	words := ExpandSynonyms(Tokenize(query))

	results := make([]models.SearchResult, 0)

	// Check for alarm queries first
	if alarmResult := e.checkAlarmQuery(query, words); alarmResult != nil {
		results = append(results, *alarmResult)
	}

	// Find best candidates using parallel search
	candidates := e.findTopCandidates(query, words)

	// Convert candidates to search results
	results = e.convertCandidatesToResults(candidates, query, results)

	sort.Slice(results, func(i, j int) bool {
		return results[i].Score > results[j].Score
	})

	return results
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
			score += constants.AlarmWordScore
		}
		if w == "critical" || w == "major" || w == "minor" {
			score += constants.AlarmSeverityScore
		}
	}
	return score
}

type candidate struct {
	key   string
	score float64
}

func (e *Engine) findTopCandidates(query string, words []string) []candidate {
	const scoreThreshold = constants.MinScoreThreshold
	const maxCandidates = constants.MaxCandidates

	candidates := make([]candidate, 0, maxCandidates)
	minScore := scoreThreshold

	// Process all entries directly without parallelism
	for key, entry := range e.db.Table {
		score := e.scoreEntry(key, entry, query, words)

		if score > scoreThreshold {
			// Update top candidates inline
			if len(candidates) < maxCandidates {
				candidates = append(candidates, candidate{key: key, score: score})
				if len(candidates) == maxCandidates {
					sort.Slice(candidates, func(i, j int) bool {
						return candidates[i].score > candidates[j].score
					})
					minScore = candidates[maxCandidates-1].score
				}
			} else if score > minScore {
				candidates[maxCandidates-1] = candidate{key: key, score: score}
				sort.Slice(candidates, func(i, j int) bool {
					return candidates[i].score > candidates[j].score
				})
				minScore = candidates[maxCandidates-1].score
			}
		}
	}

	return candidates
}

func (e *Engine) convertCandidatesToResults(candidates []candidate, query string, results []models.SearchResult) []models.SearchResult {
	for _, cand := range candidates {
		if len(results) >= constants.MaxSearchResults {
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
		Explanation:     e.generateExplanation(cand.key, query, cand.score),
	}
}
