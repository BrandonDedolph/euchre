package app

import (
	"strings"
	"testing"

	"github.com/bran/euchre/internal/engine"
)

func TestGradeCardMatchAndMiss(t *testing.T) {
	g := NewGamePlayWithSettings(GameSettings{Variant: "Standard", Tutorial: true})
	nine := engine.Card{Suit: engine.Hearts, Rank: engine.Nine}
	aceSpades := engine.Card{Suit: engine.Spades, Rank: engine.Ace}

	// Match → good grade.
	g.gradeCard("play", nine, nine)
	if !g.gradeGood || g.gradeMsg == "" {
		t.Fatalf("match: gradeGood=%v msg=%q", g.gradeGood, g.gradeMsg)
	}

	// Miss → not good, names the coach's card.
	g.gradeCard("play", nine, aceSpades)
	if g.gradeGood {
		t.Fatalf("miss should not be good: %q", g.gradeMsg)
	}
	if !strings.Contains(g.gradeMsg, aceSpades.String()) {
		t.Errorf("miss grade should name coach card %s: %q", aceSpades, g.gradeMsg)
	}
}

func TestGradeCardNoopWhenNotTutorial(t *testing.T) {
	g := NewGamePlay() // tutorial off
	c := engine.Card{Suit: engine.Hearts, Rank: engine.Nine}
	g.gradeCard("play", c, engine.Card{Suit: engine.Spades, Rank: engine.Ace})
	if g.gradeMsg != "" {
		t.Fatalf("non-tutorial should not grade: %q", g.gradeMsg)
	}
}
