package search

import (
	"fmt"
	"strings"
)

// generateExplanation creates a simple explanation for why a result was selected
func (e *Engine) generateExplanation(key, query string, score float64) string {
	words := Tokenize(query)
	keyLower := strings.ToLower(key)

	// Count matching words
	matchCount := 0
	matchedWords := []string{}
	for _, word := range words {
		if strings.Contains(keyLower, word) {
			matchCount++
			matchedWords = append(matchedWords, word)
		}
	}

	// Generate explanation based on matches
	if matchCount == 0 {
		return fmt.Sprintf("Matched based on semantic similarity (score: %.1f)", score)
	}

	if matchCount == len(words) {
		return fmt.Sprintf("Perfect match - all query terms found: %s (score: %.1f)",
			strings.Join(matchedWords, ", "), score)
	}

	return fmt.Sprintf("Partial match - found %d/%d terms: %s (score: %.1f)",
		matchCount, len(words), strings.Join(matchedWords, ", "), score)
}
