package search

import (
	"encoding/json"
	"strings"

	"github.com/eda-labs/eda-embeddingsearch/internal/eql"
	"github.com/eda-labs/eda-embeddingsearch/internal/utils"
	"github.com/eda-labs/eda-embeddingsearch/pkg/models"
)

func descriptionScore(queryLower string, entry models.EmbeddingEntry, words []string) float64 {
	var embeddingInfo struct {
		Description string   `json:"Description"`
		Fields      []string `json:"Fields"`
	}
	if err := json.Unmarshal([]byte(entry.Text), &embeddingInfo); err != nil {
		return 0
	}

	descTokens := Tokenize(embeddingInfo.Description)
	descLower := strings.ToLower(embeddingInfo.Description)
	score := 0.0

	descMatchCount := 0
	for _, w := range words {
		if utils.Contains(descTokens, w) {
			descMatchCount++
			score += 3
		}
	}

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
	if descMatchCount >= 2 && descMatchCount >= len(words)/2 {
		score += 5
	}
	return score
}

func keywordScore(keyTokens, textTokens, words []string) (float64, int) {
	score := 0.0
	pathMatchCount := 0
	if len(keyTokens) > 0 && len(words) > 0 {
		lastSegment := keyTokens[len(keyTokens)-1]
		for _, w := range words {
			if lastSegment == w {
				score += 10
			}
		}
	}

	for _, w := range words {
		if utils.Contains(keyTokens, w) {
			pathMatchCount++
			switch w {
			case "interface", "interfaces":
				score += 8
			case "statistics":
				score += 6
			case "state", "configure":
				score += 4
			default:
				score += 3
			}
		} else if utils.Contains(textTokens, w) {
			score += 1
		}
	}

	return score, pathMatchCount
}

func interfaceScore(key, keyLower, queryLower string, words []string) float64 {
	score := 0.0
	if strings.Contains(queryLower, "interface") {
		if strings.Contains(keyLower, "violator") || strings.Contains(keyLower, "security") {
			score -= 20
		}

		if strings.HasSuffix(key, ".interface") && !strings.Contains(key, ".protocols.") {
			score += 20
		}

		if strings.Contains(queryLower, "statistics") && strings.HasSuffix(key, ".interface.statistics") {
			score += 15
		}

		if !strings.Contains(queryLower, "bgp") && !strings.Contains(queryLower, "ospf") && !strings.Contains(queryLower, "isis") {
			if strings.Contains(keyLower, "protocols.bgp") || strings.Contains(keyLower, "protocols.ospf") || strings.Contains(keyLower, "protocols.isis") {
				score -= 15
			}
		}

		if strings.Contains(queryLower, "interfaces") && strings.HasSuffix(key, ".interface") {
			score += 10
		}
	}
	return score
}

func bgpScore(queryLower, key string) float64 {
	score := 0.0
	if strings.Contains(queryLower, "bgp") && strings.Contains(queryLower, "neighbor") {
		if strings.Contains(key, "bgp") && strings.HasSuffix(key, ".neighbor") {
			score += 15
		}
		if strings.Contains(key, "maintenance") {
			score -= 10
		}
	}
	return score
}

func segmentMatchScore(keyLower string, words []string) float64 {
	score := 0.0
	for _, word := range words {
		if idx := strings.Index(keyLower, word); idx != -1 {
			afterMatch := keyLower[idx+len(word):]
			dotCount := strings.Count(afterMatch, ".")
			switch {
			case dotCount == 0:
				score += 10
			case dotCount <= 2:
				score += 6
			case dotCount <= 4:
				score += 2
			}
		}
	}
	return score
}

func subinterfaceScore(query, key string) float64 {
	if strings.Contains(query, "subinterface") && strings.Contains(key, "subinterface") {
		if strings.HasSuffix(key, ".subinterface") {
			return 10
		}
		return 2
	}
	return 0
}

func exactTableScore(key string, words []string) float64 {
	score := 0.0
	for _, w := range words {
		if strings.HasSuffix(key, "."+w) {
			score += 6
		}
	}
	return score
}

func bigramScore(keyLower string, bigrams []string) float64 {
	score := 0.0
	for _, b := range bigrams {
		if strings.Contains(keyLower, strings.ReplaceAll(b, " ", ".")) {
			score += 2
		}
	}
	return score
}

func extractFieldScore(query, key string, entry *models.EmbeddingEntry) (float64, []string) {
	extractedFields := eql.ExtractFields(query, key, entry)
	if len(extractedFields) == 0 {
		return 0, nil
	}
	return float64(len(extractedFields)) * 1.5, extractedFields
}

func sequenceScore(query, key string) float64 {
	if strings.Contains(query, "interface") && strings.Contains(query, "statistics") {
		if strings.Contains(key, "interface") && strings.Contains(key, "statistics") {
			if strings.Contains(key, "interface.statistics") {
				return 8
			}
			return 4
		}
	}
	return 0
}

func pathDepthScore(keyTokens []string) float64 {
	pathDepth := len(keyTokens)
	if pathDepth == 0 {
		return 0
	}

	meaningfulDepth := 0
	for i, token := range keyTokens {
		if i > 2 && token != "state" && token != "configure" {
			meaningfulDepth++
		}
	}

	switch {
	case meaningfulDepth <= 2:
		return 20
	case meaningfulDepth <= 3:
		return 10
	case meaningfulDepth <= 4:
		return 5
	default:
		return -float64(meaningfulDepth-4) * 2
	}
}

func protocolPenalty(queryLower, key string) float64 {
	if strings.Contains(key, "protocols") && !strings.Contains(queryLower, "protocol") &&
		!strings.Contains(queryLower, "bgp") && !strings.Contains(queryLower, "ospf") &&
		!strings.Contains(queryLower, "isis") {
		return -10
	}
	return 0
}

func maintenancePenalty(queryLower, key string) float64 {
	if strings.Contains(key, "maintenance") && !strings.Contains(queryLower, "maintenance") {
		return -8
	}
	return 0
}

func errorQueryScore(queryLower, key string, extractedFields []string) float64 {
	if !strings.Contains(queryLower, "error") {
		return 0
	}
	if strings.Contains(key, "statistics") {
		for _, field := range extractedFields {
			if strings.Contains(field, "error") {
				return 10
			}
		}
	}
	return 0
}

func bandwidthScore(queryLower, key string, extractedFields []string) float64 {
	if strings.Contains(queryLower, "bandwidth") && strings.Contains(key, "interface") {
		for _, field := range extractedFields {
			if strings.Contains(field, "octets") || strings.Contains(field, "bandwidth") {
				return 10
			}
		}
	}
	return 0
}

// scoreEntry calculates the score for a single embedding entry
func (e *Engine) scoreEntry(key string, entry models.EmbeddingEntry, query string, words []string, bigrams []string) float64 {
	keyTokens := Tokenize(key)
	textTokens := Tokenize(entry.ReferenceText + " " + entry.Text)
	queryLower := strings.ToLower(query)

	score, pathMatch := keywordScore(keyTokens, textTokens, words)
	score += descriptionScore(queryLower, entry, words)

	if pathMatch == len(words) && len(words) > 1 {
		score += float64(len(words)) * 3
	}

	if strings.Contains(queryLower, "show") && strings.Contains(key, ".state.") {
		score += 5
	}

	keyLower := strings.ToLower(key)
	score += interfaceScore(key, keyLower, queryLower, words)
	score += bgpScore(queryLower, key)
	score += segmentMatchScore(keyLower, words)
	score += subinterfaceScore(query, key)
	score += exactTableScore(key, words)
	score += bigramScore(keyLower, bigrams)

	fieldScore, extractedFields := extractFieldScore(query, key, &entry)
	score += fieldScore

	score += sequenceScore(query, key)
	score += pathDepthScore(keyTokens)
	score += protocolPenalty(queryLower, key)
	score += maintenancePenalty(queryLower, key)
	score += errorQueryScore(queryLower, key, extractedFields)
	score += bandwidthScore(queryLower, key, extractedFields)

	return score
}
