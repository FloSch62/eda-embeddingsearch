// Package constants defines scoring weights and operational thresholds used
// throughout the search engine.
package constants

// Search and scoring constants
const (
	// Scoring thresholds
	BaseIndexMatchScore   = 10.0
	AllWordsMatchBonus    = 20.0
	SROSScoreThreshold    = 8.0
	DefaultScoreThreshold = 10.0
	MinScoreThreshold     = 5.0

	// Alarm scoring
	AlarmWordScore     = 10.0
	AlarmSeverityScore = 5.0

	// Search limits
	MaxSearchResults = 10
	MaxCandidates    = 20

	// EQL constants
	DefaultHighMemoryThreshold = 80
	MaxLimitValue              = 1000
	DefaultTopLimit            = 10
	RealTimeIntervalSeconds    = 1

	// Tokenizer constants
	MinTokenLength = 2

	// File permissions
	DirPermissions = 0o755
)
