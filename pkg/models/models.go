package models

import "encoding/json"

// EmbeddingEntry represents a single embedding with its vector and metadata
type EmbeddingEntry struct {
	Vector        []float64 `json:"Vector"`
	ReferenceText string    `json:"ReferenceText"`
	Text          string    `json:"Text"`
}

// EmbeddingDB represents the database of embeddings
type EmbeddingDB struct {
	Table         map[string]EmbeddingEntry `json:"Table"`
	InvertedIndex map[string][]string       `json:"-"` // word -> list of keys containing that word
}

// EQLQuery represents an EQL query with all its components
type EQLQuery struct {
	Table       string
	Fields      []string
	WhereClause string
	OrderBy     []OrderByClause
	Limit       int
	Delta       *DeltaClause
}

// OrderByClause represents an ORDER BY component
type OrderByClause struct {
	Field     string
	Direction string // ascending/descending
	Algorithm string // natural (optional)
}

// DeltaClause represents a DELTA component for streaming
type DeltaClause struct {
	Unit  string // milliseconds, seconds, etc.
	Value int
}

// SearchResult represents a search result with scoring and EQL query
type SearchResult struct {
	Key             string
	Score           float64
	EQLQuery        EQLQuery
	Description     string
	AvailableFields []string
}

// MarshalJSON customizes the JSON output for SearchResult
func (sr *SearchResult) MarshalJSON() ([]byte, error) {
	// Create a custom struct that matches the desired JSON format
	type jsonResult struct {
		Score           float64  `json:"score"`
		Query           string   `json:"query"`
		Table           string   `json:"table"`
		Description     string   `json:"description,omitempty"`
		AvailableFields []string `json:"availableFields,omitempty"`
		Fields          []string `json:"fields,omitempty"`
		Where           string   `json:"where,omitempty"`
		OrderBy         []struct {
			Field     string `json:"field"`
			Direction string `json:"direction"`
			Algorithm string `json:"algorithm,omitempty"`
		} `json:"orderBy,omitempty"`
		Limit int `json:"limit,omitempty"`
		Delta *struct {
			Unit  string `json:"unit"`
			Value int    `json:"value"`
		} `json:"delta,omitempty"`
	}

	result := jsonResult{
		Score:           sr.Score,
		Query:           sr.EQLQuery.String(),
		Table:           sr.EQLQuery.Table,
		Description:     sr.Description,
		AvailableFields: sr.AvailableFields,
		Fields:          sr.EQLQuery.Fields,
		Where:           sr.EQLQuery.WhereClause,
		Limit:           sr.EQLQuery.Limit,
	}

	// Convert OrderBy
	if len(sr.EQLQuery.OrderBy) > 0 {
		result.OrderBy = make([]struct {
			Field     string `json:"field"`
			Direction string `json:"direction"`
			Algorithm string `json:"algorithm,omitempty"`
		}, len(sr.EQLQuery.OrderBy))

		for i, ob := range sr.EQLQuery.OrderBy {
			result.OrderBy[i].Field = ob.Field
			result.OrderBy[i].Direction = ob.Direction
			result.OrderBy[i].Algorithm = ob.Algorithm
		}
	}

	// Convert Delta
	if sr.EQLQuery.Delta != nil {
		result.Delta = &struct {
			Unit  string `json:"unit"`
			Value int    `json:"value"`
		}{
			Unit:  sr.EQLQuery.Delta.Unit,
			Value: sr.EQLQuery.Delta.Value,
		}
	}

	return json.Marshal(result)
}

// EmbeddingType represents the type of embeddings to use
type EmbeddingType int

const (
	SRL EmbeddingType = iota
	SROS
)
