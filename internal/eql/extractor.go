// Package eql contains helpers for extracting fields from embedding entries
// and constructing EQL statements.
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

	// Use field keywords mapping from configuration
	fieldKeywords := FieldKeywordMappings()

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

// ExtractNodeNames extracts all node names from query for multi-node support
func ExtractNodeNames(query string) []string {
	var nodeNames []string
	words := strings.Fields(strings.ToLower(query))
	
	for i, w := range words {
		w = cleanPunctuation(w)

		// Skip generic references
		if isGenericNodeReference(w) {
			continue
		}

		// Check for specific node patterns
		if nodeName := checkNodePattern(w); nodeName != "" {
			nodeNames = append(nodeNames, nodeName)
		}

		// Check for "on <nodename>" or "for <nodename>" patterns
		if nodeName := checkPrepositionPattern(w, i, words); nodeName != "" {
			nodeNames = append(nodeNames, nodeName)
		}
	}
	
	// Remove duplicates
	uniqueNodes := make([]string, 0, len(nodeNames))
	seen := make(map[string]bool)
	for _, node := range nodeNames {
		if !seen[node] {
			uniqueNodes = append(uniqueNodes, node)
			seen[node] = true
		}
	}
	
	return uniqueNodes
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

// ExtractConditions extracts conditions for WHERE clause using dictionary-based approach
func ExtractConditions(query, tablePath string) map[string]string {
	conditions := make(map[string]string)
	lower := strings.ToLower(query)

	// Apply standard field mappings
	applyFieldMappings(lower, tablePath, conditions)
	
	// Apply regex-based mappings for value extraction
	applyRegexMappings(lower, tablePath, conditions)
	
	// Apply conditional mappings based on context
	applyConditionalMappings(lower, tablePath, conditions)
	
	// Fallback to legacy extraction for uncovered cases
	extractNumericConditions(lower, conditions)

	return conditions
}

// applyFieldMappings applies standard field mappings from configuration
func applyFieldMappings(lower, tablePath string, conditions map[string]string) {
	mappings := GetFieldMappings()
	
	for _, mapping := range mappings {
		// Check if this mapping applies to the current table
		if !isValidForTable(mapping, tablePath) {
			continue
		}
		
		// Check if any pattern matches the query
		for _, pattern := range mapping.Patterns {
			if strings.Contains(lower, strings.ToLower(pattern)) {
				conditions[mapping.FieldName] = mapping.Value
				break // Only apply first matching pattern for this mapping
			}
		}
	}
}

// applyRegexMappings applies regex-based mappings for value extraction
func applyRegexMappings(lower, tablePath string, conditions map[string]string) {
	mappings := GetRegexMappings()
	
	for _, mapping := range mappings {
		// Check if this mapping applies to the current table
		if !isValidForTable(mapping, tablePath) {
			continue
		}
		
		// Check if any pattern matches and extract value
		for _, pattern := range mapping.Patterns {
			if strings.Contains(lower, strings.ToLower(pattern)) {
				if mapping.ValuePattern != nil {
					if matches := mapping.ValuePattern.FindStringSubmatch(lower); len(matches) > 1 {
						conditions[mapping.FieldName] = matches[1]
					}
				}
				break
			}
		}
	}
}

// applyConditionalMappings applies context-dependent mappings
func applyConditionalMappings(lower, tablePath string, conditions map[string]string) {
	mappings := GetConditionalMappings()
	
	for _, mapping := range mappings {
		if mapping.Condition(lower, tablePath) {
			for _, fieldMapping := range mapping.Mappings {
				conditions[fieldMapping.FieldName] = fieldMapping.Value
			}
		}
	}
}

// isValidForTable checks if a field mapping is valid for the given table
func isValidForTable(mapping FieldMapping, tablePath string) bool {
	// If no table restrictions, it's valid for all tables
	if len(mapping.ValidTables) == 0 && len(mapping.RequiredTableKeywords) == 0 {
		return true
	}
	
	// Check if table path matches any valid table patterns
	tablePathLower := strings.ToLower(tablePath)
	for _, validTable := range mapping.ValidTables {
		if strings.Contains(tablePathLower, strings.ToLower(validTable)) {
			return true
		}
	}
	
	// Check if table path contains all required keywords
	if len(mapping.RequiredTableKeywords) > 0 {
		hasAllKeywords := true
		for _, keyword := range mapping.RequiredTableKeywords {
			if !strings.Contains(tablePathLower, strings.ToLower(keyword)) {
				hasAllKeywords = false
				break
			}
		}
		if hasAllKeywords {
			return true
		}
	}
	
	return false
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

func extractEthernetConditions(lower, tablePath string, conditions map[string]string) {
	if !strings.Contains(tablePath, "ethernet") && !strings.Contains(tablePath, "interface") {
		return
	}

	// Port speed extraction - handle various speed formats
	speedPatterns := map[string]string{
		"100g":     "100G",
		"100gbps":  "100G",
		"100 gbps": "100G",
		"40g":      "40G",
		"40gbps":   "40G",
		"25g":      "25G",
		"25gbps":   "25G",
		"10g":      "10G",
		"10gbps":   "10G",
		"1g":       "1G",
		"1gbps":    "1G",
		"gigabit":  "1G",
		"400g":     "400G",
		"400gbps":  "400G",
		"50g":      "50G",
		"50gbps":   "50G",
	}

	for pattern, speed := range speedPatterns {
		if strings.Contains(lower, pattern) {
			conditions["port-speed"] = speed
			break
		}
	}

	// Physical medium extraction
	if strings.Contains(lower, "fiber") || strings.Contains(lower, "optical") {
		conditions["physical-medium"] = "fiber"
	} else if strings.Contains(lower, "copper") || strings.Contains(lower, "dac") {
		conditions["physical-medium"] = "copper"
	}
}

func extractVLANConditions(lower, tablePath string, conditions map[string]string) {
	if !strings.Contains(tablePath, "interface") && !strings.Contains(tablePath, "vlan") {
		return
	}

	// VLAN tagging conditions
	if strings.Contains(lower, "vlan tagging enabled") || strings.Contains(lower, "vlan-tagging") {
		conditions["vlan-tagging"] = "true"
	} else if strings.Contains(lower, "vlan tagging disabled") || strings.Contains(lower, "no vlan") {
		conditions["vlan-tagging"] = "false"
	}

	// VLAN ID extraction - look for patterns like "vlan 100", "vlan id 200"
	vlanPattern := regexp.MustCompile(`vlan\s+(?:id\s+)?(\d+)`)
	if matches := vlanPattern.FindStringSubmatch(lower); len(matches) > 1 {
		conditions["vlan-id"] = matches[1]
	}

	// Tagged/untagged conditions
	if strings.Contains(lower, "tagged") && !strings.Contains(lower, "untagged") {
		conditions["vlan-tagging"] = "true"
	} else if strings.Contains(lower, "untagged") {
		conditions["vlan-tagging"] = "false"
	}
}

func extractLAGConditions(lower, tablePath string, conditions map[string]string) {
	if !strings.Contains(tablePath, "lag") && !strings.Contains(tablePath, "interface") {
		return
	}

	// LAG membership conditions
	if strings.Contains(lower, "lag members") || strings.Contains(lower, "lag member") {
		// This suggests we want interfaces that are members of LAGs
		conditions["aggregate-id"] = "!= null"
	}

	// Specific LAG conditions
	lagPattern := regexp.MustCompile(`lag\s*(\d+)`)
	if matches := lagPattern.FindStringSubmatch(lower); len(matches) > 1 {
		conditions["aggregate-id"] = "lag" + matches[1]
	}

	// LACP conditions
	if strings.Contains(lower, "lacp") {
		conditions["lacp-mode"] = "active"
	}

	// LAG type conditions
	if strings.Contains(lower, "static lag") {
		conditions["lag-type"] = "static"
	} else if strings.Contains(lower, "dynamic lag") {
		conditions["lag-type"] = "lacp"
	}
}

func extractTransceiverConditions(lower, tablePath string, conditions map[string]string) {
	if !strings.Contains(tablePath, "transceiver") && !strings.Contains(tablePath, "interface") {
		return
	}

	// Form factor conditions
	formFactorPatterns := map[string]string{
		"sfp+":   "SFP+",
		"sfp":    "SFP",
		"qsfp28": "QSFP28",
		"qsfp+":  "QSFP+",
		"qsfp":   "QSFP",
		"cfp":    "CFP",
		"xfp":    "XFP",
	}

	for pattern, formFactor := range formFactorPatterns {
		if strings.Contains(lower, pattern) {
			conditions["form-factor"] = formFactor
			break
		}
	}

	// Connector type conditions
	if strings.Contains(lower, "lc connector") || strings.Contains(lower, "lc") {
		conditions["connector-type"] = "LC"
	} else if strings.Contains(lower, "mpo") || strings.Contains(lower, "mtp") {
		conditions["connector-type"] = "MPO"
	}

	// Optical/electrical detection
	if strings.Contains(lower, "optical") || strings.Contains(lower, "fiber") {
		conditions["ethernet-pmd"] = "!~ \"BASE-T\""
	} else if strings.Contains(lower, "electrical") || strings.Contains(lower, "copper") {
		conditions["ethernet-pmd"] = "~ \"BASE-T\""
	}
}

func extractBGPConditions(lower, tablePath string, conditions map[string]string) {
	if !strings.Contains(tablePath, "bgp") {
		return
	}

	// BGP session state conditions
	if strings.Contains(tablePath, "neighbor") {
		if strings.Contains(lower, "established") {
			conditions["session-state"] = "established"
		} else if strings.Contains(lower, "idle") {
			conditions["session-state"] = "idle"
		} else if strings.Contains(lower, "active") {
			conditions["session-state"] = "active"
		} else if strings.Contains(lower, "connect") {
			conditions["session-state"] = "connect"
		} else if strings.Contains(lower, "opensent") {
			conditions["session-state"] = "opensent"
		} else if strings.Contains(lower, "openconfirm") {
			conditions["session-state"] = "openconfirm"
		}
		
		// Handle "down" as a general term for non-established sessions
		if strings.Contains(lower, "down") && !strings.Contains(lower, "established") {
			// Use != established to catch any non-established state
			conditions["session-state"] = "!= \"established\""
		}
		
		// BGP peer type conditions
		if strings.Contains(lower, "ebgp") || strings.Contains(lower, "external") {
			conditions["peer-type"] = "external"
		} else if strings.Contains(lower, "ibgp") || strings.Contains(lower, "internal") {
			conditions["peer-type"] = "internal"
		}
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

// GenerateWhereClause generates WHERE clause with field validation
func GenerateWhereClause(tablePath, query string) string {
	var whereParts []string

	// Extract node names (support multiple nodes)
	nodeNames := ExtractNodeNames(query)
	if len(nodeNames) > 0 && strings.Contains(tablePath, ".namespace.node.") {
		if len(nodeNames) == 1 {
			whereParts = append(whereParts, fmt.Sprintf(".namespace.node.name = %q", nodeNames[0]))
		} else {
			// Multiple nodes: use IN clause
			nodeList := make([]string, len(nodeNames))
			for i, name := range nodeNames {
				nodeList[i] = fmt.Sprintf("%q", name)
			}
			whereParts = append(whereParts, fmt.Sprintf(".namespace.node.name in [%s]", strings.Join(nodeList, ", ")))
		}
	}

	// Extract other conditions
	conditions := ExtractConditions(query, tablePath)
	for field, value := range conditions {
		if strings.HasPrefix(value, ">") || strings.HasPrefix(value, "<") || strings.HasPrefix(value, "=") || strings.HasPrefix(value, "!") {
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

// GenerateWhereClauseWithValidation generates WHERE clause with field validation
func GenerateWhereClauseWithValidation(tablePath, query string, availableFields []string) string {
	var whereParts []string

	// Extract node names (support multiple nodes)
	nodeNames := ExtractNodeNames(query)
	if len(nodeNames) > 0 && strings.Contains(tablePath, ".namespace.node.") {
		if len(nodeNames) == 1 {
			whereParts = append(whereParts, fmt.Sprintf(".namespace.node.name = %q", nodeNames[0]))
		} else {
			// Multiple nodes: use IN clause
			nodeList := make([]string, len(nodeNames))
			for i, name := range nodeNames {
				nodeList[i] = fmt.Sprintf("%q", name)
			}
			whereParts = append(whereParts, fmt.Sprintf(".namespace.node.name in [%s]", strings.Join(nodeList, ", ")))
		}
	}

	// Extract other conditions and validate against available fields
	conditions := ExtractConditions(query, tablePath)
	for field, value := range conditions {
		// Check if field exists in available fields
		fieldExists := false
		for _, availableField := range availableFields {
			if availableField == field {
				fieldExists = true
				break
			}
		}
		
		// Only add condition if field exists in the table
		if fieldExists {
			if strings.HasPrefix(value, ">") || strings.HasPrefix(value, "<") || strings.HasPrefix(value, "=") || strings.HasPrefix(value, "!") {
				whereParts = append(whereParts, fmt.Sprintf("%s %s", field, value))
			} else {
				whereParts = append(whereParts, fmt.Sprintf("%s = %q", field, value))
			}
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
