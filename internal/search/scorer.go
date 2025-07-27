// Package search provides scoring logic to evaluate how well each embedding
// matches a given query.
package search

import (
	"encoding/json"
	"slices"
	"strings"

	"github.com/eda-labs/eda-embeddingsearch/internal/eql"
	"github.com/eda-labs/eda-embeddingsearch/pkg/models"
)

// ScoringRule represents a parameterized scoring rule
type ScoringRule struct {
	Name      string
	CheckFunc func(query, key, keyLower string) bool
	ScoreFunc func(config *ScoringConfig) float64
}

// ConditionalScore applies a score if a condition is met
func (e *Engine) conditionalScore(condition bool, score float64) float64 {
	if condition {
		return score
	}
	return 0
}

// ContainsAllScore returns a score if the text contains all specified substrings
func (e *Engine) containsAllScore(text string, substrings []string, score float64) float64 {
	for _, substr := range substrings {
		if !strings.Contains(text, substr) {
			return 0
		}
	}
	return score
}

// ContainsAnyScore returns a score if the text contains any of the specified substrings
func (e *Engine) containsAnyScore(text string, substrings []string, score float64) float64 {
	for _, substr := range substrings {
		if strings.Contains(text, substr) {
			return score
		}
	}
	return 0
}

// SuffixScore returns a score if the text ends with the specified suffix
func (e *Engine) suffixScore(text, suffix string, score float64) float64 {
	return e.conditionalScore(strings.HasSuffix(text, suffix), score)
}

// CountBasedScore returns a score based on the count of occurrences
func (e *Engine) countBasedScore(count int, thresholds []struct {
	Count int
	Score float64
}) float64 {
	for i := len(thresholds) - 1; i >= 0; i-- {
		if count >= thresholds[i].Count {
			return thresholds[i].Score
		}
	}
	return 0
}

// scoreEntry calculates the relevance score for a candidate entry using
// various heuristics and matching rules.
func (e *Engine) scoreEntry(key string, entry models.EmbeddingEntry, query string, words []string) float64 {
	keyTokens := Tokenize(key)
	textTokens := Tokenize(entry.ReferenceText + " " + entry.Text)
	queryLower := strings.ToLower(query)
	keyLower := strings.ToLower(key)

	score := 0.0

	// Keyword scoring
	score += e.keywordScoreV2(keyTokens, textTokens, words)

	// Description scoring
	score += e.descriptionScoreV2(queryLower, entry, words)

	// Context-based scoring
	score += e.contextScore(queryLower, key, keyLower, words)

	// Field extraction scoring
	extractedFields := eql.ExtractFields(query, key, &entry)
	score += float64(len(extractedFields)) * e.config.FieldExtractScore

	// Special query scoring
	score += e.specialQueryScore(queryLower, key, extractedFields)

	// Path depth scoring
	score += e.pathDepthScore(keyTokens)

	// Penalty scoring
	score += e.penaltyScore(queryLower, key)

	return score
}

// keywordScoreV2 consolidates keyword matching logic
func (e *Engine) keywordScoreV2(keyTokens, textTokens, words []string) float64 {
	score := 0.0
	pathMatchCount := 0

	// Last segment matching
	if len(keyTokens) > 0 && len(words) > 0 {
		lastSegment := keyTokens[len(keyTokens)-1]
		for _, w := range words {
			score += e.conditionalScore(lastSegment == w, e.config.LastSegmentMatch)
		}
	}

	// Word matching with variable scores
	wordScores := map[string]float64{
		"interface":  e.config.KeywordMatchInterface,
		"interfaces": e.config.KeywordMatchInterface,
		"statistics": e.config.KeywordMatchStats,
		"state":      e.config.KeywordMatchState,
		"configure":  e.config.KeywordMatchState,
	}

	for _, w := range words {
		if slices.Contains(keyTokens, w) {
			pathMatchCount++
			if scoreVal, ok := wordScores[w]; ok {
				score += scoreVal
			} else {
				score += e.config.KeywordMatchDefault
			}
		} else if slices.Contains(textTokens, w) {
			score += e.config.TextMatch
		}
	}

	// All words match bonus
	if pathMatchCount == len(words) && len(words) > 1 {
		score += float64(len(words)) * e.config.AllWordsMatchBonus
	}

	return score
}

// descriptionScoreV2 consolidates description matching logic
func (e *Engine) descriptionScoreV2(queryLower string, entry models.EmbeddingEntry, words []string) float64 {
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

	// Count matching words
	descMatchCount := 0
	for _, w := range words {
		if slices.Contains(descTokens, w) {
			descMatchCount++
			score += e.config.DescriptionWordMatch
		}
	}

	// Pattern matching
	patterns := []struct {
		QueryPattern string
		DescPattern  string
		Score        float64
	}{
		{"list of", "list of", e.config.DescriptionListMatch},
		{"all", "all", e.config.DescriptionAllMatch},
		{"show", "display", e.config.DescriptionShowMatch},
		{"get", "retrieve", e.config.DescriptionGetMatch},
	}

	for _, p := range patterns {
		score += e.containsAllScore(queryLower+" "+descLower, []string{p.QueryPattern, p.DescPattern}, p.Score)
	}

	// Multi-match bonus
	score += e.conditionalScore(descMatchCount >= 2 && descMatchCount >= len(words)/2, e.config.DescriptionMultiMatch)

	return score
}

// contextScore handles various context-based scoring rules
func (e *Engine) contextScore(queryLower, key, keyLower string, words []string) float64 {
	score := 0.0

	// Show + state bonus
	score += e.containsAllScore(queryLower+" "+key, []string{"show", ".state."}, e.config.ShowStateBonus)

	// Interface-related scoring
	if strings.Contains(queryLower, "interface") {
		score += e.interfaceScoreV2(key, keyLower, queryLower)
	}

	// BGP-related scoring
	score += e.bgpContextScore(queryLower, key)

	// Segment and suffix matching
	score += e.segmentMatchScoreV2(keyLower, words)
	score += e.suffixMatchScore(key, words)

	// Bigram matching
	score += e.bigramMatchScore(keyLower, words)

	// Sequence matching
	score += e.sequenceMatchScore(queryLower, key)

	// Subinterface matching
	score += e.subinterfaceMatchScore(queryLower, key)

	return score
}

// bgpContextScore handles BGP-specific scoring
func (e *Engine) bgpContextScore(queryLower, key string) float64 {
	if !strings.Contains(queryLower, "bgp") {
		return 0
	}

	score := 0.0

	// Handle BGP neighbor queries - prioritize neighbor table for session queries
	if strings.Contains(queryLower, "neighbor") || strings.Contains(queryLower, "session") || strings.Contains(queryLower, "peer") {
		score += e.containsAllScore(key, []string{"bgp", ".neighbor"}, e.config.BGPNeighborMatch)
		
		// Extra boost for session state queries that should return neighbor table
		if hasSessionStateKeywords(queryLower) && strings.HasSuffix(key, ".neighbor") {
			score += e.config.BGPSessionStateBonus
		}
		
		// Penalty for non-neighbor tables when asking about sessions/neighbors
		if !strings.Contains(key, ".neighbor") && hasSessionStateKeywords(queryLower) {
			score += e.config.BGPNonNeighborPenalty
		}
		
		// Strong penalty for maintenance tables when asking about general sessions
		if strings.Contains(key, "maintenance") && !strings.Contains(queryLower, "maintenance") && hasSessionStateKeywords(queryLower) {
			score += e.config.BGPMaintenanceSessionPenalty
		}
	}
	
	// General BGP scoring for non-neighbor queries
	if strings.Contains(queryLower, "bgp") && !strings.Contains(queryLower, "neighbor") && !strings.Contains(queryLower, "session") {
		score += e.containsAllScore(key, []string{"bgp"}, e.config.BGPGeneralMatch)
	}

	// Maintenance penalty
	score += e.conditionalScore(strings.Contains(key, "maintenance"), e.config.BGPMaintenancePenalty)
	
	return score
}

// hasSessionStateKeywords checks if query has session state related keywords
func hasSessionStateKeywords(queryLower string) bool {
	sessionKeywords := []string{"established", "down", "up", "active", "session", "state", "status"}
	for _, keyword := range sessionKeywords {
		if strings.Contains(queryLower, keyword) {
			return true
		}
	}
	return false
}

// suffixMatchScore calculates score for suffix matches
func (e *Engine) suffixMatchScore(key string, words []string) float64 {
	score := 0.0
	for _, w := range words {
		score += e.suffixScore(key, "."+w, e.config.ExactTableMatch)
	}
	return score
}

// bigramMatchScore calculates score for bigram matches
func (e *Engine) bigramMatchScore(keyLower string, words []string) float64 {
	score := 0.0
	for _, w1 := range words {
		for _, w2 := range words {
			if w1 != w2 {
				bigram := w1 + "." + w2
				score += e.conditionalScore(strings.Contains(keyLower, bigram), e.config.BigramMatch)
			}
		}
	}
	return score
}

// sequenceMatchScore handles sequence-based scoring
func (e *Engine) sequenceMatchScore(queryLower, key string) float64 {
	if !strings.Contains(queryLower, "interface") || !strings.Contains(queryLower, "statistics") {
		return 0
	}
	if strings.Contains(key, "interface.statistics") {
		return e.config.SequenceMatch
	}
	if strings.Contains(key, "interface") && strings.Contains(key, "statistics") {
		return e.config.SequencePartialMatch
	}
	return 0
}

// subinterfaceMatchScore handles subinterface-specific scoring
func (e *Engine) subinterfaceMatchScore(queryLower, key string) float64 {
	if !strings.Contains(queryLower, "subinterface") || !strings.Contains(key, "subinterface") {
		return 0
	}
	score := e.suffixScore(key, ".subinterface", e.config.SubinterfaceExactMatch)
	if !strings.HasSuffix(key, ".subinterface") {
		score += e.config.SubinterfacePartialMatch
	}
	return score
}

// interfaceScoreV2 consolidated interface scoring
func (e *Engine) interfaceScoreV2(key, keyLower, queryLower string) float64 {
	score := 0.0

	// Security penalty
	score += e.containsAnyScore(keyLower, []string{"violator", "security"}, e.config.InterfaceSecurityPenalty)

	// Path scoring
	if strings.HasSuffix(key, ".interface") && !strings.Contains(key, ".protocols.") {
		score += e.config.InterfaceEndMatch
	}
	if strings.Contains(queryLower, "statistics") && strings.HasSuffix(key, ".interface.statistics") {
		score += e.config.InterfaceStatsMatch
	}
	if strings.Contains(queryLower, "interfaces") && strings.HasSuffix(key, ".interface") {
		score += e.config.InterfacePluralMatch
	}

	// Protocol penalty
	protocolsInQuery := e.containsAnyScore(queryLower, []string{"bgp", "ospf", "isis"}, 1.0) > 0
	protocolsInKey := e.containsAnyScore(keyLower, []string{"protocols.bgp", "protocols.ospf", "protocols.isis"}, 1.0) > 0
	if !protocolsInQuery && protocolsInKey {
		score += e.config.InterfaceProtocolPenalty
	}

	return score
}

// segmentMatchScoreV2 consolidated segment matching
func (e *Engine) segmentMatchScoreV2(keyLower string, words []string) float64 {
	score := 0.0
	for _, word := range words {
		if idx := strings.Index(keyLower, word); idx != -1 {
			afterMatch := keyLower[idx+len(word):]
			dotCount := strings.Count(afterMatch, ".")

			thresholds := []struct {
				Count int
				Score float64
			}{
				{0, e.config.SegmentExactMatch},
				{1, e.config.SegmentNearMatch},
				{3, e.config.SegmentFarMatch},
			}

			score += e.countBasedScore(-dotCount, thresholds)
		}
	}
	return score
}

// specialQueryScore handles special query patterns
func (e *Engine) specialQueryScore(queryLower, key string, extractedFields []string) float64 {
	score := 0.0

	// Error query scoring
	if strings.Contains(queryLower, "error") && strings.Contains(key, "statistics") {
		for _, field := range extractedFields {
			if strings.Contains(field, "error") {
				score += e.config.ErrorFieldBonus
				break
			}
		}
	}

	// Bandwidth query scoring
	if strings.Contains(queryLower, "bandwidth") && strings.Contains(key, "interface") {
		for _, field := range extractedFields {
			if strings.Contains(field, "octets") || strings.Contains(field, "bandwidth") {
				score += e.config.BandwidthFieldBonus
				break
			}
		}
	}

	return score
}

// penaltyScore applies various penalties
func (e *Engine) penaltyScore(queryLower, key string) float64 {
	score := 0.0

	// Protocol penalty
	if strings.Contains(key, "protocols") &&
		!strings.Contains(queryLower, "protocol") &&
		e.containsAnyScore(queryLower, []string{"bgp", "ospf", "isis"}, 1.0) == 0 {
		score += e.config.ProtocolPenalty
	}

	// Maintenance penalty
	if strings.Contains(key, "maintenance") && !strings.Contains(queryLower, "maintenance") {
		score += e.config.MaintenancePenalty
	}

	return score
}

// pathDepthScore calculates score based on path depth
func (e *Engine) pathDepthScore(keyTokens []string) float64 {
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
		return e.config.PathDepthBonus2
	case meaningfulDepth <= 3:
		return e.config.PathDepthBonus3
	case meaningfulDepth <= 4:
		return e.config.PathDepthBonus4
	default:
		return -float64(meaningfulDepth-4) * e.config.PathDepthPenaltyFactor
	}
}
