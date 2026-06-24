package app

import (
	"regexp"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
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
func assertAnchorsStable(t *testing.T, g *GamePlay, anchors []string, mutate map[string]func()) {
	t.Helper()

	base := g.View()
	baseHeight := lipgloss.Height(base)
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
		if got := lipgloss.Height(view); got != baseHeight {
			t.Errorf("state %q: total view height changed from %d to %d", name, baseHeight, got)
		}
	}
}

// fullLayoutWidth is wide enough to clear fullLayoutMinWidth so the full HUD
// (Recent box on the left, stacked YOU/OPP cards on the right) renders. The
// columns are now wider than before, so the old 100-col width would fall back to
// the compact layout — these tests use a width that matches the real terminal.
const fullLayoutWidth = 110

func TestGamePlayLayoutStable(t *testing.T) {
	g := renderableGamePlay(t, false, fullLayoutWidth, 40)

	// "Recent" is the left-column log box title; "YOU"/"OPP" are now the stacked
	// scoreboard cards on the RIGHT. All must stay pinned across every event-count
	// and message-length change.
	anchors := []string{"Partner", "YOU", "OPP", "Recent"}

	assertAnchorsStable(t, g, anchors, map[string]func(){
		"empty log": func() {
			g.eventLog = nil
		},
		"one event": func() {
			g.eventLog = []string{"You played 9♠"}
		},
		"two events": func() {
			g.eventLog = []string{"You played 9♠", "West played K♠"}
		},
		"overflowing log": func() {
			g.eventLog = []string{
				"e1", "e2", "e3", "e4", "e5", "e6",
				"e7", "e8", "e9", "e10", "e11", "e12", "e13", "e14",
			}
		},
		"one very long event": func() {
			g.eventLog = []string{
				"West ordered up the right bower and led it into your partner's void",
			}
		},
		"wide glyph events": func() {
			g.eventLog = []string{"You played 9♠", "West trumped ♥♣♦", "Partner led A♠"}
		},
		"long wrapping status message": func() {
			g.message = "West ordered up the jack of spades and is going alone, " +
				"so you will need to defend carefully against a strong lone hand here"
		},
		"short status message": func() {
			g.message = "Your turn"
		},
		"no log again": func() {
			g.eventLog = nil
		},
	})
}

// TestGamePlayLogBulletGrouping verifies each Recent entry is delimited with a
// leading bullet on its first line (so multi-line entries are unambiguous), and
// that the box stays a constant height as entries are added — the whole point of
// the delimiter is grouping without disturbing layout stability.
func TestGamePlayLogBulletGrouping(t *testing.T) {
	g := renderableGamePlay(t, false, fullLayoutWidth, 40)

	g.eventLog = []string{"You played 9♠"}
	one := g.View()
	if !strings.Contains(one, "• You played 9♠") {
		t.Errorf("expected a bullet-prefixed entry in the Recent box, view was:\n%s", one)
	}
	oneHeight := lipgloss.Height(one)

	// A multi-line entry must still start with a single bullet and keep the box
	// (and thus the whole view) the same height.
	g.eventLog = []string{
		"You played 9♠",
		"West ordered up the right bower and led it into your partner's void",
		"Partner led A♠",
	}
	many := g.View()
	bullets := strings.Count(many, "• ")
	if bullets < 3 {
		t.Errorf("expected at least 3 bullet markers (one per entry), got %d", bullets)
	}
	if got := lipgloss.Height(many); got != oneHeight {
		t.Errorf("view height changed with more/multi-line entries: one=%d many=%d", oneHeight, got)
	}
}

// TestGamePlayLogBoxConstantHeight verifies the recent-log box keeps the whole
// view the same height whether the log is empty or full, so it never shoves the
// rest of the layout vertically.
func TestGamePlayLogBoxConstantHeight(t *testing.T) {
	g := renderableGamePlay(t, false, fullLayoutWidth, 40)

	g.eventLog = nil
	empty := lipgloss.Height(g.View())

	g.eventLog = []string{
		"e1", "e2", "e3", "e4", "e5", "e6", "e7", "e8", "e9", "e10", "e11", "e12",
	}
	full := lipgloss.Height(g.View())

	if empty != full {
		t.Errorf("view height changed with log content: empty=%d full=%d", empty, full)
	}
}

func TestGamePlayTutorialCoachLayoutStable(t *testing.T) {
	g := renderableGamePlay(t, true, fullLayoutWidth, 40)

	assertAnchorsStable(t, g, []string{"Partner", "YOU", "OPP", "Recent"}, map[string]func(){
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

	assertAnchorsStable(t, g, []string{"Partner", "YOU", "OPP"}, map[string]func(){
		"ticker appears": func() {
			g.eventLog = []string{"You played 9♠", "West played K♠"}
		},
		"long status message": func() {
			g.message = "West ordered up and is going alone so defend carefully now"
		},
	})
}

// TestGamePlayStatusCentered verifies the status line is centered within the
// fixed-width slot (not flush-left), and that anchors stay pinned and the view
// height stays constant as the message length changes.
func TestGamePlayStatusCentered(t *testing.T) {
	g := renderableGamePlay(t, false, fullLayoutWidth, 40)
	anchors := []string{"Partner", "YOU", "OPP", "Recent"}

	g.message = "Zalpha turn now"
	base := g.View()
	wantAnchors := map[string]pos{}
	for _, a := range anchors {
		wantAnchors[a] = posOf(base, a)
	}

	// The status row should be centered: its visible text has roughly equal
	// padding on both sides (and is clearly not flush-left at the margin).
	ansi := regexp.MustCompile("\x1b\\[[0-9;]*m")
	statusLine := ""
	for _, line := range strings.Split(base, "\n") {
		if strings.Contains(ansi.ReplaceAllString(line, ""), "Zalpha") {
			statusLine = ansi.ReplaceAllString(line, "")
			break
		}
	}
	if statusLine == "" {
		t.Fatal("status marker not found in baseline view")
	}
	// Drop the outer screen border (leading "│ ... │") before measuring padding.
	// A centered short message has large, near-equal padding on both sides; a
	// flush-left one would have only a couple of columns on the left. (A few
	// columns of asymmetry is expected from integer-rounding at the slot and
	// block centering layers.)
	trimmed := strings.Trim(statusLine, "│")
	left := len(trimmed) - len(strings.TrimLeft(trimmed, " "))
	right := len(trimmed) - len(strings.TrimRight(trimmed, " "))
	if left < 20 {
		t.Errorf("status appears flush-left (left pad %d); expected centered", left)
	}
	if diff := left - right; diff < -5 || diff > 5 {
		t.Errorf("status not centered: left pad %d vs right pad %d", left, right)
	}

	// Anchors and height stay stable across message length changes.
	baseHeight := lipgloss.Height(base)
	for name, msg := range map[string]string{
		"short": "Zalpha now",
		"long":  "Zalpha " + strings.Repeat("very long combined message ", 6),
	} {
		g.message = msg
		view := g.View()
		for _, a := range anchors {
			if got := posOf(view, a); got != wantAnchors[a] {
				t.Errorf("case %q: anchor %q moved from %+v to %+v", name, a, wantAnchors[a], got)
			}
		}
		if got := lipgloss.Height(view); got != baseHeight {
			t.Errorf("case %q: view height changed from %d to %d", name, baseHeight, got)
		}
	}
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
