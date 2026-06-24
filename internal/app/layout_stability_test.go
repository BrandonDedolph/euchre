package app

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

// pos is the row/column of a marker within a rendered view.
type pos struct{ row, col int }

// posOf returns the row and column of the first occurrence of marker, or
// {-1,-1} if it never appears. Column is measured in runes so that wide ANSI
// styling and multibyte glyphs don't skew it.
func posOf(view, marker string) pos {
	for i, line := range strings.Split(view, "\n") {
		if c := strings.Index(line, marker); c >= 0 {
			return pos{i, len([]rune(line[:c]))}
		}
	}
	return pos{-1, -1}
}

// rowOf returns just the row of the first line containing marker, or -1.
func rowOf(view, marker string) int { return posOf(view, marker).row }

// renderableGamePlay builds a GamePlay parked in a static, non-animating play
// state at a fixed terminal size so View() exercises the real layout path.
func renderableGamePlay(t *testing.T, tutorial bool, w, h int) *GamePlay {
	t.Helper()
	g := newGamePlay(rulesFromVariant(variantFromSettings(GameSettings{Variant: "Standard"})), tutorial)
	g.isShuffling = false
	g.isDealing = false
	g.width = w
	g.height = h
	g.updateTableView()
	return g
}

// assertAnchorsStable renders the view in each state and checks that the table
// and side panels occupy the exact same rows every time — i.e. nothing that is
// meant to be fixed in place jumps when variable content changes.
func assertAnchorsStable(t *testing.T, g *GamePlay, mutate map[string]func()) {
	t.Helper()
	anchors := []string{"Partner", "YOU", "OPP"}

	base := g.View()
	want := map[string]pos{}
	for _, a := range anchors {
		p := posOf(base, a)
		if p.row < 0 {
			t.Fatalf("anchor %q not found in baseline view", a)
		}
		want[a] = p
	}

	for name, fn := range mutate {
		fn()
		view := g.View()
		for _, a := range anchors {
			if got := posOf(view, a); got != want[a] {
				t.Errorf("state %q: anchor %q moved from %+v to %+v", name, a, want[a], got)
			}
		}
	}
}

func TestGamePlayLayoutStable(t *testing.T) {
	g := renderableGamePlay(t, false, 100, 40)

	assertAnchorsStable(t, g, map[string]func(){
		"recent ticker appears": func() {
			g.eventLog = []string{"You played 9♠", "West played K♠", "Partner played A♠"}
		},
		"long wrapping status message": func() {
			g.message = "West ordered up the jack of spades and is going alone, " +
				"so you will need to defend carefully against a strong lone hand here"
		},
		"short status message": func() {
			g.message = "Your turn"
		},
		"no ticker again": func() {
			g.eventLog = nil
		},
	})
}

func TestGamePlayTutorialCoachLayoutStable(t *testing.T) {
	g := renderableGamePlay(t, true, 100, 40)

	assertAnchorsStable(t, g, map[string]func(){
		"short coach tip": func() {
			g.message = "Pick a card"
			g.gradeMsg = "Good."
			g.gradeGood = true
		},
		"long coach tip": func() {
			g.gradeMsg = "Leading your off-ace here is risky because the opponents " +
				"are void in that suit and can trump it, costing you a likely trick."
		},
		"ticker plus coach": func() {
			g.eventLog = []string{"You played 9♠", "West trumped with J♣"}
		},
	})
}

// TestGamePlayCompactLayoutStable exercises the narrower (panel-less) layout
// path, where a single score bar replaces the side panels.
func TestGamePlayCompactLayoutStable(t *testing.T) {
	g := renderableGamePlay(t, false, 70, 40)

	assertAnchorsStable(t, g, map[string]func(){
		"ticker appears": func() {
			g.eventLog = []string{"You played 9♠", "West played K♠"}
		},
		"long status message": func() {
			g.message = "West ordered up and is going alone so defend carefully now"
		},
	})
}

// TestMainMenuLayoutStable verifies the menu does not shift as the selection
// moves across items whose descriptions wrap to a second line.
func TestMainMenuLayoutStable(t *testing.T) {
	m := NewMainMenu()
	m, _ = updateModel(m, tea.WindowSizeMsg{Width: 100, Height: 40})

	base := m.View()
	titleRow := rowOf(base, "Learn and play")
	if titleRow < 0 {
		t.Fatal("subtitle not found in baseline main menu view")
	}

	for i := 0; i < 5; i++ {
		m, _ = updateModel(m, tea.KeyMsg{Type: tea.KeyDown})
		if got := rowOf(m.View(), "Learn and play"); got != titleRow {
			t.Errorf("after %d down presses, subtitle moved from row %d to row %d", i+1, titleRow, got)
		}
	}
}

// updateModel is a small helper to drive a tea.Model and keep the concrete type.
func updateModel[T tea.Model](m T, msg tea.Msg) (T, tea.Cmd) {
	updated, cmd := m.Update(msg)
	return updated.(T), cmd
}
