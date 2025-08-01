// Package search defines configuration parameters for scoring search results
// across different matching dimensions.
package search

// ScoringConfig contains all the scoring weights and penalties
type ScoringConfig struct {
	// Keyword matching scores
	LastSegmentMatch      float64
	KeywordMatchInterface float64
	KeywordMatchStats     float64
	KeywordMatchState     float64
	KeywordMatchDefault   float64
	TextMatch             float64

	// Description matching
	DescriptionWordMatch  float64
	DescriptionListMatch  float64
	DescriptionAllMatch   float64
	DescriptionShowMatch  float64
	DescriptionGetMatch   float64
	DescriptionMultiMatch float64

	// Interface scoring
	InterfaceEndMatch        float64
	InterfaceStatsMatch      float64
	InterfacePluralMatch     float64
	InterfaceSecurityPenalty float64
	InterfaceProtocolPenalty float64

	// BGP scoring
	BGPNeighborMatch             float64
	BGPSessionStateBonus         float64
	BGPNonNeighborPenalty        float64
	BGPGeneralMatch              float64
	BGPMaintenancePenalty        float64
	BGPMaintenanceSessionPenalty float64

	// Path depth scoring
	PathDepthBonus2        float64
	PathDepthBonus3        float64
	PathDepthBonus4        float64
	PathDepthPenaltyFactor float64

	// Segment matching
	SegmentExactMatch float64
	SegmentNearMatch  float64
	SegmentFarMatch   float64

	// Other matches
	SubinterfaceExactMatch   float64
	SubinterfacePartialMatch float64
	ExactTableMatch          float64
	BigramMatch              float64
	FieldExtractScore        float64
	SequenceMatch            float64
	SequencePartialMatch     float64

	// Context bonuses
	ShowStateBonus     float64
	AllWordsMatchBonus float64

	// Penalties
	ProtocolPenalty    float64
	MaintenancePenalty float64

	// Special query scoring
	ErrorFieldBonus     float64
	BandwidthFieldBonus float64
}

// DefaultScoringConfig returns the default scoring configuration
func DefaultScoringConfig() *ScoringConfig {
	return &ScoringConfig{
		// Keyword matching scores
		LastSegmentMatch:      10,
		KeywordMatchInterface: 8,
		KeywordMatchStats:     6,
		KeywordMatchState:     4,
		KeywordMatchDefault:   3,
		TextMatch:             1,

		// Description matching
		DescriptionWordMatch:  3,
		DescriptionListMatch:  5,
		DescriptionAllMatch:   3,
		DescriptionShowMatch:  2,
		DescriptionGetMatch:   2,
		DescriptionMultiMatch: 5,

		// Interface scoring
		InterfaceEndMatch:        20,
		InterfaceStatsMatch:      15,
		InterfacePluralMatch:     10,
		InterfaceSecurityPenalty: -20,
		InterfaceProtocolPenalty: -15,

		// BGP scoring
		BGPNeighborMatch:             15,
		BGPSessionStateBonus:         20,
		BGPNonNeighborPenalty:        -15,
		BGPGeneralMatch:              10,
		BGPMaintenancePenalty:        -10,
		BGPMaintenanceSessionPenalty: -25,

		// Path depth scoring
		PathDepthBonus2:        20,
		PathDepthBonus3:        10,
		PathDepthBonus4:        5,
		PathDepthPenaltyFactor: 2,

		// Segment matching
		SegmentExactMatch: 10,
		SegmentNearMatch:  6,
		SegmentFarMatch:   2,

		// Other matches
		SubinterfaceExactMatch:   10,
		SubinterfacePartialMatch: 2,
		ExactTableMatch:          6,
		BigramMatch:              2,
		FieldExtractScore:        1.5,
		SequenceMatch:            8,
		SequencePartialMatch:     4,

		// Context bonuses
		ShowStateBonus:     5,
		AllWordsMatchBonus: 3,

		// Penalties
		ProtocolPenalty:    -10,
		MaintenancePenalty: -8,

		// Special query scoring
		ErrorFieldBonus:     10,
		BandwidthFieldBonus: 10,
	}
}
