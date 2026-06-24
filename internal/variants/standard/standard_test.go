package standard

import (
	"testing"

	"github.com/BrandonDedolph/euchre/internal/engine"
)

// TestScoreRound_DefendAloneEuchre verifies a defend-alone euchre awards 4
// points to the defenders, not the flat 2 for an ordinary euchre.
func TestScoreRound_DefendAloneEuchre(t *testing.T) {
	s := New()

	// Makers (team 0) were euchred while the defenders (team 1) defended alone.
	result := engine.RoundResult{
		Makers:           0,
		WasEuchred:       true,
		WasDefendedAlone: true,
		DefendPoints:     4,
	}
	got := s.ScoreRound(result)
	if got.Team1Delta != 4 {
		t.Errorf("defend-alone euchre should award defenders 4, got Team1Delta=%d", got.Team1Delta)
	}
	if got.Team0Delta != 0 {
		t.Errorf("euchred makers should score 0, got Team0Delta=%d", got.Team0Delta)
	}
}

// TestScoreRound_OrdinaryEuchre verifies a plain euchre awards the standard 2.
func TestScoreRound_OrdinaryEuchre(t *testing.T) {
	s := New()
	result := engine.RoundResult{
		Makers:       1,
		WasEuchred:   true,
		DefendPoints: 2,
	}
	got := s.ScoreRound(result)
	if got.Team0Delta != 2 {
		t.Errorf("ordinary euchre should award defenders 2, got Team0Delta=%d", got.Team0Delta)
	}
	if got.Team1Delta != 0 {
		t.Errorf("euchred makers should score 0, got Team1Delta=%d", got.Team1Delta)
	}
}

// TestScoreRound_MakersScore verifies the making branch is unaffected.
func TestScoreRound_MakersScore(t *testing.T) {
	s := New()
	result := engine.RoundResult{
		Makers:      0,
		MakerPoints: 2,
	}
	got := s.ScoreRound(result)
	if got.Team0Delta != 2 {
		t.Errorf("makers should score their MakerPoints, got Team0Delta=%d", got.Team0Delta)
	}
}
