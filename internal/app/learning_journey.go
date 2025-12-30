package app

import (
	"regexp"
	"strings"

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

	// Create visual view
	boxWidth := max(60, lj.width*2/3)
	boxHeight := max(12, lj.height/2)
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

	var innerContent string
	switch lj.phase {
	case PhaseWelcome:
		innerContent = lj.renderWelcome(width - 4)
	case PhaseLessonContent:
		innerContent = lj.renderLessonContent(width - 4)
	case PhaseCompletion:
		innerContent = lj.renderCompletion(width - 4)
	}

	// Center content and wrap in screen border
	centeredContent := lipgloss.Place(width-4, height-4, lipgloss.Center, lipgloss.Center, innerContent)
	screenBox := theme.Current.ScreenBorder.
		Width(width - 2).
		Height(height - 2).
		Render(centeredContent)

	return lipgloss.Place(width, height, lipgloss.Center, lipgloss.Center, screenBox)
}

func (lj *LearningJourney) renderWelcome(width int) string {
	titleStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#3498DB")).
		Bold(true)

	title := titleStyle.Render("Welcome to Your Euchre Journey")

	intro := theme.Current.Body.Render(`You're about to learn one of America's
most beloved card games in a few easy steps.`)

	// Build lesson list
	var lessonList strings.Builder
	lessonList.WriteString(theme.Current.Subtitle.Render("What you'll learn:") + "\n\n")

	bulletStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#95A5A6"))
	for _, lesson := range lj.allLessons {
		lessonList.WriteString(bulletStyle.Render("  ○ ") + lesson.Title + "\n")
	}

	timeEstimate := theme.Current.Muted.Render("Estimated time: ~10 minutes")

	buttonStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#FFFFFF")).
		Background(lipgloss.Color("#3498DB")).
		Padding(0, 3).
		Bold(true)

	button := buttonStyle.Render("Begin Your Journey")

	help := theme.Current.Help.Render("Press Enter to start • Esc to go back")

	content := lipgloss.PlaceHorizontal(width, lipgloss.Center, title) + "\n\n" +
		lipgloss.PlaceHorizontal(width, lipgloss.Center, intro) + "\n\n" +
		lipgloss.PlaceHorizontal(width, lipgloss.Center, lessonList.String()) + "\n" +
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

	// Content box with dynamic size (min 60x12)
	boxWidth := max(60, width*2/3)
	boxHeight := max(12, lj.height/2)
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
		textStyle := theme.Current.Body
		contentParts = append(contentParts, textStyle.Render(colorizeCards(visualSection.TextBefore)))
		contentParts = append(contentParts, "")
	}

	// Visual element
	if visualSection.Visual != nil {
		// Initialize visual view if needed
		if lj.visualView == nil || lj.visualView.Element != visualSection.Visual {
			boxWidth := max(60, width*2/3)
			boxHeight := max(12, lj.height/2)
			lj.visualView = components.NewLessonVisualView(visualSection.Visual, boxWidth, boxHeight)
		}
		visualContent := lj.visualView.Render()
		contentParts = append(contentParts, visualContent)
		contentParts = append(contentParts, "")
	}

	// Text after visual
	if visualSection.TextAfter != "" {
		textStyle := theme.Current.Body
		contentParts = append(contentParts, textStyle.Render(colorizeCards(visualSection.TextAfter)))
	}

	// Join content parts
	contentText := strings.Join(contentParts, "\n")

	// Content box
	boxWidth := max(60, width*2/3)
	boxHeight := max(14, lj.height/2)
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

	message := theme.Current.Body.Render(`Congratulations!

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

// colorizeCards applies red coloring to hearts/diamonds and their card notations
func colorizeCards(text string) string {
	// Match card notations like "J♥", "A♦", "10♥", "9♦" etc.
	// Also match standalone suit symbols
	cardPattern := regexp.MustCompile(`(\d{1,2}|[JQKA])?([♥♦])`)

	return cardPattern.ReplaceAllStringFunc(text, func(match string) string {
		return theme.Current.CardRed.Render(match)
	})
}
