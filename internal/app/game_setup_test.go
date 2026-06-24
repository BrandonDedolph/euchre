package app

import (
	"testing"

	"github.com/bran/euchre/internal/ai"
	"github.com/bran/euchre/internal/ai/rule_based"
)

// selectDifficulty drives the setup menu to the AI Difficulty item and selects
// it once, returning the resulting GameSetup state.
func cycleDifficulty(g *GameSetup) {
	g.menu.Selected = 4 // AI Difficulty item
	g.handleSelect()
}

func TestGameSetupDifficultyDefaultsToMedium(t *testing.T) {
	g := NewGameSetup()
	if g.difficulty != ai.DifficultyMedium {
		t.Fatalf("default difficulty = %v, want Medium", g.difficulty)
	}
	if got := g.menu.Items[4].Label; got != "AI Difficulty: Medium" {
		t.Fatalf("default label = %q, want %q", got, "AI Difficulty: Medium")
	}
}

func TestGameSetupDifficultyCycles(t *testing.T) {
	g := NewGameSetup()

	// Medium -> Hard -> Easy -> Medium
	wants := []struct {
		diff  ai.Difficulty
		label string
	}{
		{ai.DifficultyHard, "AI Difficulty: Hard"},
		{ai.DifficultyEasy, "AI Difficulty: Easy"},
		{ai.DifficultyMedium, "AI Difficulty: Medium"},
	}

	for i, w := range wants {
		cycleDifficulty(g)
		if g.difficulty != w.diff {
			t.Errorf("cycle %d: difficulty = %v, want %v", i, g.difficulty, w.diff)
		}
		if got := g.menu.Items[4].Label; got != w.label {
			t.Errorf("cycle %d: label = %q, want %q", i, got, w.label)
		}
	}
}

func TestGameSetupStartGameCarriesDifficulty(t *testing.T) {
	g := NewGameSetup()
	cycleDifficulty(g) // Medium -> Hard

	g.menu.Selected = 0 // Start Game
	_, cmd := g.handleSelect()
	if cmd == nil {
		t.Fatal("Start Game produced no command")
	}

	msg := cmd()
	navMsg, ok := msg.(NavigateMsg)
	if !ok {
		t.Fatalf("Start Game message type = %T, want NavigateMsg", msg)
	}
	settings, ok := navMsg.Data.(GameSettings)
	if !ok {
		t.Fatalf("nav data type = %T, want GameSettings", navMsg.Data)
	}
	if settings.Difficulty != ai.DifficultyHard {
		t.Fatalf("GameSettings.Difficulty = %v, want Hard", settings.Difficulty)
	}
}

func TestDifficultyReachesAIPlayers(t *testing.T) {
	gp := NewGamePlayWithSettings(GameSettings{Variant: "Standard", Difficulty: ai.DifficultyHard})
	if len(gp.aiPlayers) == 0 {
		t.Fatal("no AI players created")
	}
	checked := 0
	for i, p := range gp.aiPlayers {
		if p == nil {
			continue // the human's seat has no AI
		}
		got := p.(*rule_based.AI).Difficulty()
		if got != ai.DifficultyHard {
			t.Errorf("AI player %d difficulty = %v, want Hard", i, got)
		}
		checked++
	}
	if checked == 0 {
		t.Fatal("no non-nil AI players to verify")
	}
}
