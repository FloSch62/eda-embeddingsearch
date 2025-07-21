package test

import (
	"math"
	"testing"

	"github.com/eda-labs/eda-embeddingsearch/internal/search"
)

func TestDotProduct(t *testing.T) {
	tests := []struct {
		name     string
		a        []float64
		b        []float64
		expected float64
	}{
		{
			name:     "normal vectors",
			a:        []float64{1, 2, 3},
			b:        []float64{4, 5, 6},
			expected: 32, // 1*4 + 2*5 + 3*6
		},
		{
			name:     "zero vector",
			a:        []float64{1, 2, 3},
			b:        []float64{0, 0, 0},
			expected: 0,
		},
		{
			name:     "different lengths",
			a:        []float64{1, 2},
			b:        []float64{3, 4, 5},
			expected: 0,
		},
		{
			name:     "empty vectors",
			a:        []float64{},
			b:        []float64{},
			expected: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := search.DotProduct(tt.a, tt.b)
			if result != tt.expected {
				t.Errorf("DotProduct(%v, %v) = %v, want %v", tt.a, tt.b, result, tt.expected)
			}
		})
	}
}

func TestMagnitude(t *testing.T) {
	tests := []struct {
		name     string
		v        []float64
		expected float64
	}{
		{
			name:     "normal vector",
			v:        []float64{3, 4},
			expected: 5, // sqrt(9 + 16)
		},
		{
			name:     "unit vector",
			v:        []float64{1, 0, 0},
			expected: 1,
		},
		{
			name:     "zero vector",
			v:        []float64{0, 0, 0},
			expected: 0,
		},
		{
			name:     "empty vector",
			v:        []float64{},
			expected: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := search.Magnitude(tt.v)
			if math.Abs(result-tt.expected) > 1e-10 {
				t.Errorf("Magnitude(%v) = %v, want %v", tt.v, result, tt.expected)
			}
		})
	}
}

func TestCosineSimilarity(t *testing.T) {
	tests := []struct {
		name     string
		a        []float64
		b        []float64
		expected float64
	}{
		{
			name:     "identical vectors",
			a:        []float64{1, 0, 0},
			b:        []float64{1, 0, 0},
			expected: 1,
		},
		{
			name:     "orthogonal vectors",
			a:        []float64{1, 0},
			b:        []float64{0, 1},
			expected: 0,
		},
		{
			name:     "opposite vectors",
			a:        []float64{1, 0},
			b:        []float64{-1, 0},
			expected: -1,
		},
		{
			name:     "different lengths",
			a:        []float64{1, 2},
			b:        []float64{3, 4, 5},
			expected: 0,
		},
		{
			name:     "zero vector",
			a:        []float64{1, 2, 3},
			b:        []float64{0, 0, 0},
			expected: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := search.CosineSimilarity(tt.a, tt.b)
			if math.Abs(result-tt.expected) > 1e-10 {
				t.Errorf("CosineSimilarity(%v, %v) = %v, want %v", tt.a, tt.b, result, tt.expected)
			}
		})
	}
}