package search

import (
	"encoding/json"
)

func parseEmbeddingInfo(text string) (description string, fields []string) {
	var embeddingInfo struct {
		Description string   `json:"Description"`
		Fields      []string `json:"Fields"`
	}

	if err := json.Unmarshal([]byte(text), &embeddingInfo); err == nil {
		return embeddingInfo.Description, embeddingInfo.Fields
	}

	return "", []string{}
}
