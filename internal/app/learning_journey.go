package app

import (
	"regexp"
	"strings"

	"github.com/bran/euchre/internal/engine"
	"github.com/bran/euchre/internal/tutorial"
	_ "github.com/bran/euchre/internal/tutorial/content" // Register lessons
	"github.com/bran/euchre/internal/ui/components"
	"github.com/bran/euchre/internal/ui/theme"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// JourneyPhase represents the current phase of the learning journey
type JourneyPhase int

const (
	PhaseWelcome JourneyPhase = iota
	PhaseLessonContent
	PhaseCompletion
)

// LearningJourney is the guided learning experience screen
type LearningJourney struct {
	phase          JourneyPhase
	allLessons     []*tutorial.Lesson
	currentLesson  int
	currentSection int
	scroll         int
	width          int
	height         int

	// Visual section support
	visualView *components.LessonVisualView
	animCtrl   *components.AnimationController
}

// NewLearningJourney creates a new learning journey screen
func NewLearningJourney() *LearningJourney {
	return &LearningJourney{
		phase:          PhaseWelcome,
		allLessons:     tutorial.AllInOrder(),
		currentLesson:  0,
		currentSection: 0,
	}
}

// Init implements tea.Model
func (lj *LearningJourney) Init() tea.Cmd {
	return nil
}

// Update implements tea.Model
func (lj *LearningJourney) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		lj.width = msg.Width
		lj.height = msg.Height

	case components.AnimTickMsg:
		// Handle animation tick
		if lj.animCtrl != nil {
			return lj, lj.animCtrl.Tick()
		}

	case tea.KeyMsg:
		switch lj.phase {
		case PhaseWelcome:
			return lj.updateWelcome(msg)
		case PhaseLessonContent:
			return lj.updateLessonContent(msg)
		case PhaseCompletion:
			return lj.updateCompletion(msg)
		}
	}

	return lj, nil
}

func (lj *LearningJourney) updateWelcome(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "enter", " ":
		lj.phase = PhaseLessonContent
		lj.setupVisualSection()
	case "q", "esc":
		return lj, Navigate(ScreenMainMenu)
	}
	return lj, nil
}

func (lj *LearningJourney) updateLessonContent(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "enter", " ", "right", "l", "n":
		// If we have an animation that needs user input, advance it
		if lj.animCtrl != nil && lj.animCtrl.NeedsUserInput() {
			return lj, lj.animCtrl.Advance()
		}
		// If we have a visual sequence that's not complete, advance it
		if lj.visualView != nil && !lj.visualView.IsSequenceComplete() {
			lj.visualView.AdvanceSequence()
			return lj, nil
		}
		return lj.advanceContent()
	case "left", "h", "b":
		return lj.goBack()
	case "up", "k":
		if lj.scroll > 0 {
			lj.scroll--
		}
	case "down", "j":
		lj.scroll++
	case "q", "esc":
		return lj, Navigate(ScreenMainMenu)
	}
	return lj, nil
}

func (lj *LearningJourney) updateCompletion(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "enter", " ":
		return lj, Navigate(ScreenMainMenu)
	case "r":
		// Restart journey
		lj.phase = PhaseWelcome
		lj.currentLesson = 0
		lj.currentSection = 0
		lj.scroll = 0
	case "p":
		// Start playing
		return lj, Navigate(ScreenGameSetup)
	case "q", "esc":
		return lj, Navigate(ScreenMainMenu)
	}
	return lj, nil
}

func (lj *LearningJourney) advanceContent() (tea.Model, tea.Cmd) {
	if len(lj.allLessons) == 0 {
		return lj, nil
	}

	lesson := lj.allLessons[lj.currentLesson]
	sectionCount := lesson.SectionCount()

	if lj.currentSection < sectionCount-1 {
		// More sections in this lesson
		lj.currentSection++
		lj.scroll = 0
		lj.setupVisualSection()
		return lj, nil
	}

	// Finished this lesson
	if lj.currentLesson < len(lj.allLessons)-1 {
		// Move to next lesson
		lj.currentLesson++
		lj.currentSection = 0
		lj.scroll = 0
		lj.setupVisualSection()
		return lj, nil
	}

	// Journey complete!
	lj.phase = PhaseCompletion
	lj.visualView = nil
	lj.animCtrl = nil
	return lj, nil
}

func (lj *LearningJourney) goBack() (tea.Model, tea.Cmd) {
	if lj.currentSection > 0 {
		// Previous section in this lesson
		lj.currentSection--
		lj.scroll = 0
		lj.setupVisualSection()
		return lj, nil
	}

	if lj.currentLesson > 0 {
		// Go to last section of previous lesson
		lj.currentLesson--
		prevLesson := lj.allLessons[lj.currentLesson]
		lj.currentSection = prevLesson.SectionCount() - 1
		lj.scroll = 0
		lj.setupVisualSection()
		return lj, nil
	}

	// At the very beginning - go back to welcome
	lj.phase = PhaseWelcome
	lj.visualView = nil
	lj.animCtrl = nil
	return lj, nil
}

// setupVisualSection initializes the visual view for the current section
func (lj *LearningJourney) setupVisualSection() {
	lj.visualView = nil
	lj.animCtrl = nil

	if len(lj.allLessons) == 0 {
		return
	}

	lesson := lj.allLessons[lj.currentLesson]
	if !lesson.HasVisuals() {
		return
	}

	visualSection := lesson.GetVisualSection(lj.currentSection)
	if visualSection == nil || visualSection.Visual == nil {
		return
	}

	// Create visual view - use larger size for trump hierarchy
	boxWidth, boxHeight := 50, 10
	if visualSection.Visual.Type == tutorial.VisualTrumpHierarchy {
		boxWidth, boxHeight = 60, 16
	}
	lj.visualView = components.NewLessonVisualView(visualSection.Visual, boxWidth, boxHeight)

	// If the visual has a sequence, set up the animation controller
	if len(visualSection.Visual.Sequence) > 0 {
		lj.animCtrl = components.NewAnimationController(visualSection.Visual.Sequence)
	}
}

// View implements tea.Model
func (lj *LearningJourney) View() string {
	width := lj.width
	height := lj.height
	if width == 0 {
		width = 80
	}
	if height == 0 {
		height = 24
	}

	var header, content, footer string

	switch lj.phase {
	case PhaseWelcome:
		header, content, footer = lj.renderWelcomeParts(width - 4)
	case PhaseLessonContent:
		header, content, footer = lj.renderLessonContentParts(width - 4)
	case PhaseCompletion:
		header, content, footer = lj.renderCompletionParts(width - 4)
	}

	// Calculate content area height
	headerHeight := lipgloss.Height(header)
	footerHeight := lipgloss.Height(footer)
	contentHeight := height - headerHeight - footerHeight - 2 // spacing

	// Center content in the middle area
	centeredContent := lipgloss.Place(width, contentHeight, lipgloss.Center, lipgloss.Center, content)

	// Assemble: header at top, content in middle, footer at bottom
	return header + "\n" + centeredContent + "\n" + footer
}

func (lj *LearningJourney) renderWelcome(width int) string {
	// Decorative card display - show the four Jacks (the most important cards in Euchre)
	decorativeCards := []engine.Card{
		{Suit: engine.Spades, Rank: engine.Jack},
		{Suit: engine.Hearts, Rank: engine.Jack},
		{Suit: engine.Diamonds, Rank: engine.Jack},
		{Suit: engine.Clubs, Rank: engine.Jack},
	}
	cardViews := make([]string, len(decorativeCards))
	for i, card := range decorativeCards {
		cv := components.NewCardView(card)
		cardViews[i] = cv.Render()
	}
	cardRow := lipgloss.JoinHorizontal(lipgloss.Top, cardViews...)

	// Title with accent
	titleStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#3498DB")).
		Bold(true)

	title := titleStyle.Render("Learn to Play Euchre")

	// Subtitle/tagline
	tagline := theme.Current.LessonText.Render("Master America's favorite trick-taking card game")

	// Lesson list with numbers
	var lessonList strings.Builder
	numberStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#3498DB")).
		Bold(true)
	lessonTitleStyle := theme.Current.LessonText

	for i, lesson := range lj.allLessons {
		num := numberStyle.Render(string(rune('1'+i)) + ".")
		lessonList.WriteString("   " + num + " " + lessonTitleStyle.Render(lesson.Title) + "\n")
	}

	// Content box for lesson list
	contentBox := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#3498DB")).
		Padding(1, 2).
		Render(lessonList.String())

	// Time estimate with icon
	timeEstimate := theme.Current.Muted.Render("◷ About 10 minutes")

	// Button
	buttonStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#FFFFFF")).
		Background(lipgloss.Color("#27AE60")).
		Padding(0, 4).
		Bold(true)

	button := buttonStyle.Render("▶ Start Learning")

	// Help text
	help := theme.Current.Help.Render("Enter: Start • Esc: Back to menu")

	// Assemble layout
	content := lipgloss.PlaceHorizontal(width, lipgloss.Center, cardRow) + "\n\n" +
		lipgloss.PlaceHorizontal(width, lipgloss.Center, title) + "\n" +
		lipgloss.PlaceHorizontal(width, lipgloss.Center, tagline) + "\n\n" +
		lipgloss.PlaceHorizontal(width, lipgloss.Center, contentBox) + "\n\n" +
		lipgloss.PlaceHorizontal(width, lipgloss.Center, timeEstimate) + "\n\n" +
		lipgloss.PlaceHorizontal(width, lipgloss.Center, button) + "\n\n" +
		lipgloss.PlaceHorizontal(width, lipgloss.Center, help)

	return content
}

func (lj *LearningJourney) renderLessonContent(width int) string {
	if len(lj.allLessons) == 0 {
		return "No lessons available"
	}

	lesson := lj.allLessons[lj.currentLesson]
	sectionCount := lesson.SectionCount()

	// Progress indicator
	progress := components.NewJourneyProgress(len(lj.allLessons), lj.currentLesson)
	progressStr := progress.Render()

	// Lesson indicator
	lessonLabel := theme.Current.Primary.Render(
		strings.Repeat(" ", 4) + "Lesson " + string(rune('0'+lj.currentLesson+1)) + " of " + string(rune('0'+len(lj.allLessons))))

	// Check if this lesson has visual sections
	if lesson.HasVisuals() {
		return lj.renderVisualLessonContent(width, lesson, progressStr, lessonLabel, sectionCount)
	}

	// Fall back to text-based rendering
	section := lesson.Sections[lj.currentSection]

	// Section title
	sectionTitleStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#3498DB")).
		Bold(true)

	sectionTitle := sectionTitleStyle.Render(section.Title)

	// Content box with fixed size
	boxWidth := 50
	boxHeight := 10
	contentStyle := lipgloss.NewStyle().
		Width(boxWidth).
		Height(boxHeight).
		Padding(1, 2).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#3498DB"))

	// Apply scroll to content
	lines := strings.Split(section.Content, "\n")
	if lj.scroll > 0 && lj.scroll < len(lines) {
		lines = lines[lj.scroll:]
	}
	contentText := strings.Join(lines, "\n")
	contentText = colorizeCards(contentText)
	contentBox := contentStyle.Render(contentText)

	// Section progress within lesson
	sectionProgress := lj.renderSectionProgress(len(lesson.Sections), lj.currentSection)

	// Navigation hints
	var navHints string
	if lj.currentLesson == 0 && lj.currentSection == 0 {
		navHints = "                              Continue →"
	} else if lj.currentLesson == len(lj.allLessons)-1 && lj.currentSection == len(lesson.Sections)-1 {
		navHints = "← Back                        Finish"
	} else {
		navHints = "← Back                        Continue →"
	}
	navStyle := theme.Current.Muted
	navHintsRendered := navStyle.Render(navHints)

	help := theme.Current.Help.Render("←/→: Navigate • ↑/↓: Scroll • Esc: Exit")

	content := lipgloss.PlaceHorizontal(width, lipgloss.Center, progressStr) + "\n" +
		lipgloss.PlaceHorizontal(width, lipgloss.Center, lessonLabel) + "\n\n" +
		lipgloss.PlaceHorizontal(width, lipgloss.Center, sectionTitle) + "\n" +
		lipgloss.PlaceHorizontal(width, lipgloss.Center, contentBox) + "\n" +
		lipgloss.PlaceHorizontal(width, lipgloss.Center, sectionProgress) + "\n" +
		lipgloss.PlaceHorizontal(width, lipgloss.Center, navHintsRendered) + "\n\n" +
		lipgloss.PlaceHorizontal(width, lipgloss.Center, help)

	return content
}

func (lj *LearningJourney) renderVisualLessonContent(width int, lesson *tutorial.Lesson, progressStr, lessonLabel string, sectionCount int) string {
	visualSection := lesson.GetVisualSection(lj.currentSection)
	if visualSection == nil {
		return "No section available"
	}

	// Section title
	sectionTitleStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#3498DB")).
		Bold(true)

	sectionTitle := sectionTitleStyle.Render(visualSection.Title)

	// Build content
	var contentParts []string

	// Text before visual
	if visualSection.TextBefore != "" {
		textStyle := theme.Current.LessonText
		contentParts = append(contentParts, textStyle.Render(colorizeCards(visualSection.TextBefore)))
		contentParts = append(contentParts, "")
	}

	// Visual element
	if visualSection.Visual != nil {
		// Initialize visual view if needed
		if lj.visualView == nil || lj.visualView.Element != visualSection.Visual {
			boxWidth, boxHeight := 50, 10
			if visualSection.Visual.Type == tutorial.VisualTrumpHierarchy {
				boxWidth, boxHeight = 60, 16
			}
			lj.visualView = components.NewLessonVisualView(visualSection.Visual, boxWidth, boxHeight)
		}
		visualContent := lj.visualView.Render()
		contentParts = append(contentParts, visualContent)
		contentParts = append(contentParts, "")
	}

	// Text after visual
	if visualSection.TextAfter != "" {
		textStyle := theme.Current.LessonText
		contentParts = append(contentParts, textStyle.Render(colorizeCards(visualSection.TextAfter)))
	}

	// Join content parts
	contentText := strings.Join(contentParts, "\n")

	// Content box - use larger size for trump hierarchy
	boxWidth, boxHeight := 50, 10
	if visualSection.Visual != nil && visualSection.Visual.Type == tutorial.VisualTrumpHierarchy {
		boxWidth, boxHeight = 60, 16
	}
	contentStyle := lipgloss.NewStyle().
		Width(boxWidth).
		Height(boxHeight).
		Padding(1, 2).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#3498DB"))

	contentBox := contentStyle.Render(contentText)

	// Section progress within lesson
	sectionProgress := lj.renderSectionProgress(sectionCount, lj.currentSection)

	// Navigation hints
	var navHints string
	if lj.currentLesson == 0 && lj.currentSection == 0 {
		navHints = "                              Continue →"
	} else if lj.currentLesson == len(lj.allLessons)-1 && lj.currentSection == sectionCount-1 {
		navHints = "← Back                        Finish"
	} else {
		navHints = "← Back                        Continue →"
	}
	navStyle := theme.Current.Muted
	navHintsRendered := navStyle.Render(navHints)

	help := theme.Current.Help.Render("←/→: Navigate • Esc: Exit")

	content := lipgloss.PlaceHorizontal(width, lipgloss.Center, progressStr) + "\n" +
		lipgloss.PlaceHorizontal(width, lipgloss.Center, lessonLabel) + "\n\n" +
		lipgloss.PlaceHorizontal(width, lipgloss.Center, sectionTitle) + "\n" +
		lipgloss.PlaceHorizontal(width, lipgloss.Center, contentBox) + "\n" +
		lipgloss.PlaceHorizontal(width, lipgloss.Center, sectionProgress) + "\n" +
		lipgloss.PlaceHorizontal(width, lipgloss.Center, navHintsRendered) + "\n\n" +
		lipgloss.PlaceHorizontal(width, lipgloss.Center, help)

	return content
}

func (lj *LearningJourney) renderSectionProgress(total, current int) string {
	if total <= 1 {
		return ""
	}

	filled := current + 1
	barWidth := 20
	filledWidth := (filled * barWidth) / total
	emptyWidth := barWidth - filledWidth

	bar := strings.Repeat("█", filledWidth) + strings.Repeat("░", emptyWidth)

	progressStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#3498DB"))

	return progressStyle.Render(bar) + theme.Current.Muted.Render(" Section "+string(rune('0'+current+1))+" of "+string(rune('0'+total)))
}

func (lj *LearningJourney) renderCompletion(width int) string {
	titleStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#27AE60")).
		Bold(true)

	title := titleStyle.Render("Journey Complete!")

	message := theme.Current.LessonText.Render(`Congratulations!

You've completed all the lessons and
are ready to play Euchre!`)

	// All completed progress
	progress := components.NewJourneyProgress(len(lj.allLessons), len(lj.allLessons))
	progressStr := progress.RenderCompact()

	// Checkmarks under dots
	checkStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#27AE60"))
	checks := ""
	for i := 0; i < len(lj.allLessons); i++ {
		checks += checkStyle.Render("✓")
		if i < len(lj.allLessons)-1 {
			checks += "   "
		}
	}

	buttonStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#FFFFFF")).
		Background(lipgloss.Color("#27AE60")).
		Padding(0, 3).
		Bold(true)

	button := buttonStyle.Render("Start Playing")

	help := theme.Current.Help.Render("Enter: Main menu • p: Play game • r: Review lessons")

	content := lipgloss.PlaceHorizontal(width, lipgloss.Center, title) + "\n\n" +
		lipgloss.PlaceHorizontal(width, lipgloss.Center, message) + "\n\n" +
		lipgloss.PlaceHorizontal(width, lipgloss.Center, progressStr) + "\n" +
		lipgloss.PlaceHorizontal(width, lipgloss.Center, checks) + "\n\n" +
		lipgloss.PlaceHorizontal(width, lipgloss.Center, button) + "\n\n" +
		lipgloss.PlaceHorizontal(width, lipgloss.Center, help)

	return content
}

// renderWelcomeParts returns header, content, footer for welcome phase
func (lj *LearningJourney) renderWelcomeParts(width int) (string, string, string) {
	// Header: decorative cards + title + tagline
	decorativeCards := []engine.Card{
		{Suit: engine.Spades, Rank: engine.Jack},
		{Suit: engine.Hearts, Rank: engine.Jack},
		{Suit: engine.Diamonds, Rank: engine.Jack},
		{Suit: engine.Clubs, Rank: engine.Jack},
	}
	cardViews := make([]string, len(decorativeCards))
	for i, card := range decorativeCards {
		cv := components.NewCardView(card)
		cardViews[i] = cv.Render()
	}
	cardRow := lipgloss.JoinHorizontal(lipgloss.Top, cardViews...)

	titleStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#3498DB")).
		Bold(true)
	title := titleStyle.Render("Learn to Play Euchre")
	tagline := theme.Current.LessonText.Render("Master America's favorite trick-taking card game")

	header := lipgloss.PlaceHorizontal(width, lipgloss.Center, cardRow) + "\n\n" +
		lipgloss.PlaceHorizontal(width, lipgloss.Center, title) + "\n" +
		lipgloss.PlaceHorizontal(width, lipgloss.Center, tagline)

	// Content: lesson list + time + button
	var lessonList strings.Builder
	numberStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#3498DB")).
		Bold(true)
	lessonTitleStyle := theme.Current.LessonText

	for i, lesson := range lj.allLessons {
		num := numberStyle.Render(string(rune('1'+i)) + ".")
		lessonList.WriteString("   " + num + " " + lessonTitleStyle.Render(lesson.Title) + "\n")
	}

	contentBox := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#3498DB")).
		Padding(1, 2).
		Render(lessonList.String())

	timeEstimate := theme.Current.Muted.Render("◷ About 10 minutes")

	buttonStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#FFFFFF")).
		Background(lipgloss.Color("#27AE60")).
		Padding(0, 4).
		Bold(true)
	button := buttonStyle.Render("▶ Start Learning")

	content := lipgloss.PlaceHorizontal(width, lipgloss.Center, contentBox) + "\n\n" +
		lipgloss.PlaceHorizontal(width, lipgloss.Center, timeEstimate) + "\n\n" +
		lipgloss.PlaceHorizontal(width, lipgloss.Center, button)

	// Footer: help text
	footer := lipgloss.PlaceHorizontal(width, lipgloss.Center,
		theme.Current.Help.Render("Enter: Start • Esc: Back to menu"))

	return header, content, footer
}

// renderLessonContentParts returns header, content, footer for lesson phase
func (lj *LearningJourney) renderLessonContentParts(width int) (string, string, string) {
	if len(lj.allLessons) == 0 {
		return "", "No lessons available", ""
	}

	lesson := lj.allLessons[lj.currentLesson]

	// Header: progress + lesson label
	progress := components.NewJourneyProgress(len(lj.allLessons), lj.currentLesson)
	progressStr := progress.Render()
	lessonLabel := theme.Current.Primary.Render(
		strings.Repeat(" ", 4) + "Lesson " + string(rune('0'+lj.currentLesson+1)) + " of " + string(rune('0'+len(lj.allLessons))))

	header := lipgloss.PlaceHorizontal(width, lipgloss.Center, progressStr) + "\n" +
		lipgloss.PlaceHorizontal(width, lipgloss.Center, lessonLabel)

	// Footer: help text
	footer := lipgloss.PlaceHorizontal(width, lipgloss.Center,
		theme.Current.Help.Render("←/→: Navigate • ↑/↓: Scroll • Esc: Exit"))

	// Content varies based on whether lesson has visuals
	var content string
	if lesson.HasVisuals() {
		content = lj.renderVisualLessonBody(width, lesson)
	} else {
		content = lj.renderTextLessonBody(width, lesson)
	}

	return header, content, footer
}

// renderTextLessonBody renders the body content for text-based lessons
func (lj *LearningJourney) renderTextLessonBody(width int, lesson *tutorial.Lesson) string {
	section := lesson.Sections[lj.currentSection]

	sectionTitleStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#3498DB")).
		Bold(true)
	sectionTitle := sectionTitleStyle.Render(section.Title)

	boxWidth := 50
	boxHeight := 10
	contentStyle := lipgloss.NewStyle().
		Width(boxWidth).
		Height(boxHeight).
		Padding(1, 2).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#3498DB"))

	lines := strings.Split(section.Content, "\n")
	if lj.scroll > 0 && lj.scroll < len(lines) {
		lines = lines[lj.scroll:]
	}
	contentText := strings.Join(lines, "\n")
	contentText = colorizeCards(contentText)
	contentBox := contentStyle.Render(contentText)

	sectionProgress := lj.renderSectionProgress(len(lesson.Sections), lj.currentSection)
	navHints := lj.renderNavHints(lesson)

	return lipgloss.PlaceHorizontal(width, lipgloss.Center, sectionTitle) + "\n" +
		lipgloss.PlaceHorizontal(width, lipgloss.Center, contentBox) + "\n" +
		lipgloss.PlaceHorizontal(width, lipgloss.Center, sectionProgress) + "\n" +
		lipgloss.PlaceHorizontal(width, lipgloss.Center, navHints)
}

// renderVisualLessonBody renders the body content for visual lessons
func (lj *LearningJourney) renderVisualLessonBody(width int, lesson *tutorial.Lesson) string {
	visualSection := lesson.GetVisualSection(lj.currentSection)
	if visualSection == nil {
		return "No section available"
	}

	sectionTitleStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#3498DB")).
		Bold(true)
	sectionTitle := sectionTitleStyle.Render(visualSection.Title)

	boxWidth := 50
	boxHeight := 10
	if visualSection.Visual != nil && visualSection.Visual.Type == tutorial.VisualTrumpHierarchy {
		boxWidth = 60
		boxHeight = 16
	}

	contentStyle := lipgloss.NewStyle().
		Width(boxWidth).
		Height(boxHeight).
		Padding(1, 2).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#3498DB"))

	var contentParts []string
	if visualSection.TextBefore != "" {
		contentParts = append(contentParts, colorizeCards(visualSection.TextBefore))
	}
	if lj.visualView != nil {
		contentParts = append(contentParts, lj.visualView.Render())
	}
	if visualSection.TextAfter != "" {
		contentParts = append(contentParts, colorizeCards(visualSection.TextAfter))
	}

	contentBox := contentStyle.Render(strings.Join(contentParts, "\n\n"))

	sectionProgress := lj.renderSectionProgress(lesson.SectionCount(), lj.currentSection)
	navHints := lj.renderNavHints(lesson)

	return lipgloss.PlaceHorizontal(width, lipgloss.Center, sectionTitle) + "\n" +
		lipgloss.PlaceHorizontal(width, lipgloss.Center, contentBox) + "\n" +
		lipgloss.PlaceHorizontal(width, lipgloss.Center, sectionProgress) + "\n" +
		lipgloss.PlaceHorizontal(width, lipgloss.Center, navHints)
}

// renderNavHints returns navigation hints based on current position
func (lj *LearningJourney) renderNavHints(lesson *tutorial.Lesson) string {
	sectionCount := lesson.SectionCount()
	var navHints string
	if lj.currentLesson == 0 && lj.currentSection == 0 {
		navHints = "                              Continue →"
	} else if lj.currentLesson == len(lj.allLessons)-1 && lj.currentSection == sectionCount-1 {
		navHints = "← Back                        Finish"
	} else {
		navHints = "← Back                        Continue →"
	}
	return theme.Current.Muted.Render(navHints)
}

// renderCompletionParts returns header, content, footer for completion phase
func (lj *LearningJourney) renderCompletionParts(width int) (string, string, string) {
	// Header: title
	titleStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#27AE60")).
		Bold(true)
	title := titleStyle.Render("Journey Complete!")

	header := lipgloss.PlaceHorizontal(width, lipgloss.Center, title)

	// Content: message + progress + button
	message := theme.Current.LessonText.Render(`Congratulations!

You've completed all the lessons and
are ready to play Euchre!`)

	progress := components.NewJourneyProgress(len(lj.allLessons), len(lj.allLessons))
	progressStr := progress.RenderCompact()

	checkStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#27AE60"))
	checks := ""
	for i := 0; i < len(lj.allLessons); i++ {
		checks += checkStyle.Render("✓")
		if i < len(lj.allLessons)-1 {
			checks += "   "
		}
	}

	buttonStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#FFFFFF")).
		Background(lipgloss.Color("#27AE60")).
		Padding(0, 3).
		Bold(true)
	button := buttonStyle.Render("Start Playing")

	content := lipgloss.PlaceHorizontal(width, lipgloss.Center, message) + "\n\n" +
		lipgloss.PlaceHorizontal(width, lipgloss.Center, progressStr) + "\n" +
		lipgloss.PlaceHorizontal(width, lipgloss.Center, checks) + "\n\n" +
		lipgloss.PlaceHorizontal(width, lipgloss.Center, button)

	// Footer: help text
	footer := lipgloss.PlaceHorizontal(width, lipgloss.Center,
		theme.Current.Help.Render("Enter: Main menu • p: Play game • r: Review lessons"))

	return header, content, footer
}

// colorizeCards applies red coloring to hearts/diamonds and their card notations
func colorizeCards(text string) string {
	// Match card notations like "J♥", "A♦", "10♥", "9♦" etc.
	// Also match standalone suit symbols
	cardPattern := regexp.MustCompile(`(\d{1,2}|[JQKA])?([♥♦])`)

	return cardPattern.ReplaceAllStringFunc(text, func(match string) string {
		return theme.Current.CardRed.Render(match)
	})
}
