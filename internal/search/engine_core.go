package search

import (
	"github.com/eda-labs/eda-embeddingsearch/pkg/models"
)

// Engine represents the search engine
type Engine struct {
	db     *models.EmbeddingDB
	config *ScoringConfig
}

// NewEngine creates a new search engine
func NewEngine(db *models.EmbeddingDB) *Engine {
	return &Engine{
		db:     db,
		config: DefaultScoringConfig(),
	}
}
