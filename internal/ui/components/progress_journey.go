package components

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// JourneyProgress renders a visual progress indicator for the learning journey
type JourneyProgress struct {
	TotalSteps  int
	CurrentStep int  // 0-indexed
	ShowNumbers bool // Whether to show step numbers below dots
}

// NewJourneyProgress creates a new journey progress component
func NewJourneyProgress(total, current int) *JourneyProgress {
	return &JourneyProgress{
		TotalSteps:  total,
		CurrentStep: current,
		ShowNumbers: true,
	}
}

// Render returns the progress visualization
func (jp *JourneyProgress) Render() string {
	if jp.TotalSteps == 0 {
		return ""
	}

	completedStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#27AE60")) // Green

	currentStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#3498DB")). // Blue
		Bold(true)

	upcomingStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#95A5A6")) // Gray

	lineStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#7F8C8D")) // Darker gray

	// Build the dots line
	var dotsLine strings.Builder
	var numbersLine strings.Builder

	for i := 0; i < jp.TotalSteps; i++ {
		var dot string
		var numStyle lipgloss.Style

		if i < jp.CurrentStep {
			// Completed
			dot = completedStyle.Render("●")
			numStyle = completedStyle
		} else if i == jp.CurrentStep {
			// Current
			dot = currentStyle.Render("●")
			numStyle = currentStyle
		} else {
			// Upcoming
			dot = upcomingStyle.Render("○")
			numStyle = upcomingStyle
		}

		dotsLine.WriteString(dot)

		if jp.ShowNumbers {
			numbersLine.WriteString(numStyle.Render(fmt.Sprintf("%d", i+1)))
		}

		// Add connector line between dots (except after last)
		if i < jp.TotalSteps-1 {
			dotsLine.WriteString(lineStyle.Render("━━━"))
			if jp.ShowNumbers {
				numbersLine.WriteString("   ") // Spacing to align with connector
			}
		}
	}

	result := dotsLine.String()
	if jp.ShowNumbers {
		result += "\n" + numbersLine.String()
	}

	return result
}

// RenderCompact returns a compact progress visualization without numbers
func (jp *JourneyProgress) RenderCompact() string {
	jp.ShowNumbers = false
	return jp.Render()
}

// RenderWithLabel returns the progress with a label showing current position
func (jp *JourneyProgress) RenderWithLabel(lessonTitle string) string {
	labelStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#3498DB"))

	progress := jp.Render()
	label := labelStyle.Render(fmt.Sprintf("Lesson %d of %d: %s", jp.CurrentStep+1, jp.TotalSteps, lessonTitle))

	return progress + "\n\n" + label
}
