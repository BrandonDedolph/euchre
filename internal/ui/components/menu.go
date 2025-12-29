package components

import (
	"strings"

	"github.com/bran/euchre/internal/ui/theme"
)

// MenuItem represents a menu option
type MenuItem struct {
	Label       string
	Description string
	Disabled    bool
	Action      func()
}

// Menu represents a selectable menu
type Menu struct {
	Title    string
	Items    []MenuItem
	Selected int
}

// NewMenu creates a new menu
func NewMenu(title string, items []MenuItem) *Menu {
	return &Menu{
		Title:    title,
		Items:    items,
		Selected: 0,
	}
}

// MoveUp moves selection up
func (m *Menu) MoveUp() {
	if m.Selected > 0 {
		m.Selected--
		// Skip disabled items
		for m.Selected > 0 && m.Items[m.Selected].Disabled {
			m.Selected--
		}
	}
}

// MoveDown moves selection down
func (m *Menu) MoveDown() {
	if m.Selected < len(m.Items)-1 {
		m.Selected++
		// Skip disabled items
		for m.Selected < len(m.Items)-1 && m.Items[m.Selected].Disabled {
			m.Selected++
		}
	}
}

// CurrentItem returns the currently selected item
func (m *Menu) CurrentItem() *MenuItem {
	if m.Selected >= 0 && m.Selected < len(m.Items) {
		return &m.Items[m.Selected]
	}
	return nil
}

// Render returns the visual representation of the menu
func (m *Menu) Render() string {
	var sb strings.Builder

	if m.Title != "" {
		sb.WriteString(theme.Current.Title.Render(m.Title))
		sb.WriteString("\n\n")
	}

	// Fixed description area at top
	if m.Selected >= 0 && m.Selected < len(m.Items) {
		desc := m.Items[m.Selected].Description
		if desc != "" {
			sb.WriteString(theme.Current.Subtitle.Render(desc))
			sb.WriteString("\n\n")
		}
	}

	for i, item := range m.Items {
		var line string
		if i == m.Selected {
			line = theme.Current.MenuItemSelected.Render("> " + item.Label)
		} else if item.Disabled {
			line = theme.Current.MenuItemDisabled.Render("  " + item.Label)
		} else {
			line = theme.Current.MenuItem.Render("  " + item.Label)
		}

		sb.WriteString(line)
		sb.WriteString("\n")
	}

	return sb.String()
}
