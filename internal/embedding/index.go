package embedding

import (
	"github.com/eda-labs/eda-embeddingsearch/internal/search"
	"github.com/eda-labs/eda-embeddingsearch/pkg/models"
)

// BuildInvertedIndex creates an inverted index for fast word-based lookups
func BuildInvertedIndex(db *models.EmbeddingDB) {
	if db.InvertedIndex != nil && len(db.InvertedIndex) > 0 {
		// Already built
		return
	}

	db.InvertedIndex = make(map[string][]string)

	for key, entry := range db.Table {
		// Index key tokens
		keyTokens := search.Tokenize(key)
		for _, token := range keyTokens {
			db.InvertedIndex[token] = append(db.InvertedIndex[token], key)
		}

		// Index reference text tokens (limited to avoid memory bloat)
		refTokens := search.Tokenize(entry.ReferenceText)
		for i, token := range refTokens {
			if i > 50 { // Limit to first 50 tokens
				break
			}
			db.InvertedIndex[token] = append(db.InvertedIndex[token], key)
		}

		// Also index Text field for better matching
		textTokens := search.Tokenize(entry.Text)
		for i, token := range textTokens {
			if i > 30 { // Limit tokens from Text field
				break
			}
			db.InvertedIndex[token] = append(db.InvertedIndex[token], key)
		}
	}

	// Deduplicate entries
	for word, keys := range db.InvertedIndex {
		seen := make(map[string]bool)
		unique := make([]string, 0, len(keys))
		for _, key := range keys {
			if !seen[key] {
				seen[key] = true
				unique = append(unique, key)
			}
		}
		db.InvertedIndex[word] = unique
	}
}