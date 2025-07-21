package search

import (
	"strings"

	"github.com/eda-labs/eda-embeddingsearch/internal/constants"
)

// Tokenize converts a string into lowercase tokens
func Tokenize(s string) []string {
	s = strings.ToLower(s)
	s = strings.ReplaceAll(s, ".", " ")
	s = strings.ReplaceAll(s, "-", " ")
	s = strings.ReplaceAll(s, "_", " ")

	// Get all tokens
	tokens := strings.Fields(s)

	// Filter out common stop words for better natural language handling
	stopWords := map[string]bool{
		"the": true, "a": true, "an": true, "and": true, "or": true,
		"but": true, "in": true, "on": true, "at": true, "to": true,
		"for": true, "of": true, "with": true, "by": true, "from": true,
		"is": true, "are": true, "was": true, "were": true, "been": true,
		"have": true, "has": true, "had": true, "do": true, "does": true,
		"did": true, "will": true, "would": true, "could": true, "should": true,
		"may": true, "might": true, "must": true, "can": true, "what": true,
		"which": true, "who": true, "when": true, "where": true, "how": true,
		"why": true, "that": true, "this": true, "these": true, "those": true,
		"i": true, "me": true, "my": true, "mine": true, "we": true,
		"us": true, "our": true, "ours": true, "you": true, "your": true,
		"yours": true, "he": true, "him": true, "his": true, "she": true,
		"her": true, "hers": true, "it": true, "its": true, "they": true,
		"them": true, "their": true, "theirs": true,
	}

	// Only filter stop words if we have enough meaningful words
	meaningfulWords := 0
	for _, token := range tokens {
		if !stopWords[token] && len(token) >= constants.MinTokenLength {
			meaningfulWords++
		}
	}

	// If we have at least 2 meaningful words, filter stop words
	if meaningfulWords >= 2 {
		filtered := make([]string, 0, len(tokens))
		for _, token := range tokens {
			if !stopWords[token] || token == "all" || token == "show" || token == "get" || token == "list" {
				filtered = append(filtered, token)
			}
		}
		return filtered
	}

	return tokens
}

// ExpandSynonyms expands words with their synonyms
func ExpandSynonyms(words []string) []string {
	//nolint:misspell // intentionally include common misspellings for expansion
	synonyms := map[string]string{
		"stats":         "statistics",
		"stat":          "statistics",
		"alarms":        "alarm",
		"alarm":         "alarms",
		"fanspeed":      "fan",
		"fan-speed":     "fan",
		"temp":          "temperature",
		"temps":         "temperature",
		"mtu":           "mtu",
		"interswitch":   "link",
		"links":         "link",
		"iface":         "interface",
		"ifaces":        "interface",
		"intf":          "interface",
		"intfs":         "interface",
		"interfaces":    "interface", // Map plural to singular
		"neighbors":     "neighbor",
		"routes":        "route",
		"metrics":       "metric",
		"info":          "information",
		"config":        "configure",
		"configuration": "configure",
		// Common typos
		"inferface":  "interface",
		"inferfaces": "interface",
		"interace":   "interface",
		"intrface":   "interface",
		"interfce":   "interface",
		"interfacs":  "interface",
		"interfaes":  "interface",
		"inerface":   "interface",
		"inerfaces":  "interface",
		"statitics":  "statistics",
		"statsitics": "statistics",
		"statistcs":  "statistics",
		"statistis":  "statistics",
		"neighors":   "neighbor",
		"neigbors":   "neighbor",
		"neighbor":   "neighbor",
		"routers":    "router",
		"sysem":      "system",
		"systm":      "system",
		"bandwith":   "bandwidth",
		"bandwdth":   "bandwidth",
		"alrms":      "alarm",
		"alrm":       "alarm",
		"confg":      "configure",
		"cofig":      "configure",
		"usge":       "usage",
		"useage":     "usage",
		"dwn":        "down",
		"drps":       "drops",
		"drop":       "drops",
	}
	out := make([]string, 0, len(words))
	for _, w := range words {
		if s, ok := synonyms[w]; ok {
			out = append(out, s)
		} else {
			out = append(out, w)
		}
	}
	return out
}
