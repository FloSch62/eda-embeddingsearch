package search

import "math"

// DotProduct calculates the dot product of two vectors
func DotProduct(a, b []float64) float64 {
	if len(a) != len(b) {
		return 0
	}
	sum := 0.0
	for i := range a {
		sum += a[i] * b[i]
	}
	return sum
}

// Magnitude calculates the magnitude of a vector
func Magnitude(v []float64) float64 {
	sum := 0.0
	for _, val := range v {
		sum += val * val
	}
	return math.Sqrt(sum)
}

// CosineSimilarity calculates the cosine similarity between two vectors
func CosineSimilarity(a, b []float64) float64 {
	if len(a) != len(b) || len(a) == 0 {
		return 0
	}

	dot := DotProduct(a, b)
	magA := Magnitude(a)
	magB := Magnitude(b)

	if magA == 0 || magB == 0 {
		return 0
	}

	return dot / (magA * magB)
}

// CreateQueryEmbedding creates a simple embedding from query words using TF-IDF-like approach
func CreateQueryEmbedding(query string, vocabSize int) []float64 {
	// Create a random but deterministic embedding based on query
	// This is a simplified approach - in production you'd use a proper embedding model
	embedding := make([]float64, vocabSize)
	words := ExpandSynonyms(Tokenize(query))

	// Use a deterministic random approach based on word content
	for _, word := range words {
		// Create a seed from the word
		seed := int64(0)
		for _, ch := range word {
			seed = seed*31 + int64(ch)
		}

		// Generate pseudo-random values
		for j := 0; j < vocabSize; j++ {
			// Simple linear congruential generator
			seed = (seed*1103515245 + 12345) & 0x7fffffff
			value := float64(seed) / float64(0x7fffffff)

			// Create sparse embedding with some randomness
			if value < 0.1 { // 10% chance of non-zero
				embedding[j] += (value - 0.05) * 2.0 / float64(len(words))
			}
		}
	}

	// Add some Gaussian-like smoothing
	smoothed := make([]float64, vocabSize)
	for i := range embedding {
		sum := embedding[i] * 0.5
		if i > 0 {
			sum += embedding[i-1] * 0.25
		}
		if i < vocabSize-1 {
			sum += embedding[i+1] * 0.25
		}
		smoothed[i] = sum
	}

	// Normalize to unit vector
	mag := Magnitude(smoothed)
	if mag > 0 {
		for i := range smoothed {
			smoothed[i] /= mag
		}
	}

	return smoothed
}