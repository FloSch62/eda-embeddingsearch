package eql

import (
	"encoding/json"
	"fmt"
	"regexp"
	"slices"
	"strconv"
	"strings"

	"github.com/eda-labs/eda-embeddingsearch/internal/constants"
	"github.com/eda-labs/eda-embeddingsearch/pkg/models"
)

// ParseEmbeddingText parses the Text field to get available fields
func ParseEmbeddingText(text string) []string {
	var data struct {
		Fields []string `json:"Fields"`
	}
	if err := json.Unmarshal([]byte(text), &data); err == nil {
		return data.Fields
	}
	return []string{}
}

// ExtractFields extracts fields from natural language
func ExtractFields(query, tablePath string, embeddingEntry *models.EmbeddingEntry) []string {
	fields := []string{}
	lower := strings.ToLower(query)

	// Get available fields from embedding
	availableFields := ParseEmbeddingText(embeddingEntry.Text)

	// Field keywords to field names mapping
	fieldKeywords := map[string][]string{
		"state":       {"admin-state", "oper-state", "state"},
		"status":      {"status", "oper-state", "admin-state"},
		"description": {"description"},
		"name":        {"name"},
		"memory":      {"memory", "memory-usage", "used"},
		"cpu":         {"cpu", "cpu-usage"},
		"traffic":     {"in-octets", "out-octets"},
		"bandwidth":   {"in-octets", "out-octets"},
		"packets":     {"in-packets", "out-packets"},
		"errors":      {"in-error-packets", "out-error-packets", "in-errors", "out-errors"},
		"severity":    {"severity"},
		"time":        {"time-created", "last-change", "last-clear"},
		"octets":      {"in-octets", "out-octets"},
		"mtu":         {"mtu", "ip-mtu", "oper-ip-mtu"},
		"drops":       {"in-drops", "out-drops", "in-discards", "out-discards"},
	}

	// Function to find matching available fields
	findMatchingFields := func(keywords []string) []string {
		var matches []string
		for _, keyword := range keywords {
			for _, available := range availableFields {
				if strings.Contains(strings.ToLower(available), keyword) {
					if !slices.Contains(matches, available) {
						matches = append(matches, available)
					}
				}
			}
		}
		return matches
	}

	// Check for specific field requests based on query keywords
	for keyword, possibleFields := range fieldKeywords {
		if strings.Contains(lower, keyword) {
			matches := findMatchingFields(possibleFields)
			for _, match := range matches {
				if !slices.Contains(fields, match) {
					fields = append(fields, match)
				}
			}
		}
	}

	// Special handling for interface errors when no statistics table
	if strings.Contains(lower, "error") && strings.Contains(tablePath, "interface") && !strings.Contains(tablePath, "statistics") {
		// Suggest looking at statistics if no direct error fields found
		if len(fields) == 0 {
			fields = append(fields, "statistics")
		}
	}

	return fields
}

// ExtractNodeName extracts node name from query
func ExtractNodeName(query string) string {
	words := strings.Fields(strings.ToLower(query))
	for i, w := range words {
		w = cleanPunctuation(w)

		// Skip generic references
		if isGenericNodeReference(w) {
			continue
		}

		// Check for specific node patterns
		if nodeName := checkNodePattern(w); nodeName != "" {
			return nodeName
		}

		// Check for "on <nodename>" or "for <nodename>" patterns
		if nodeName := checkPrepositionPattern(w, i, words); nodeName != "" {
			return nodeName
		}
	}
	return ""
}

func cleanPunctuation(word string) string {
	word = strings.TrimSuffix(word, "?")
	word = strings.TrimSuffix(word, "!")
	word = strings.TrimSuffix(word, ".")
	return strings.TrimSuffix(word, ",")
}

func isGenericNodeReference(word string) bool {
	return word == "nodes" || word == "node" || word == "my"
}

func checkNodePattern(word string) string {
	if (strings.HasPrefix(word, "leaf") || strings.HasPrefix(word, "spine")) && len(word) > 4 {
		lastChar := word[len(word)-1]
		if lastChar >= '0' && lastChar <= '9' {
			return word
		}
	}
	return ""
}

func checkPrepositionPattern(word string, index int, words []string) string {
	if (word == "on" || word == "for" || word == "from") && index+1 < len(words) {
		next := cleanPunctuation(words[index+1])
		if !isSkipWord(next) && len(next) > 1 {
			return next
		}
	}
	return ""
}

func isSkipWord(word string) bool {
	skipWords := map[string]bool{
		"nodes": true, "node": true, "my": true, "the": true,
		"bgp": true, "ospf": true, "isis": true, "mpls": true,
		"interface": true, "interfaces": true, "router": true,
		"system": true, "all": true, "any": true,
		"errors": true, "error": true, "drops": true, "drop": true,
		"statistics": true, "stats": true, "status": true,
		"configuration": true, "config": true, "state": true,
		"up": true, "down": true, "active": true, "inactive": true,
	}
	return skipWords[word]
}

// ExtractConditions extracts conditions for WHERE clause
func ExtractConditions(query, tablePath string) map[string]string {
	conditions := make(map[string]string)
	lower := strings.ToLower(query)

	extractInterfaceConditions(lower, tablePath, conditions)
	extractAlarmConditions(lower, tablePath, conditions)
	extractProcessConditions(lower, tablePath, conditions)
	extractNumericConditions(lower, conditions)

	return conditions
}

func extractInterfaceConditions(lower, tablePath string, conditions map[string]string) {
	if !strings.Contains(tablePath, "interface") {
		return
	}

	if strings.Contains(lower, "up") {
		conditions["oper-state"] = "up"
	} else if strings.Contains(lower, "down") {
		conditions["oper-state"] = "down"
	}

	if strings.Contains(lower, "enabled") {
		conditions["admin-state"] = "enable"
	} else if strings.Contains(lower, "disabled") {
		conditions["admin-state"] = "disable"
	}
}

func extractAlarmConditions(lower, tablePath string, conditions map[string]string) {
	if !strings.Contains(tablePath, "alarm") {
		return
	}

	switch {
	case strings.Contains(lower, "critical"):
		conditions["severity"] = "critical"
	case strings.Contains(lower, "major"):
		conditions["severity"] = "major"
	case strings.Contains(lower, "minor"):
		conditions["severity"] = "minor"
	}

	if strings.Contains(lower, "unacknowledged") || strings.Contains(lower, "not acknowledged") {
		conditions["acknowledged"] = "false"
	}
}

func extractProcessConditions(lower, tablePath string, conditions map[string]string) {
	if strings.Contains(tablePath, "process") && strings.Contains(lower, "high memory") {
		conditions["memory-usage-threshold"] = "> " + strconv.Itoa(constants.DefaultHighMemoryThreshold)
	}
}

func extractNumericConditions(lower string, conditions map[string]string) {
	numericPattern := regexp.MustCompile(`(\w+)\s*(greater than|less than|equal to|!=|>=|<=|>|<|=)\s*(\d+)`)
	matches := numericPattern.FindAllStringSubmatch(lower, -1)

	for _, match := range matches {
		field := match[1]
		op := normalizeOperator(match[2])
		value := match[3]
		conditions[field] = op + " " + value
	}
}

func normalizeOperator(op string) string {
	switch op {
	case "greater than":
		return ">"
	case "less than":
		return "<"
	case "equal to":
		return "="
	default:
		return op
	}
}

// GenerateWhereClause generates WHERE clause
func GenerateWhereClause(tablePath, query string) string {
	var whereParts []string

	// Extract node name
	nodeName := ExtractNodeName(query)
	if nodeName != "" && strings.Contains(tablePath, ".namespace.node.") {
		whereParts = append(whereParts, fmt.Sprintf(".namespace.node.name = %q", nodeName))
	}

	// Extract other conditions
	conditions := ExtractConditions(query, tablePath)
	for field, value := range conditions {
		if strings.HasPrefix(value, ">") || strings.HasPrefix(value, "<") || strings.HasPrefix(value, "=") {
			whereParts = append(whereParts, fmt.Sprintf("%s %s", field, value))
		} else {
			whereParts = append(whereParts, fmt.Sprintf("%s = %q", field, value))
		}
	}

	if len(whereParts) == 0 {
		return ""
	}

	return strings.Join(whereParts, " and ")
}

// ExtractOrderBy extracts ORDER BY clauses
func ExtractOrderBy(query, tablePath string, embeddingEntry *models.EmbeddingEntry) []models.OrderByClause {
	lower := strings.ToLower(query)
	availableFields := ParseEmbeddingText(embeddingEntry.Text)

	fieldFinder := createFieldFinder(availableFields)

	var orderBy []models.OrderByClause

	// Check for descending sort patterns
	orderBy = extractDescendingSort(lower, fieldFinder, orderBy)

	// Check for ascending sort patterns
	orderBy = extractAscendingSort(lower, fieldFinder, orderBy)

	// Check for time-based sorting
	orderBy = extractTimeSort(lower, tablePath, fieldFinder, orderBy)

	// Default natural sorting
	orderBy = extractDefaultSort(lower, fieldFinder, orderBy)

	return orderBy
}

func createFieldFinder(availableFields []string) func([]string) string {
	return func(keywords []string) string {
		for _, keyword := range keywords {
			for _, field := range availableFields {
				if strings.Contains(strings.ToLower(field), keyword) {
					return field
				}
			}
		}
		return ""
	}
}

func extractDescendingSort(lower string, findSortField func([]string) string, orderBy []models.OrderByClause) []models.OrderByClause {
	if !hasDescendingKeywords(lower) {
		return orderBy
	}

	sortConfig := getDescendingSortConfig(lower)
	if sortConfig.keywords != nil {
		if sortField := findSortField(sortConfig.keywords); sortField != "" {
			orderBy = append(orderBy, models.OrderByClause{
				Field:     sortField,
				Direction: "descending",
			})
		}
	}

	return orderBy
}

func extractAscendingSort(lower string, findSortField func([]string) string, orderBy []models.OrderByClause) []models.OrderByClause {
	if !hasAscendingKeywords(lower) {
		return orderBy
	}

	if strings.Contains(lower, "memory") {
		if sortField := findSortField([]string{"memory-usage", "memory-utilization", "utilization", "used"}); sortField != "" {
			orderBy = append(orderBy, models.OrderByClause{
				Field:     sortField,
				Direction: "ascending",
			})
		}
	}

	return orderBy
}

func extractTimeSort(lower, tablePath string, findSortField func([]string) string, orderBy []models.OrderByClause) []models.OrderByClause {
	if !strings.Contains(tablePath, "alarm") {
		return orderBy
	}

	if strings.Contains(lower, "recent") || strings.Contains(lower, "latest") {
		if sortField := findSortField([]string{"time-created", "last-change", "timestamp"}); sortField != "" {
			orderBy = append(orderBy, models.OrderByClause{
				Field:     sortField,
				Direction: "descending",
			})
		}
	}

	return orderBy
}

func extractDefaultSort(lower string, findSortField func([]string) string, orderBy []models.OrderByClause) []models.OrderByClause {
	if len(orderBy) == 0 && strings.Contains(lower, "sort") {
		if sortField := findSortField([]string{"name"}); sortField != "" {
			orderBy = append(orderBy, models.OrderByClause{
				Field:     sortField,
				Direction: "ascending",
				Algorithm: "natural",
			})
		}
	}

	return orderBy
}

func hasDescendingKeywords(lower string) bool {
	return strings.Contains(lower, "top") ||
		strings.Contains(lower, "highest") ||
		strings.Contains(lower, "most")
}

func hasAscendingKeywords(lower string) bool {
	return strings.Contains(lower, "lowest") ||
		strings.Contains(lower, "least")
}

type sortConfig struct {
	keywords []string
}

func getDescendingSortConfig(lower string) sortConfig {
	switch {
	case strings.Contains(lower, "memory"):
		return sortConfig{keywords: []string{"memory-usage", "memory-utilization", "utilization", "used"}}
	case strings.Contains(lower, "cpu"):
		return sortConfig{keywords: []string{"cpu-utilization", "cpu-usage", "cpu"}}
	case strings.Contains(lower, "traffic"):
		return sortConfig{keywords: []string{"in-octets", "out-octets", "octets"}}
	default:
		return sortConfig{}
	}
}

// ExtractLimit extracts LIMIT value
func ExtractLimit(query string) int {
	lower := strings.ToLower(query)

	// Look for "top N" or "first N" patterns
	patterns := []string{
		`top (\d+)`,
		`first (\d+)`,
		`limit (\d+)`,
		`(\d+) results`,
	}

	for _, pattern := range patterns {
		re := regexp.MustCompile(pattern)
		if matches := re.FindStringSubmatch(lower); len(matches) > 1 {
			if limit, err := strconv.Atoi(matches[1]); err == nil && limit > 0 && limit <= constants.MaxLimitValue {
				return limit
			}
		}
	}

	// Default limits for certain queries
	if strings.Contains(lower, "top") || strings.Contains(lower, "highest") {
		return constants.DefaultTopLimit
	}

	return 0
}

// ExtractDelta extracts DELTA clause
func ExtractDelta(query string) *models.DeltaClause {
	lower := strings.ToLower(query)

	// Look for update frequency patterns
	patterns := map[string]*regexp.Regexp{
		"second":      regexp.MustCompile(`every (\d+) seconds?`),
		"millisecond": regexp.MustCompile(`every (\d+) milliseconds?`),
		"realtime":    regexp.MustCompile(`real[\s-]?time|streaming`),
	}

	for unit, pattern := range patterns {
		if matches := pattern.FindStringSubmatch(lower); len(matches) > 1 {
			if value, err := strconv.Atoi(matches[1]); err == nil && value > 0 {
				return &models.DeltaClause{
					Unit:  unit + "s",
					Value: value,
				}
			}
		}
	}

	// Real-time = 1 second updates
	if strings.Contains(lower, "real") && strings.Contains(lower, "time") {
		return &models.DeltaClause{
			Unit:  "seconds",
			Value: constants.RealTimeIntervalSeconds,
		}
	}

	return nil
}
