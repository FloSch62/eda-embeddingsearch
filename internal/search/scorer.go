package search

import (
	"encoding/json"
	"strings"

	"github.com/eda-labs/eda-embeddingsearch/internal/eql"
	"github.com/eda-labs/eda-embeddingsearch/internal/utils"
	"github.com/eda-labs/eda-embeddingsearch/pkg/models"
)

// scoreEntry calculates the score for a single embedding entry
func (e *Engine) scoreEntry(key string, entry models.EmbeddingEntry, query string, words []string, bigrams []string) float64 {
	keyTokens := Tokenize(key)
	textTokens := Tokenize(entry.ReferenceText + " " + entry.Text)
	score := 0.0
	queryLower := strings.ToLower(query)

	// Parse description from Text field if available
	var embeddingInfo struct {
		Description string   `json:"Description"`
		Fields      []string `json:"Fields"`
	}
	if err := json.Unmarshal([]byte(entry.Text), &embeddingInfo); err == nil {
		// Check if description contains query words
		descTokens := Tokenize(embeddingInfo.Description)
		descLower := strings.ToLower(embeddingInfo.Description)

		// Count matching words in description
		descMatchCount := 0
		for _, w := range words {
			if utils.Contains(descTokens, w) {
				descMatchCount++
				score += 3 // Increased from 2
			}
		}

		// Bonus for natural language phrases in description
		if strings.Contains(queryLower, "list of") && strings.Contains(descLower, "list of") {
			score += 5
		}
		if strings.Contains(queryLower, "all") && strings.Contains(descLower, "all") {
			score += 3
		}
		if strings.Contains(queryLower, "show") && strings.Contains(descLower, "display") {
			score += 2
		}
		if strings.Contains(queryLower, "get") && strings.Contains(descLower, "retrieve") {
			score += 2
		}

		// If multiple query words appear in description, give extra bonus
		if descMatchCount >= 2 && descMatchCount >= len(words)/2 {
			score += 5
		}
	}

	// Check for exact last segment match (high priority)
	if len(keyTokens) > 0 && len(words) > 0 {
		lastSegment := keyTokens[len(keyTokens)-1]
		for _, w := range words {
			if lastSegment == w {
				score += 10 // Reduced from 20
			}
		}
	}

	// Count exact word matches in path
	pathMatchCount := 0
	for _, w := range words {
		if utils.Contains(keyTokens, w) {
			pathMatchCount++
			// Give scores for important keywords (reduced values)
			if w == "interface" || w == "interfaces" {
				score += 8
			} else if w == "statistics" {
				score += 6
			} else if w == "state" || w == "configure" {
				score += 4
			} else {
				score += 3
			}
		} else if utils.Contains(textTokens, w) {
			score += 1
		}
	}

	// Bonus for paths that contain ALL query words (reduced)
	if pathMatchCount == len(words) && len(words) > 1 {
		score += float64(len(words)) * 3
	}

	// Prefer state paths for "show" commands
	if strings.Contains(queryLower, "show") && strings.Contains(key, ".state.") {
		score += 5
	}

	// Path hierarchy scoring - prefer direct matches over nested ones
	keyLower := strings.ToLower(key)

	// Special handling for interface queries - penalize security/violator paths
	if strings.Contains(queryLower, "interface") {
		if strings.Contains(keyLower, "violator") || strings.Contains(keyLower, "security") {
			score -= 20 // Reduced penalty
		}

		// Prefer paths that end with the main query term
		if strings.HasSuffix(key, ".interface") && !strings.Contains(key, ".protocols.") {
			score += 20 // Bonus for paths ending in interface (not protocol-specific)
		}

		// Prefer direct statistics paths
		if strings.Contains(queryLower, "statistics") && strings.HasSuffix(key, ".interface.statistics") {
			score += 15
		}

		// Penalize protocol-specific interface paths for general interface queries
		if !strings.Contains(queryLower, "bgp") && !strings.Contains(queryLower, "ospf") && !strings.Contains(queryLower, "isis") {
			if strings.Contains(keyLower, "protocols.bgp") || strings.Contains(keyLower, "protocols.ospf") || strings.Contains(keyLower, "protocols.isis") {
				score -= 15
			}
		}

		// For "interfaces" plural query, prefer the main interface table
		if strings.Contains(queryLower, "interfaces") && strings.HasSuffix(key, ".interface") {
			score += 10
		}
	}

	// Special handling for BGP queries
	if strings.Contains(queryLower, "bgp") && strings.Contains(queryLower, "neighbor") {
		// Prefer paths that have bgp and end with neighbor
		if strings.Contains(key, "bgp") && strings.HasSuffix(key, ".neighbor") {
			score += 15
		}
		// Penalize maintenance paths
		if strings.Contains(key, "maintenance") {
			score -= 10
		}
	}

	// Count the segments after the main keyword match
	for _, word := range words {
		if idx := strings.Index(keyLower, word); idx != -1 {
			// Count dots after the match
			afterMatch := keyLower[idx+len(word):]
			dotCount := strings.Count(afterMatch, ".")
			// Fewer dots = more direct match = higher score
			if dotCount == 0 {
				score += 10 // Perfect end match
			} else if dotCount <= 2 {
				score += 6
			} else if dotCount <= 4 {
				score += 2
			}
		}
	}

	// Special handling for subinterface queries
	if strings.Contains(query, "subinterface") && strings.Contains(key, "subinterface") {
		if strings.HasSuffix(key, ".subinterface") {
			score += 10 // Reduced
		} else {
			score += 2
		}
	}

	// Boost for exact table matches (last segment matches query word)
	for _, w := range words {
		if strings.HasSuffix(key, "."+w) {
			score += 6
		}
	}

	// Bigram matching
	for _, b := range bigrams {
		if strings.Contains(keyLower, strings.ReplaceAll(b, " ", ".")) {
			score += 2
		}
	}

	// Boost score for tables that can extract the requested fields
	extractedFields := eql.ExtractFields(query, key, &entry)
	if len(extractedFields) > 0 {
		score += float64(len(extractedFields)) * 1.5 // Reduced
	}

	// Prefer paths that have query words in sequence
	if strings.Contains(query, "interface") && strings.Contains(query, "statistics") {
		if strings.Contains(key, "interface") && strings.Contains(key, "statistics") {
			// Check if statistics comes right after interface
			if strings.Contains(key, "interface.statistics") {
				score += 8
			} else {
				score += 4
			}
		}
	}

	// Strongly prefer shorter, more direct paths
	pathDepth := len(keyTokens)
	if pathDepth > 0 {
		// Count meaningful segments (excluding namespace, node, nodename)
		meaningfulDepth := 0
		for i, token := range keyTokens {
			if i > 2 && token != "state" && token != "configure" {
				meaningfulDepth++
			}
		}

		// Strong preference for direct paths
		if meaningfulDepth <= 2 {
			score += 20 // Big bonus for very direct paths
		} else if meaningfulDepth <= 3 {
			score += 10
		} else if meaningfulDepth <= 4 {
			score += 5
		} else {
			// Penalize deeply nested paths
			score -= float64(meaningfulDepth-4) * 2
		}
	}

	// Additional penalty for overly specific paths when query is general
	if strings.Contains(key, "protocols") && !strings.Contains(queryLower, "protocol") &&
		!strings.Contains(queryLower, "bgp") && !strings.Contains(queryLower, "ospf") &&
		!strings.Contains(queryLower, "isis") {
		score -= 10 // Penalize protocol-specific paths for general queries
	}

	// Penalize maintenance/group paths unless specifically requested
	if strings.Contains(key, "maintenance") && !strings.Contains(queryLower, "maintenance") {
		score -= 8
	}

	// For error queries, prefer paths with error fields
	if strings.Contains(queryLower, "error") {
		if strings.Contains(key, "statistics") && len(extractedFields) > 0 {
			// Check if we found error-related fields
			for _, field := range extractedFields {
				if strings.Contains(field, "error") {
					score += 10
					break
				}
			}
		}
	}

	// For bandwidth queries, prefer interface paths with traffic fields
	if strings.Contains(queryLower, "bandwidth") && strings.Contains(key, "interface") {
		if len(extractedFields) > 0 {
			for _, field := range extractedFields {
				if strings.Contains(field, "octets") || strings.Contains(field, "bandwidth") {
					score += 10
					break
				}
			}
		}
	}

	return score
}
