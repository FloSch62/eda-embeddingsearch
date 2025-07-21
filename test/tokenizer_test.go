// Package test contains unit tests for the search package utilities.
package test

//nolint:misspell // test inputs intentionally contain misspellings

import (
	"reflect"
	"slices"
	"testing"

	"github.com/eda-labs/eda-embeddingsearch/internal/search"
)

func TestTokenize(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{
			name:     "simple words",
			input:    "show interface statistics",
			expected: []string{"show", "interface", "statistics"},
		},
		{
			name:     "with dots and dashes",
			input:    "bgp.neighbor-state",
			expected: []string{"bgp", "neighbor", "state"},
		},
		{
			name:     "with underscores",
			input:    "cpu_usage_percent",
			expected: []string{"cpu", "usage", "percent"},
		},
		{
			name:     "mixed case",
			input:    "Show Interface Statistics",
			expected: []string{"show", "interface", "statistics"},
		},
		{
			name:     "with stop words filtered",
			input:    "show the interface statistics for the router",
			expected: []string{"show", "interface", "statistics", "router"},
		},
		{
			name:     "only stop words",
			input:    "the and or",
			expected: []string{"the", "and", "or"}, // Not filtered when only stop words
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := search.Tokenize(tt.input)
			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("Tokenize(%q) = %v, want %v", tt.input, result, tt.expected)
			}
		})
	}
}

func TestExpandSynonyms(t *testing.T) {
	tests := []struct {
		name     string
		input    []string
		expected []string
	}{
		{
			name:     "basic synonyms",
			input:    []string{"stats", "iface", "temp"},
			expected: []string{"statistics", "interface", "temperature"},
		},
		{
			name: "typo corrections",
			//nolint:misspell // testing intentional misspellings for typo correction
			input:    []string{"interfce", "neighors", "bandwith"},
			expected: []string{"interface", "neighbor", "bandwidth"},
		},
		{
			name:     "no synonyms",
			input:    []string{"show", "system", "version"},
			expected: []string{"show", "system", "version"},
		},
		{
			name:     "mixed synonyms",
			input:    []string{"stats", "show", "intf"},
			expected: []string{"statistics", "show", "interface"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := search.ExpandSynonyms(tt.input)
			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("ExpandSynonyms(%v) = %v, want %v", tt.input, result, tt.expected)
			}
		})
	}
}

func TestContains(t *testing.T) {
	tests := []struct {
		name     string
		tokens   []string
		word     string
		expected bool
	}{
		{
			name:     "word exists",
			tokens:   []string{"show", "interface", "statistics"},
			word:     "interface",
			expected: true,
		},
		{
			name:     "word not exists",
			tokens:   []string{"show", "interface", "statistics"},
			word:     "bgp",
			expected: false,
		},
		{
			name:     "empty tokens",
			tokens:   []string{},
			word:     "test",
			expected: false,
		},
		{
			name:     "case sensitive",
			tokens:   []string{"show", "interface", "statistics"},
			word:     "Interface",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := slices.Contains(tt.tokens, tt.word)
			if result != tt.expected {
				t.Errorf("Contains(%v, %q) = %v, want %v", tt.tokens, tt.word, result, tt.expected)
			}
		})
	}
}
