package app

import (
	tea "github.com/charmbracelet/bubbletea"
)

// Screen represents a screen in the application
type Screen int

const (
	ScreenMainMenu Screen = iota
	ScreenGameSetup
	ScreenGamePlay
	ScreenGameResult
	ScreenLearningJourney
	ScreenQuickReference
	ScreenSettings
)

// App is the root Bubble Tea model
type App struct {
	currentScreen Screen
	screenModels  map[Screen]tea.Model
	width         int
	height        int
	quitting      bool
}

// New creates a new App
func New() *App {
	app := &App{
		currentScreen: ScreenMainMenu,
		screenModels:  make(map[Screen]tea.Model),
	}

	// Initialize screen models
	app.screenModels[ScreenMainMenu] = NewMainMenu()

	return app
}

// Init implements tea.Model
func (a *App) Init() tea.Cmd {
	return nil
}

// Update implements tea.Model
func (a *App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			a.quitting = true
			return a, tea.Quit
		}

	case tea.WindowSizeMsg:
		a.width = msg.Width
		a.height = msg.Height

	case NavigateMsg:
		return a.navigate(msg.Screen, msg.Data)

	case QuitMsg:
		a.quitting = true
		return a, tea.Quit
	}

	// Update current screen
	if model, ok := a.screenModels[a.currentScreen]; ok {
		updatedModel, cmd := model.Update(msg)
		a.screenModels[a.currentScreen] = updatedModel
		return a, cmd
	}

	return a, nil
}

// View implements tea.Model
func (a *App) View() string {
	if a.quitting {
		return "Thanks for playing Euchre!\n"
	}

	if model, ok := a.screenModels[a.currentScreen]; ok {
		return model.View()
	}

	return "Loading..."
}

// navigate switches to a different screen
func (a *App) navigate(screen Screen, data interface{}) (*App, tea.Cmd) {
	// Create the screen model if it doesn't exist
	switch screen {
	case ScreenMainMenu:
		a.screenModels[screen] = NewMainMenu()
	case ScreenGameSetup:
		a.screenModels[screen] = NewGameSetup()
	case ScreenGamePlay:
		a.screenModels[screen] = NewGamePlay()
	case ScreenQuickReference:
		a.screenModels[screen] = NewQuickReference()
	case ScreenLearningJourney:
		a.screenModels[screen] = NewLearningJourney()
	}

	// Pass current window size to the new screen
	if a.width > 0 && a.height > 0 {
		if model, ok := a.screenModels[screen]; ok {
			updatedModel, _ := model.Update(tea.WindowSizeMsg{
				Width:  a.width,
				Height: a.height,
			})
			a.screenModels[screen] = updatedModel
		}
	}

	a.currentScreen = screen

	// Call Init on the new screen to start any async operations
	if model, ok := a.screenModels[screen]; ok {
		return a, model.Init()
	}
	return a, nil
}

// NavigateMsg is sent to navigate between screens
type NavigateMsg struct {
	Screen Screen
	Data   interface{}
}

// Navigate returns a command to navigate to a screen
func Navigate(screen Screen) tea.Cmd {
	return func() tea.Msg {
		return NavigateMsg{Screen: screen}
	}
}

// NavigateWithData returns a command to navigate with data
func NavigateWithData(screen Screen, data interface{}) tea.Cmd {
	return func() tea.Msg {
		return NavigateMsg{Screen: screen, Data: data}
	}
}

// QuitMsg signals the app to quit
type QuitMsg struct{}

// Quit returns a command to quit the app
func Quit() tea.Cmd {
	return func() tea.Msg {
		return QuitMsg{}
	}
}
