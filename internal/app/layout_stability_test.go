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

// fullLayoutWidth is wide enough to clear fullLayoutMinWidth (~89) so the full
// HUD (YOU card flanking the left, OPP card flanking the right) renders.
const fullLayoutWidth = 100

func TestGamePlayLayoutStable(t *testing.T) {
	g := renderableGamePlay(t, false, fullLayoutWidth, 40)

	// "YOU"/"OPP" are the scoreboard cards flanking the table; "Partner" anchors
	// the table itself. All must stay pinned across every per-seat action and
	// message-length change.
	anchors := []string{"Partner", "YOU", "OPP"}

	setActions := func(a [4]string) func() {
		return func() {
			g.playerAction = a
			g.updateTableView()
		}
	}

	assertAnchorsStable(t, g, anchors, map[string]func(){
		"no actions":    setActions([4]string{}),
		"one action":    setActions([4]string{"", "passes", "", ""}),
		"all seats act": setActions([4]string{"orders up", "passes", "passes", "calls ♠"}),
		"long action labels": setActions([4]string{
			"orders up, alone!",
			"this is an extremely long action label that should be truncated",
			"defends alone!",
			"calls ♠, alone!",
		}),
		"long wrapping status message": func() {
			g.message = "West ordered up the jack of spades and is going alone, " +
				"so you will need to defend carefully against a strong lone hand here"
		},
		"short status message": func() {
			g.message = "Your turn"
		},
		"actions cleared again": setActions([4]string{}),
	})
}

// TestGamePlaySeatActionsStable verifies the reserved per-seat action line keeps
// the table anchors at constant (row,col) and the total view height constant as
// action labels appear, change length, and clear — i.e. the line is always
// reserved and never shifts the layout. It also asserts an action string
// actually appears next to its seat in the rendered view.
func TestGamePlaySeatActionsStable(t *testing.T) {
	g := renderableGamePlay(t, false, fullLayoutWidth, 40)

	setActions := func(a [4]string) func() {
		return func() {
			g.playerAction = a
			g.updateTableView()
		}
	}

	assertAnchorsStable(t, g, []string{"Partner", "YOU", "OPP"}, map[string]func(){
		"west passes":          setActions([4]string{"", "passes", "", ""}),
		"very long west label": setActions([4]string{"", strings.Repeat("verylong ", 8), "", ""}),
		"empty":                setActions([4]string{}),
	})

	// The action label should actually render near its seat. "West (0)" is the
	// West seat label; "passes" must appear on the line directly below it.
	g.playerAction = [4]string{"", "passes", "", ""}
	g.updateTableView()
	view := g.View()
	westRow := rowOf(view, "West")
	if westRow < 0 {
		t.Fatal("West seat label not found")
	}
	passesRow := rowOf(view, "passes")
	if passesRow < 0 {
		t.Fatal("expected 'passes' action label to appear in the view")
	}
	if passesRow != westRow+1 {
		t.Errorf("expected 'passes' directly under West (row %d), got row %d", westRow+1, passesRow)
	}
}

func TestGamePlayTutorialCoachLayoutStable(t *testing.T) {
	g := renderableGamePlay(t, true, fullLayoutWidth, 40)

	assertAnchorsStable(t, g, []string{"Partner", "YOU", "OPP"}, map[string]func(){
		"short coach tip": func() {
			g.message = "Pick a card"
			g.gradeMsg = "Good."
			g.gradeGood = true
		},
		"long coach tip": func() {
			g.gradeMsg = "Leading your off-ace here is risky because the opponents " +
				"are void in that suit and can trump it, costing you a likely trick."
		},
		"seat actions plus coach": func() {
			g.playerAction = [4]string{"orders up", "passes", "", "calls ♥"}
			g.updateTableView()
		},
	})
}

// TestGamePlayCompactLayoutStable exercises the narrower (panel-less) layout
// path, where a single score bar replaces the side panels.
func TestGamePlayCompactLayoutStable(t *testing.T) {
	g := renderableGamePlay(t, false, 70, 40)

	assertAnchorsStable(t, g, []string{"Partner", "YOU", "OPP"}, map[string]func(){
		"seat actions appear": func() {
			g.playerAction = [4]string{"", "passes", "orders up", ""}
			g.updateTableView()
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
	anchors := []string{"Partner", "YOU", "OPP"}

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
