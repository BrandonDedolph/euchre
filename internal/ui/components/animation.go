package components

import (
	"time"

	"github.com/bran/euchre/internal/tutorial"
	tea "github.com/charmbracelet/bubbletea"
)

// AnimationController manages sequence playback for lesson animations
type AnimationController struct {
	Steps       []tutorial.SequenceStep
	CurrentStep int
	IsPaused    bool // Waiting for user to advance
	IsComplete  bool
	OnComplete  func()

	// Timing
	stepStartTime time.Time
	elapsedMs     int
}

// NewAnimationController creates a new animation controller
func NewAnimationController(steps []tutorial.SequenceStep) *AnimationController {
	return &AnimationController{
		Steps:       steps,
		CurrentStep: 0,
		IsPaused:    false,
		IsComplete:  len(steps) == 0,
	}
}

// AnimTickMsg is sent when the animation should advance
type AnimTickMsg struct {
	Time time.Time
}

// Start begins the animation sequence
func (a *AnimationController) Start() tea.Cmd {
	if len(a.Steps) == 0 {
		a.IsComplete = true
		return nil
	}

	a.CurrentStep = 0
	a.IsComplete = false
	a.stepStartTime = time.Now()
	a.elapsedMs = 0

	// Check if first step needs user input
	if a.Steps[0].PauseMs == 0 {
		a.IsPaused = true
		return nil
	}

	return a.tickCmd()
}

// Advance moves to the next step (for user-triggered advancement)
func (a *AnimationController) Advance() tea.Cmd {
	if a.IsComplete {
		return nil
	}

	// Only advance if paused (waiting for user)
	if !a.IsPaused {
		return nil
	}

	a.IsPaused = false
	a.CurrentStep++
	a.elapsedMs = 0
	a.stepStartTime = time.Now()

	if a.CurrentStep >= len(a.Steps) {
		a.IsComplete = true
		if a.OnComplete != nil {
			a.OnComplete()
		}
		return nil
	}

	// Check if next step needs user input
	if a.Steps[a.CurrentStep].PauseMs == 0 {
		a.IsPaused = true
		return nil
	}

	return a.tickCmd()
}

// Tick processes an animation tick
func (a *AnimationController) Tick() tea.Cmd {
	if a.IsComplete || a.IsPaused {
		return nil
	}

	if a.CurrentStep >= len(a.Steps) {
		a.IsComplete = true
		if a.OnComplete != nil {
			a.OnComplete()
		}
		return nil
	}

	step := a.Steps[a.CurrentStep]
	a.elapsedMs = int(time.Since(a.stepStartTime).Milliseconds())

	// Check if current step is done
	if step.PauseMs > 0 && a.elapsedMs >= step.PauseMs {
		a.CurrentStep++
		a.elapsedMs = 0
		a.stepStartTime = time.Now()

		if a.CurrentStep >= len(a.Steps) {
			a.IsComplete = true
			if a.OnComplete != nil {
				a.OnComplete()
			}
			return nil
		}

		// Check if next step needs user input
		if a.Steps[a.CurrentStep].PauseMs == 0 {
			a.IsPaused = true
			return nil
		}
	}

	return a.tickCmd()
}

// Reset resets the animation to the beginning
func (a *AnimationController) Reset() {
	a.CurrentStep = 0
	a.IsPaused = false
	a.IsComplete = len(a.Steps) == 0
	a.elapsedMs = 0
}

// GetCurrentStep returns the current sequence step
func (a *AnimationController) GetCurrentStep() *tutorial.SequenceStep {
	if a.CurrentStep >= len(a.Steps) {
		return nil
	}
	return &a.Steps[a.CurrentStep]
}

// GetCurrentMessage returns the message for the current step
func (a *AnimationController) GetCurrentMessage() string {
	step := a.GetCurrentStep()
	if step == nil {
		return ""
	}
	return step.Message
}

// GetProgress returns the progress as a fraction (0.0 to 1.0)
func (a *AnimationController) GetProgress() float64 {
	if len(a.Steps) == 0 {
		return 1.0
	}
	return float64(a.CurrentStep) / float64(len(a.Steps))
}

// tickCmd returns a command that ticks the animation
func (a *AnimationController) tickCmd() tea.Cmd {
	// Use a reasonable tick interval (60fps = ~16ms)
	return tea.Tick(time.Millisecond*16, func(t time.Time) tea.Msg {
		return AnimTickMsg{Time: t}
	})
}

// StepIndicator renders a progress indicator for the current step
func (a *AnimationController) StepIndicator() string {
	if len(a.Steps) == 0 {
		return ""
	}

	total := len(a.Steps)

	// Build dots
	dots := ""
	for i := 0; i < total; i++ {
		if i < a.CurrentStep {
			dots += "●"
		} else if i == a.CurrentStep && !a.IsComplete {
			dots += "◉"
		} else {
			dots += "○"
		}
		if i < total-1 {
			dots += " "
		}
	}

	return dots
}

// NeedsUserInput returns true if waiting for user to advance
func (a *AnimationController) NeedsUserInput() bool {
	return a.IsPaused && !a.IsComplete
}
