// Package eql contains field mapping configurations for natural language to EQL conversion
package eql

import (
	"regexp"
	"strings"
)

// FieldMapping represents a mapping from natural language patterns to field conditions
type FieldMapping struct {
	// Patterns that trigger this mapping (case-insensitive)
	Patterns []string
	// Target field name
	FieldName string
	// Value to assign (can be literal or pattern-based)
	Value string
	// Regex pattern for extracting values from query (optional)
	ValuePattern *regexp.Regexp
	// Tables where this field is valid (empty means all tables)
	ValidTables []string
	// Whether this mapping requires the table path to contain certain keywords
	RequiredTableKeywords []string
}

// ConditionalMapping represents conditional field mappings
type ConditionalMapping struct {
	// Condition that must be met for this mapping to apply
	Condition func(query, tablePath string) bool
	// Mappings that apply when condition is met
	Mappings []FieldMapping
}

// GetFieldMappings returns all configured field mappings
func GetFieldMappings() []FieldMapping {
	return []FieldMapping{
		// === INTERFACE STATE MAPPINGS ===
		{
			Patterns:              []string{"up", "operational", "active"},
			FieldName:             "oper-state",
			Value:                 "up",
			RequiredTableKeywords: []string{"interface"},
		},
		{
			Patterns:              []string{"down", "failed", "inactive"},
			FieldName:             "oper-state",
			Value:                 "down",
			RequiredTableKeywords: []string{"interface"},
		},
		{
			Patterns:              []string{"enabled", "enable"},
			FieldName:             "admin-state",
			Value:                 "enable",
			RequiredTableKeywords: []string{"interface"},
		},
		{
			Patterns:              []string{"disabled", "disable"},
			FieldName:             "admin-state",
			Value:                 "disable",
			RequiredTableKeywords: []string{"interface"},
		},

		// === PORT SPEED MAPPINGS ===
		{
			Patterns:              []string{"400g", "400gbps", "400 gbps"},
			FieldName:             "port-speed",
			Value:                 "400G",
			RequiredTableKeywords: []string{"ethernet", "interface"},
		},
		{
			Patterns:              []string{"100g", "100gbps", "100 gbps"},
			FieldName:             "port-speed",
			Value:                 "100G",
			RequiredTableKeywords: []string{"ethernet", "interface"},
		},
		{
			Patterns:              []string{"50g", "50gbps", "50 gbps"},
			FieldName:             "port-speed",
			Value:                 "50G",
			RequiredTableKeywords: []string{"ethernet", "interface"},
		},
		{
			Patterns:              []string{"40g", "40gbps", "40 gbps"},
			FieldName:             "port-speed",
			Value:                 "40G",
			RequiredTableKeywords: []string{"ethernet", "interface"},
		},
		{
			Patterns:              []string{"25g", "25gbps", "25 gbps"},
			FieldName:             "port-speed",
			Value:                 "25G",
			RequiredTableKeywords: []string{"ethernet", "interface"},
		},
		{
			Patterns:              []string{"10g", "10gbps", "10 gbps"},
			FieldName:             "port-speed",
			Value:                 "10G",
			RequiredTableKeywords: []string{"ethernet", "interface"},
		},
		{
			Patterns:              []string{"1g", "1gbps", "1 gbps", "gigabit"},
			FieldName:             "port-speed",
			Value:                 "1G",
			RequiredTableKeywords: []string{"ethernet", "interface"},
		},

		// === PHYSICAL MEDIUM MAPPINGS ===
		{
			Patterns:              []string{"fiber", "optical", "sfp", "qsfp"},
			FieldName:             "physical-medium",
			Value:                 "fiber",
			RequiredTableKeywords: []string{"ethernet", "interface"},
		},
		{
			Patterns:              []string{"copper", "dac", "electrical"},
			FieldName:             "physical-medium",
			Value:                 "copper",
			RequiredTableKeywords: []string{"ethernet", "interface"},
		},

		// === ETHERNET PMD MAPPINGS ===
		{
			Patterns:              []string{"copper", "electrical"},
			FieldName:             "ethernet-pmd",
			Value:                 "~ \"BASE-T\"",
			RequiredTableKeywords: []string{"transceiver", "ethernet"},
		},
		{
			Patterns:              []string{"fiber", "optical"},
			FieldName:             "ethernet-pmd",
			Value:                 "!~ \"BASE-T\"",
			RequiredTableKeywords: []string{"transceiver", "ethernet"},
		},

		// === VLAN MAPPINGS ===
		{
			Patterns:              []string{"vlan tagging enabled", "vlan-tagging", "tagged"},
			FieldName:             "vlan-tagging",
			Value:                 "true",
			RequiredTableKeywords: []string{"interface"},
		},
		{
			Patterns:              []string{"vlan tagging disabled", "no vlan", "untagged"},
			FieldName:             "vlan-tagging",
			Value:                 "false",
			RequiredTableKeywords: []string{"interface"},
		},

		// === TRANSCEIVER FORM FACTOR MAPPINGS ===
		{
			Patterns:              []string{"qsfp28"},
			FieldName:             "form-factor",
			Value:                 "QSFP28",
			RequiredTableKeywords: []string{"transceiver"},
		},
		{
			Patterns:              []string{"qsfp+"},
			FieldName:             "form-factor",
			Value:                 "QSFP+",
			RequiredTableKeywords: []string{"transceiver"},
		},
		{
			Patterns:              []string{"qsfp"},
			FieldName:             "form-factor",
			Value:                 "QSFP",
			RequiredTableKeywords: []string{"transceiver"},
		},
		{
			Patterns:              []string{"sfp+"},
			FieldName:             "form-factor",
			Value:                 "SFP+",
			RequiredTableKeywords: []string{"transceiver"},
		},
		{
			Patterns:              []string{"sfp"},
			FieldName:             "form-factor",
			Value:                 "SFP",
			RequiredTableKeywords: []string{"transceiver"},
		},
		{
			Patterns:              []string{"cfp"},
			FieldName:             "form-factor",
			Value:                 "CFP",
			RequiredTableKeywords: []string{"transceiver"},
		},
		{
			Patterns:              []string{"xfp"},
			FieldName:             "form-factor",
			Value:                 "XFP",
			RequiredTableKeywords: []string{"transceiver"},
		},

		// === LAG MAPPINGS ===
		{
			Patterns:              []string{"lag members", "lag member"},
			FieldName:             "aggregate-id",
			Value:                 "!= null",
			RequiredTableKeywords: []string{"ethernet"},
		},
		{
			Patterns:              []string{"lacp"},
			FieldName:             "lacp-mode",
			Value:                 "active",
			RequiredTableKeywords: []string{"lag"},
		},
		{
			Patterns:              []string{"static lag"},
			FieldName:             "lag-type",
			Value:                 "static",
			RequiredTableKeywords: []string{"lag"},
		},
		{
			Patterns:              []string{"dynamic lag"},
			FieldName:             "lag-type",
			Value:                 "lacp",
			RequiredTableKeywords: []string{"lag"},
		},

		// === BGP SESSION STATE MAPPINGS ===
		{
			Patterns:              []string{"established"},
			FieldName:             "session-state",
			Value:                 "established",
			RequiredTableKeywords: []string{"bgp", "neighbor"},
		},
		{
			Patterns:              []string{"idle"},
			FieldName:             "session-state",
			Value:                 "idle",
			RequiredTableKeywords: []string{"bgp", "neighbor"},
		},
		{
			Patterns:              []string{"active"},
			FieldName:             "session-state",
			Value:                 "active",
			RequiredTableKeywords: []string{"bgp", "neighbor"},
		},
		{
			Patterns:              []string{"connect"},
			FieldName:             "session-state",
			Value:                 "connect",
			RequiredTableKeywords: []string{"bgp", "neighbor"},
		},
		{
			Patterns:              []string{"opensent"},
			FieldName:             "session-state",
			Value:                 "opensent",
			RequiredTableKeywords: []string{"bgp", "neighbor"},
		},
		{
			Patterns:              []string{"openconfirm"},
			FieldName:             "session-state",
			Value:                 "openconfirm",
			RequiredTableKeywords: []string{"bgp", "neighbor"},
		},

		// === BGP PEER TYPE MAPPINGS ===
		{
			Patterns:              []string{"ebgp", "external"},
			FieldName:             "peer-type",
			Value:                 "external",
			RequiredTableKeywords: []string{"bgp", "neighbor"},
		},
		{
			Patterns:              []string{"ibgp", "internal"},
			FieldName:             "peer-type",
			Value:                 "internal",
			RequiredTableKeywords: []string{"bgp", "neighbor"},
		},

		// === CONNECTOR TYPE MAPPINGS ===
		{
			Patterns:              []string{"lc connector", "lc"},
			FieldName:             "connector-type",
			Value:                 "LC",
			RequiredTableKeywords: []string{"transceiver"},
		},
		{
			Patterns:              []string{"mpo", "mtp"},
			FieldName:             "connector-type",
			Value:                 "MPO",
			RequiredTableKeywords: []string{"transceiver"},
		},
	}
}

// GetRegexMappings returns mappings that use regex for value extraction
func GetRegexMappings() []FieldMapping {
	return []FieldMapping{
		// VLAN ID extraction - "vlan 100", "vlan id 200"
		{
			Patterns:              []string{"vlan"},
			FieldName:             "vlan-id",
			ValuePattern:          regexp.MustCompile(`vlan\s+(?:id\s+)?(\d+)`),
			RequiredTableKeywords: []string{"vlan"},
		},
		// LAG ID extraction - "lag1", "lag 2"
		{
			Patterns:              []string{"lag"},
			FieldName:             "aggregate-id",
			ValuePattern:          regexp.MustCompile(`lag\s*(\d+)`),
			RequiredTableKeywords: []string{"ethernet"},
		},
		// AS number extraction - "AS 65001", "as number 65002"
		{
			Patterns:              []string{"as ", "as number"},
			FieldName:             "peer-as",
			ValuePattern:          regexp.MustCompile(`as\s+(?:number\s+)?(\d+)`),
			RequiredTableKeywords: []string{"bgp"},
		},
		// MTU extraction - "mtu 9000", "mtu 1500"
		{
			Patterns:              []string{"mtu"},
			FieldName:             "mtu",
			ValuePattern:          regexp.MustCompile(`mtu\s+(\d+)`),
			RequiredTableKeywords: []string{"interface"},
		},
	}
}

// GetConditionalMappings returns mappings that depend on context
func GetConditionalMappings() []ConditionalMapping {
	return []ConditionalMapping{
		// Special handling for "down" in BGP context
		{
			Condition: func(query, tablePath string) bool {
				return strings.Contains(strings.ToLower(query), "down") &&
					strings.Contains(tablePath, "bgp") &&
					strings.Contains(tablePath, "neighbor") &&
					!strings.Contains(strings.ToLower(query), "established")
			},
			Mappings: []FieldMapping{
				{
					FieldName: "session-state",
					Value:     "!= \"established\"",
				},
			},
		},
	}
}

// FieldKeywordMappings returns enhanced field keyword mappings for field extraction
func FieldKeywordMappings() map[string][]string {
	return map[string][]string{
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

		// Enhanced networking field mappings
		"speed":       {"port-speed", "lag-speed", "member-speed"},
		"vlan":        {"vlan-tagging", "vlan-id", "tpid"},
		"lag":         {"aggregate-id", "lag-type", "lacp-mode"},
		"optical":     {"form-factor", "connector-type", "wavelength"},
		"transceiver": {"form-factor", "vendor", "serial-number"},
		"fiber":       {"physical-medium", "connector-type", "wavelength"},
		"copper":      {"physical-medium", "ethernet-pmd"},
		"sfp":         {"form-factor", "vendor-part-number"},
		"mac":         {"hw-mac-address", "system-id-mac"},
		"power":       {"input-power", "output-power", "laser-bias-current"},
		"vendor":      {"vendor", "vendor-part-number", "vendor-serial-number"},
		"aggregate":   {"aggregate-id", "lag-type", "min-links"},
		"lacp":        {"lacp-mode", "lacp-port-priority", "interval"},
		"tagged":      {"vlan-tagging", "vlan-id"},
		"physical":    {"physical-medium", "linecard", "forwarding-complex"},
		"hardware":    {"hw-mac-address", "form-factor", "vendor"},
	}
}
