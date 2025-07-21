package eql

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/eda-labs/eda-embeddingsearch/internal/utils"
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
func ExtractFields(query string, tablePath string, embeddingEntry *models.EmbeddingEntry) []string {
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
					if !utils.Contains(matches, available) {
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
				if !utils.Contains(fields, match) {
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
		// Clean punctuation from word
		w = strings.TrimSuffix(w, "?")
		w = strings.TrimSuffix(w, "!")
		w = strings.TrimSuffix(w, ".")
		w = strings.TrimSuffix(w, ",")

		// Skip generic references
		if w == "nodes" || w == "node" || w == "my" {
			continue
		}

		// Check for specific node patterns
		if (strings.HasPrefix(w, "leaf") || strings.HasPrefix(w, "spine")) && len(w) > 4 {
			lastChar := w[len(w)-1]
			if lastChar >= '0' && lastChar <= '9' {
				return w
			}
		}

		// Check for "on <nodename>" or "for <nodename>" patterns
		if (w == "on" || w == "for" || w == "from") && i+1 < len(words) {
			next := strings.TrimSuffix(words[i+1], "?")
			next = strings.TrimSuffix(next, "!")
			next = strings.TrimSuffix(next, ".")
			next = strings.TrimSuffix(next, ",")
			// Skip common words and protocol names
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
			if !skipWords[next] && len(next) > 1 {
				return next
			}
		}
	}
	return ""
}

// ExtractConditions extracts conditions for WHERE clause
func ExtractConditions(query string, tablePath string) map[string]string {
	conditions := make(map[string]string)
	lower := strings.ToLower(query)

	// State conditions for interfaces
	if strings.Contains(tablePath, "interface") {
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

	// Alarm severity conditions
	if strings.Contains(tablePath, "alarm") {
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

	// Process conditions
	if strings.Contains(tablePath, "process") && strings.Contains(lower, "high memory") {
		conditions["memory-usage-threshold"] = "> 80"
	}

	// Extract numeric comparisons (e.g., "mtu greater than 1500")
	numericPattern := regexp.MustCompile(`(\w+)\s*(greater than|less than|equal to|!=|>=|<=|>|<|=)\s*(\d+)`)
	matches := numericPattern.FindAllStringSubmatch(lower, -1)
	for _, match := range matches {
		field := match[1]
		op := match[2]
		value := match[3]

		// Convert operator
		switch op {
		case "greater than":
			op = ">"
		case "less than":
			op = "<"
		case "equal to":
			op = "="
		}

		conditions[field] = op + " " + value
	}

	return conditions
}

// GenerateWhereClause generates WHERE clause
func GenerateWhereClause(tablePath string, query string) string {
	var whereParts []string

	// Extract node name
	nodeName := ExtractNodeName(query)
	if nodeName != "" && strings.Contains(tablePath, ".namespace.node.") {
		whereParts = append(whereParts, fmt.Sprintf(".namespace.node.name = \"%s\"", nodeName))
	}

	// Extract other conditions
	conditions := ExtractConditions(query, tablePath)
	for field, value := range conditions {
		if strings.HasPrefix(value, ">") || strings.HasPrefix(value, "<") || strings.HasPrefix(value, "=") {
			whereParts = append(whereParts, fmt.Sprintf("%s %s", field, value))
		} else {
			whereParts = append(whereParts, fmt.Sprintf("%s = \"%s\"", field, value))
		}
	}

	if len(whereParts) == 0 {
		return ""
	}

	return strings.Join(whereParts, " and ")
}

// ExtractOrderBy extracts ORDER BY clauses
func ExtractOrderBy(query string, tablePath string, embeddingEntry *models.EmbeddingEntry) []models.OrderByClause {
	var orderBy []models.OrderByClause
	lower := strings.ToLower(query)

	// Get available fields from embedding
	availableFields := ParseEmbeddingText(embeddingEntry.Text)

	// Function to find the best matching field for sorting
	findSortField := func(keywords []string) string {
		for _, keyword := range keywords {
			for _, field := range availableFields {
				if strings.Contains(strings.ToLower(field), keyword) {
					return field
				}
			}
		}
		return ""
	}

	// Common sorting patterns
	if strings.Contains(lower, "top") || strings.Contains(lower, "highest") || strings.Contains(lower, "most") {
		switch {
		case strings.Contains(lower, "memory"):
			// Look for memory-related fields
			sortField := findSortField([]string{"memory-usage", "memory-utilization", "utilization", "used"})
			if sortField != "" {
				orderBy = append(orderBy, models.OrderByClause{Field: sortField, Direction: "descending"})
			}
		case strings.Contains(lower, "cpu"):
			// Look for CPU-related fields
			sortField := findSortField([]string{"cpu-utilization", "cpu-usage", "cpu"})
			if sortField != "" {
				orderBy = append(orderBy, models.OrderByClause{Field: sortField, Direction: "descending"})
			}
		case strings.Contains(lower, "traffic"):
			// Look for traffic-related fields
			sortField := findSortField([]string{"in-octets", "out-octets", "octets"})
			if sortField != "" {
				orderBy = append(orderBy, models.OrderByClause{Field: sortField, Direction: "descending"})
			}
		}
	}

	if strings.Contains(lower, "lowest") || strings.Contains(lower, "least") {
		if strings.Contains(lower, "memory") {
			sortField := findSortField([]string{"memory-usage", "memory-utilization", "utilization", "used"})
			if sortField != "" {
				orderBy = append(orderBy, models.OrderByClause{Field: sortField, Direction: "ascending"})
			}
		}
	}

	// Sort by time for alarms
	if strings.Contains(tablePath, "alarm") && (strings.Contains(lower, "recent") || strings.Contains(lower, "latest")) {
		sortField := findSortField([]string{"time-created", "last-change", "timestamp"})
		if sortField != "" {
			orderBy = append(orderBy, models.OrderByClause{Field: sortField, Direction: "descending"})
		}
	}

	// Natural sorting for names
	if len(orderBy) == 0 && strings.Contains(lower, "sort") {
		sortField := findSortField([]string{"name"})
		if sortField != "" {
			orderBy = append(orderBy, models.OrderByClause{Field: sortField, Direction: "ascending", Algorithm: "natural"})
		}
	}

	return orderBy
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
			if limit, err := strconv.Atoi(matches[1]); err == nil && limit > 0 && limit <= 1000 {
				return limit
			}
		}
	}

	// Default limits for certain queries
	if strings.Contains(lower, "top") || strings.Contains(lower, "highest") {
		return 10
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
			Value: 1,
		}
	}

	return nil
}
