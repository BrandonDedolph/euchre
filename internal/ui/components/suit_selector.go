package components

import (
	"github.com/bran/euchre/internal/engine"
	"github.com/charmbracelet/lipgloss"
)

// SuitSelector represents a visual suit selection UI
type SuitSelector struct {
	Selected    int  // 0=Spades, 1=Hearts, 2=Diamonds, 3=Clubs
	ExcludeSuit engine.Suit // Suit to exclude (turned card suit in round 2)
}

// NewSuitSelector creates a new suit selector
func NewSuitSelector(excludeSuit engine.Suit) *SuitSelector {
	s := &SuitSelector{
		Selected:    0,
		ExcludeSuit: excludeSuit,
	}
	// If starting suit is excluded, move to next
	if s.getSuitAt(0) == excludeSuit {
		s.MoveRight()
	}
	return s
}

// suits in order
var selectorSuits = []engine.Suit{
	engine.Spades,
	engine.Hearts,
	engine.Diamonds,
	engine.Clubs,
}

func (s *SuitSelector) getSuitAt(idx int) engine.Suit {
	if idx >= 0 && idx < len(selectorSuits) {
		return selectorSuits[idx]
	}
	return engine.NoSuit
}

// MoveLeft moves selection to the left
func (s *SuitSelector) MoveLeft() {
	for i := 0; i < len(selectorSuits); i++ {
		s.Selected--
		if s.Selected < 0 {
			s.Selected = len(selectorSuits) - 1
		}
		if s.getSuitAt(s.Selected) != s.ExcludeSuit {
			break
		}
	}
}

// MoveRight moves selection to the right
func (s *SuitSelector) MoveRight() {
	for i := 0; i < len(selectorSuits); i++ {
		s.Selected++
		if s.Selected >= len(selectorSuits) {
			s.Selected = 0
		}
		if s.getSuitAt(s.Selected) != s.ExcludeSuit {
			break
		}
	}
}

// SelectedSuit returns the currently selected suit
func (s *SuitSelector) SelectedSuit() engine.Suit {
	return s.getSuitAt(s.Selected)
}

// Render returns the visual representation of the suit selector
func (s *SuitSelector) Render() string {
	var parts []string

	for i, suit := range selectorSuits {
		if suit == s.ExcludeSuit {
			continue
		}

		symbol := suit.Symbol()
		name := suit.String()

		// Determine colors
		var fgColor lipgloss.Color
		if suit == engine.Hearts || suit == engine.Diamonds {
			fgColor = lipgloss.Color("#E74C3C") // Red
		} else {
			fgColor = lipgloss.Color("#FFF8E7") // Cream white for black suits
		}

		var style lipgloss.Style
		if i == s.Selected {
			// Selected style - highlighted
			style = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#FFFFFF")).
				Background(lipgloss.Color("#3498DB")).
				Bold(true).
				Padding(0, 1)
		} else {
			// Normal style
			style = lipgloss.NewStyle().
				Foreground(fgColor).
				Padding(0, 1)
		}

		parts = append(parts, style.Render(symbol+" "+name))
	}

	// Join with spacing
	return lipgloss.JoinHorizontal(lipgloss.Center, parts...)
}
