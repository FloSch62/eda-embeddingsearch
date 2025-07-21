package search

import (
	"sort"
	"strings"

	"github.com/eda-labs/eda-embeddingsearch/internal/constants"
)

type scoredCandidate struct {
	key   string
	score float64
}

func (e *Engine) scoreCandidates(candidateKeys map[string]int, query string, words []string) []scoredCandidate {
	candidates := make([]scoredCandidate, 0, len(candidateKeys))

	for key, matchCount := range candidateKeys {
		score := e.calculateCandidateScore(key, matchCount, query, words)
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

func (e *Engine) calculateCandidateScore(key string, matchCount int, query string, words []string) float64 {
	entry := e.db.Table[key]

	// Base score from inverted index matches
	baseScore := float64(matchCount) * constants.BaseIndexMatchScore

	// Bonus for having all query words in the key
	if hasAllWords(key, words) {
		baseScore += float64(len(words)) * constants.AllWordsMatchBonus
	}

	// Additional scoring
	additionalScore := e.scoreEntry(key, entry, query, words)

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
		return constants.SROSScoreThreshold
	}
	return constants.DefaultScoreThreshold
}
