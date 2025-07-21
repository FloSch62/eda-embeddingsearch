package models

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

// EmbeddingType represents the type of embeddings to use
type EmbeddingType int

const (
	SRL EmbeddingType = iota
	SROS
)
