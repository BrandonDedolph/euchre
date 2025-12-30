package app

import (
	"fmt"
	"strings"
	"time"

	"github.com/bran/euchre/internal/ai"
	"github.com/bran/euchre/internal/ai/rule_based"
	"github.com/bran/euchre/internal/engine"
	"github.com/bran/euchre/internal/ui/components"
	"github.com/bran/euchre/internal/ui/theme"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

const aiTurnDelay = 500 * time.Millisecond
const aiBidDelay = 1200 * time.Millisecond // Slower for bidding so user can follow
const dealCardDelay = 100 * time.Millisecond

// Animation timing constants
const (
	shuffleFrameDelay  = 80 * time.Millisecond
	shuffleTotalFrames = 24
	trumpFlashDelay    = 200 * time.Millisecond
	trumpFlashTotal    = 8
	scoreAnimDelay     = 150 * time.Millisecond
	scoreAnimTotal     = 10
	cardPlayDelay      = 60 * time.Millisecond
	cardPlayFrames     = 5
	trickCollectDelay  = 80 * time.Millisecond
	trickCollectFrames = 4
	turnPulseDelay     = 300 * time.Millisecond
	celebrationDelay   = 100 * time.Millisecond
	celebrationTotal   = 20
	cardFlipDelay      = 100 * time.Millisecond
	cardFlipTotal      = 4
)

// GamePlay is the main game playing screen
type GamePlay struct {
	game               *engine.Game
	aiPlayers          []ai.Player
	humanPlayer        int
	selectedCard       int
	message            string
	tableView          *components.TableView
	width              int
	height             int
	waitingForTrickAck bool                // Waiting for user to acknowledge trick result
	completedTrick     *engine.TrickResult // The completed trick to display
	isDealing          bool                // Currently animating the deal
	dealStep           int                 // Current step in deal animation (0-19: 20 cards dealt)
	waitingForRoundAck bool                // Waiting for user to acknowledge round result

	// Animation states
	isShuffling       bool   // Shuffle animation before dealing
	shuffleStep       int    // Current shuffle animation frame
	trumpFlashFrames  int    // Frames remaining for trump flash effect
	scoreAnimFrames   int    // Frames remaining for score animation
	scoreDelta        [2]int // Score change to animate [team0, team1]
	previousScores    [2]int // Scores before the animation
	turnPulseFrame    int    // Frame counter for turn indicator pulse
	celebrationFrames int    // Frames remaining for winner celebration
	cardFlipFrames    int    // Frames remaining for card flip reveal

	// Suit selector for bidding round 2
	suitSelector *components.SuitSelector
}

// NewGamePlay creates a new game play screen
func NewGamePlay() *GamePlay {
	config := engine.DefaultGameConfig()
	game := engine.NewGame(config)

	gp := &GamePlay{
		game:         game,
		humanPlayer:  0, // Player 0 is the human
		aiPlayers:    rule_based.CreateAIPlayers(0, ai.DifficultyMedium),
		selectedCard: 0,
		tableView:    components.NewTableView(),
		isShuffling:  true, // Start with shuffle animation
		shuffleStep:  0,
		isDealing:    false,
		dealStep:     0,
	}

	// Start the first round (cards are dealt in engine, animation is visual only)
	game.StartRound()
	gp.updateDealingView() // Show empty hands initially

	return gp
}

// Init implements tea.Model
func (g *GamePlay) Init() tea.Cmd {
	// Start turn pulse animation
	pulseCmd := tea.Tick(turnPulseDelay, func(t time.Time) tea.Msg {
		return turnPulseTickMsg{}
	})

	if g.isShuffling {
		shuffleCmd := tea.Tick(shuffleFrameDelay, func(t time.Time) tea.Msg {
			return shuffleTickMsg{}
		})
		return tea.Batch(shuffleCmd, pulseCmd)
	}
	if g.isDealing {
		return tea.Batch(g.nextDealCard(), pulseCmd)
	}
	return tea.Batch(g.processAITurns(), pulseCmd)
}

// Update implements tea.Model
func (g *GamePlay) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		g.width = msg.Width
		g.height = msg.Height

	case tea.KeyMsg:
		return g.handleKeyPress(msg)

	case aiTurnMsg:
		// AI made a move, add delay before continuing
		return g, tea.Tick(aiTurnDelay, func(t time.Time) tea.Msg {
			return aiContinueMsg{}
		})

	case aiBidMsg:
		// AI made a bid, use longer delay so user can follow
		g.message = msg.message
		g.updateTableView()
		return g, tea.Tick(aiBidDelay, func(t time.Time) tea.Msg {
			return aiContinueMsg{}
		})

	case aiContinueMsg:
		// Continue processing AI turns after delay
		return g, g.processAITurns()

	case humanTurnMsg:
		// It's the human's turn - update the display
		g.updateTableView()
		return g, nil

	case tempMessageMsg:
		// Restore the original message after showing a temporary message
		g.message = msg.originalMsg
		return g, nil

	case dealCardMsg:
		// Animate dealing one card
		if !g.isDealing {
			// Already done dealing, ignore stale message
			return g, nil
		}
		g.dealStep++
		g.updateDealingView()
		if g.dealStep >= 20 { // All 20 cards dealt (5 per player)
			g.isDealing = false
			g.message = "Revealing turned card..."
			// Start card flip animation
			g.cardFlipFrames = cardFlipTotal
			g.tableView.CardFlipFrames = cardFlipTotal
			g.tableView.CardFlipTotal = cardFlipTotal
			g.updateTableView()
			return g, tea.Tick(cardFlipDelay, func(t time.Time) tea.Msg {
				return cardFlipTickMsg{}
			})
		}
		return g, g.nextDealCard()

	case trickDoneMsg:
		// Show completed trick and wait for acknowledgment
		g.waitingForTrickAck = true
		g.completedTrick = &msg.result
		g.tableView.CurrentTrick = msg.result.Cards
		winnerName := g.tableView.PlayerNames[msg.result.Winner]

		// Use correct grammar: "You win" vs "East wins"
		verb := "wins"
		if msg.result.Winner == g.humanPlayer {
			verb = "win"
		}

		// Build a clear message about the trick result
		trickMsg := fmt.Sprintf("%s %s the trick", winnerName, verb)
		if msg.result.WasTrumped {
			trickMsg += " with trump"
		}
		g.message = trickMsg
		return g, nil

	case roundCompleteMsg:
		// Show round results and wait for acknowledgment
		g.waitingForRoundAck = true
		scores := g.game.Scores()

		// Calculate score delta and start animation
		g.scoreDelta[0] = scores[0] - g.previousScores[0]
		g.scoreDelta[1] = scores[1] - g.previousScores[1]
		g.previousScores[0] = scores[0]
		g.previousScores[1] = scores[1]
		if g.scoreDelta[0] != 0 || g.scoreDelta[1] != 0 {
			g.scoreAnimFrames = scoreAnimTotal
		}

		// Get round result details
		roundHistory := g.game.RoundHistory()
		var roundMsg string
		if len(roundHistory) > 0 {
			lastRound := roundHistory[len(roundHistory)-1]
			yourTeamMade := lastRound.Makers == 0

			if lastRound.WasEuchred {
				if yourTeamMade {
					roundMsg = "Euchred! Opponents score 2 points."
				} else {
					roundMsg = "You euchred them! +2 points!"
				}
			} else if lastRound.MakerTricks == 5 {
				if yourTeamMade {
					if lastRound.WasAlone {
						roundMsg = "March going alone! +4 points!"
					} else {
						roundMsg = "March! +2 points!"
					}
				} else {
					if lastRound.WasAlone {
						roundMsg = "Opponents march alone for 4 points."
					} else {
						roundMsg = "Opponents march for 2 points."
					}
				}
			} else {
				if yourTeamMade {
					roundMsg = fmt.Sprintf("Made it with %d tricks. +1 point.", lastRound.MakerTricks)
				} else {
					roundMsg = fmt.Sprintf("Opponents made it with %d tricks.", lastRound.MakerTricks)
				}
			}
		}

		// Build command for score animation
		var cmd tea.Cmd
		if g.scoreAnimFrames > 0 {
			cmd = tea.Tick(scoreAnimDelay, func(t time.Time) tea.Msg {
				return scoreAnimTickMsg{}
			})
		}

		if g.game.IsOver() {
			winner := g.game.Winner()
			if winner == 0 {
				g.message = fmt.Sprintf("%s Game Over! Your team wins %d-%d!", roundMsg, scores[0], scores[1])
				// Trigger celebration animation for winning
				g.celebrationFrames = celebrationTotal
				celebCmd := tea.Tick(celebrationDelay, func(t time.Time) tea.Msg {
					return celebrationTickMsg{}
				})
				if cmd != nil {
					return g, tea.Batch(cmd, celebCmd)
				}
				return g, celebCmd
			} else {
				g.message = fmt.Sprintf("%s Game Over! Opponents win %d-%d.", roundMsg, scores[1], scores[0])
			}
		} else {
			g.message = fmt.Sprintf("%s Score: You %d - Opponents %d", roundMsg, scores[0], scores[1])
		}
		return g, cmd

	case aiErrorMsg:
		// AI action failed - display error and return to menu
		g.message = fmt.Sprintf("AI error (%s): %v", msg.action, msg.err)
		return g, Navigate(ScreenMainMenu)

	case aiCardPlayMsg:
		// AI played a card, animate then continue
		g.updateTableView()
		return g, tea.Batch(
			tea.Tick(cardPlayDelay, func(t time.Time) tea.Msg { return cardPlayTickMsg{} }),
			tea.Tick(cardPlayDelay*time.Duration(cardPlayFrames+2), func(t time.Time) tea.Msg {
				return aiContinueMsg{}
			}),
		)

	case aiCardPlayWithTrickMsg:
		// AI played a card that completed a trick
		// Don't call updateTableView() - it would reset CurrentTrick to the new empty trick
		// CurrentTrick was already set to result.Cards in processAITurns
		result := msg.result
		return g, tea.Batch(
			tea.Tick(cardPlayDelay, func(t time.Time) tea.Msg { return cardPlayTickMsg{} }),
			tea.Tick(cardPlayDelay*time.Duration(cardPlayFrames+2), func(t time.Time) tea.Msg {
				return trickDoneMsg{result: result}
			}),
		)

	// Animation handlers
	case shuffleTickMsg:
		g.shuffleStep++
		if g.shuffleStep >= shuffleTotalFrames {
			g.isShuffling = false
			g.isDealing = true
			g.dealStep = 0
			g.updateDealingView()
			return g, g.nextDealCard()
		}
		return g, tea.Tick(shuffleFrameDelay, func(t time.Time) tea.Msg {
			return shuffleTickMsg{}
		})

	case trumpFlashTickMsg:
		g.trumpFlashFrames--
		if g.trumpFlashFrames <= 0 {
			return g, g.processAITurns()
		}
		return g, tea.Tick(trumpFlashDelay, func(t time.Time) tea.Msg {
			return trumpFlashTickMsg{}
		})

	case cardFlipTickMsg:
		g.cardFlipFrames--
		g.tableView.CardFlipFrames = g.cardFlipFrames
		if g.cardFlipFrames <= 0 {
			// Flip animation done, update table and continue to bidding
			g.message = ""
			g.updateTableView()
			return g, tea.Tick(500*time.Millisecond, func(t time.Time) tea.Msg {
				return aiContinueMsg{}
			})
		}
		return g, tea.Tick(cardFlipDelay, func(t time.Time) tea.Msg {
			return cardFlipTickMsg{}
		})

	case scoreAnimTickMsg:
		g.scoreAnimFrames--
		if g.scoreAnimFrames <= 0 {
			return g, nil
		}
		return g, tea.Tick(scoreAnimDelay, func(t time.Time) tea.Msg {
			return scoreAnimTickMsg{}
		})

	case cardPlayTickMsg:
		if g.tableView.CardPlayAnim != nil {
			g.tableView.CardPlayAnim.Frame++
			if g.tableView.CardPlayAnim.Frame >= g.tableView.CardPlayAnim.TotalFrames {
				g.tableView.CardPlayAnim = nil
				return g, nil
			}
			return g, tea.Tick(cardPlayDelay, func(t time.Time) tea.Msg {
				return cardPlayTickMsg{}
			})
		}
		return g, nil

	case trickCollectTickMsg:
		if g.tableView.TrickCollectAnim != nil {
			g.tableView.TrickCollectAnim.Frame++
			if g.tableView.TrickCollectAnim.Frame >= g.tableView.TrickCollectAnim.TotalFrames {
				g.tableView.TrickCollectAnim = nil
				// Don't clear CurrentTrick here - updateTableView will set it from the round
				// This prevents flicker between tricks
				g.updateTableView()
				return g, g.processAITurns()
			}
			return g, tea.Tick(trickCollectDelay, func(t time.Time) tea.Msg {
				return trickCollectTickMsg{}
			})
		}
		return g, nil

	case turnPulseTickMsg:
		g.turnPulseFrame++
		// Continue pulsing while game is active
		if !g.game.IsOver() && !g.waitingForRoundAck {
			return g, tea.Tick(turnPulseDelay, func(t time.Time) tea.Msg {
				return turnPulseTickMsg{}
			})
		}
		return g, nil

	case celebrationTickMsg:
		g.celebrationFrames--
		if g.celebrationFrames <= 0 {
			return g, nil
		}
		return g, tea.Tick(celebrationDelay, func(t time.Time) tea.Msg {
			return celebrationTickMsg{}
		})
	}

	return g, nil
}

// handleKeyPress handles keyboard input
func (g *GamePlay) handleKeyPress(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// During dealing animation, only allow quit
	if g.isDealing {
		if msg.String() == "q" || msg.String() == "esc" {
			return g, Navigate(ScreenMainMenu)
		}
		return g, nil
	}

	// If waiting for round acknowledgment, Enter continues or exits
	if g.waitingForRoundAck {
		switch msg.String() {
		case "enter", " ":
			g.waitingForRoundAck = false
			g.message = ""
			if g.game.IsOver() {
				return g, Navigate(ScreenMainMenu)
			}
			// Start next round with shuffle animation
			g.game.StartRound()
			g.isShuffling = true
			g.shuffleStep = 0
			g.isDealing = false
			g.updateDealingView()
			return g, tea.Tick(shuffleFrameDelay, func(t time.Time) tea.Msg {
				return shuffleTickMsg{}
			})
		case "q", "esc":
			return g, Navigate(ScreenMainMenu)
		}
		return g, nil
	}

	// If waiting for trick acknowledgment, Enter continues
	if g.waitingForTrickAck {
		switch msg.String() {
		case "enter", " ":
			g.waitingForTrickAck = false
			g.message = ""

			// Start trick collection animation
			if g.completedTrick != nil {
				g.tableView.TrickCollectAnim = &components.TrickCollectAnim{
					Winner:      g.completedTrick.Winner,
					Cards:       g.completedTrick.Cards,
					Frame:       0,
					TotalFrames: trickCollectFrames,
				}
				g.completedTrick = nil

				// Check if round is complete after animation
				if g.game.NeedsNewRound() {
					return g, tea.Batch(
						tea.Tick(trickCollectDelay, func(t time.Time) tea.Msg { return trickCollectTickMsg{} }),
						tea.Tick(trickCollectDelay*time.Duration(trickCollectFrames+1), func(t time.Time) tea.Msg {
							return roundCompleteMsg{}
						}),
					)
				}
				return g, tea.Tick(trickCollectDelay, func(t time.Time) tea.Msg { return trickCollectTickMsg{} })
			}

			g.completedTrick = nil
			g.updateTableView()
			// Check if round is complete
			if g.game.NeedsNewRound() {
				return g, func() tea.Msg { return roundCompleteMsg{} }
			}
			return g, g.processAITurns()
		case "q", "esc":
			return g, Navigate(ScreenMainMenu)
		}
		return g, nil
	}

	// Check if card selection should be allowed (only during discard/play phases when it's your turn)
	phase := g.game.Phase()
	canSelectCard := (phase == engine.PhaseDiscard || phase == engine.PhasePlay) &&
		g.game.CurrentPlayer() == g.humanPlayer

	// Initialize suit selector for round 2 bidding if needed
	if phase == engine.PhaseBidRound2 && g.game.CurrentPlayer() == g.humanPlayer && g.suitSelector == nil {
		g.suitSelector = components.NewSuitSelector(g.game.TurnedCard().Suit)
	}

	switch msg.String() {
	case "q", "esc":
		return g, Navigate(ScreenMainMenu)

	case "left", "h":
		if phase == engine.PhaseBidRound2 && g.suitSelector != nil && g.game.CurrentPlayer() == g.humanPlayer {
			g.suitSelector.MoveLeft()
		} else if canSelectCard && g.selectedCard > 0 {
			g.selectedCard--
		}

	case "right", "l":
		if phase == engine.PhaseBidRound2 && g.suitSelector != nil && g.game.CurrentPlayer() == g.humanPlayer {
			g.suitSelector.MoveRight()
		} else if canSelectCard {
			hand := g.game.Hand(g.humanPlayer)
			if g.selectedCard < len(hand)-1 {
				g.selectedCard++
			}
		}

	case "enter", " ":
		return g.handleAction()

	case "p":
		// Pass during bidding
		return g.handlePass()

	case "a":
		// Go alone during bidding
		return g.handleAlone()

	case "c":
		// Call clubs in round 2 (keep keyboard shortcuts as alternative)
		return g.handleCallSuit(engine.Clubs, false)

	case "d":
		// Call diamonds in round 2
		return g.handleCallSuit(engine.Diamonds, false)

	case "s":
		// Call spades in round 2
		return g.handleCallSuit(engine.Spades, false)
	}

	return g, nil
}

// handleAction handles the main action (playing a card or ordering up)
func (g *GamePlay) handleAction() (tea.Model, tea.Cmd) {
	phase := g.game.Phase()
	currentPlayer := g.game.CurrentPlayer()

	if currentPlayer != g.humanPlayer {
		return g.showTempMessage("Not your turn")
	}

	switch phase {
	case engine.PhaseBidRound1:
		// Order up
		action := engine.OrderUpAction{
			PlayerIdx: g.humanPlayer,
			Alone:     false,
		}
		if err := g.game.ApplyAction(action); err != nil {
			g.message = err.Error()
		} else {
			g.message = "You ordered it up!"
			g.updateTableView()
		}

	case engine.PhaseBidRound2:
		// Use suit selector to call trump
		if g.suitSelector == nil {
			g.suitSelector = components.NewSuitSelector(g.game.TurnedCard().Suit)
		}
		suit := g.suitSelector.SelectedSuit()
		action := engine.CallTrumpAction{
			PlayerIdx: g.humanPlayer,
			Suit:      suit,
			Alone:     false,
		}
		if err := g.game.ApplyAction(action); err != nil {
			g.message = err.Error()
		} else {
			g.message = fmt.Sprintf("You called %s!", suit)
			g.suitSelector = nil // Reset for next time
			g.updateTableView()
		}

	case engine.PhaseDiscard:
		// Discard the selected card
		hand := g.game.Hand(g.humanPlayer)
		if g.selectedCard >= 0 && g.selectedCard < len(hand) {
			card := hand[g.selectedCard]
			action := engine.DiscardAction{
				PlayerIdx: g.humanPlayer,
				Card:      card,
			}
			if err := g.game.ApplyAction(action); err != nil {
				g.message = err.Error()
			} else {
				g.message = fmt.Sprintf("Discarded %s", card)
				g.selectedCard = 0
				g.updateTableView()
			}
		}

	case engine.PhasePlay:
		// Play the selected card
		hand := g.game.Hand(g.humanPlayer)
		if g.selectedCard >= 0 && g.selectedCard < len(hand) {
			card := hand[g.selectedCard]

			// Track trick history to detect completion
			round := g.game.Round()
			historyLen := 0
			if round != nil {
				historyLen = len(round.TrickHistory())
			}

			action := engine.PlayCardAction{
				PlayerIdx: g.humanPlayer,
				Card:      card,
			}
			if err := g.game.ApplyAction(action); err != nil {
				return g.showTempMessage(err.Error())
			}
			g.selectedCard = 0

			// Start card play animation
			g.tableView.CardPlayAnim = &components.CardPlayAnim{
				Card:        card,
				FromPlayer:  g.humanPlayer,
				Frame:       0,
				TotalFrames: cardPlayFrames,
			}

			// Check if trick just completed
			if round != nil && len(round.TrickHistory()) > historyLen {
				history := round.TrickHistory()
				result := history[len(history)-1]
				// Set CurrentTrick to the completed trick so it displays during animation
				g.tableView.CurrentTrick = result.Cards
				// Delay trick done message to allow card animation to finish
				return g, tea.Batch(
					tea.Tick(cardPlayDelay, func(t time.Time) tea.Msg { return cardPlayTickMsg{} }),
					tea.Tick(cardPlayDelay*time.Duration(cardPlayFrames+1), func(t time.Time) tea.Msg {
						return trickDoneMsg{result: result}
					}),
				)
			}
			// Trick not complete - update table and animate
			g.updateTableView()
			return g, tea.Batch(
				tea.Tick(cardPlayDelay, func(t time.Time) tea.Msg { return cardPlayTickMsg{} }),
				tea.Tick(cardPlayDelay*time.Duration(cardPlayFrames+2), func(t time.Time) tea.Msg {
					return aiContinueMsg{}
				}),
			)
		}
	}

	return g, g.processAITurns()
}

// handlePass handles passing during bidding
func (g *GamePlay) handlePass() (tea.Model, tea.Cmd) {
	phase := g.game.Phase()
	if phase != engine.PhaseBidRound1 && phase != engine.PhaseBidRound2 {
		return g, nil
	}

	if g.game.CurrentPlayer() != g.humanPlayer {
		return g.showTempMessage("Not your turn")
	}

	action := engine.PassAction{PlayerIdx: g.humanPlayer}
	if err := g.game.ApplyAction(action); err != nil {
		g.message = err.Error()
	} else {
		g.message = "You passed"
		g.updateTableView()
	}

	return g, g.processAITurns()
}

// handleAlone handles going alone
func (g *GamePlay) handleAlone() (tea.Model, tea.Cmd) {
	phase := g.game.Phase()
	if phase != engine.PhaseBidRound1 && phase != engine.PhaseBidRound2 {
		return g, nil
	}

	if g.game.CurrentPlayer() != g.humanPlayer {
		return g.showTempMessage("Not your turn")
	}

	if phase == engine.PhaseBidRound1 {
		action := engine.OrderUpAction{
			PlayerIdx: g.humanPlayer,
			Alone:     true,
		}
		if err := g.game.ApplyAction(action); err != nil {
			g.message = err.Error()
		} else {
			g.message = "Going alone!"
			g.updateTableView()
		}
	}

	return g, g.processAITurns()
}

// handleCallSuit handles calling a specific suit in round 2
func (g *GamePlay) handleCallSuit(suit engine.Suit, alone bool) (tea.Model, tea.Cmd) {
	phase := g.game.Phase()
	if phase != engine.PhaseBidRound2 {
		return g, nil
	}

	if g.game.CurrentPlayer() != g.humanPlayer {
		return g.showTempMessage("Not your turn")
	}

	// Can't call the turned card's suit in round 2
	if suit == g.game.TurnedCard().Suit {
		g.message = fmt.Sprintf("Can't call %s - it was turned down", suit)
		return g, nil
	}

	action := engine.CallTrumpAction{
		PlayerIdx: g.humanPlayer,
		Suit:      suit,
		Alone:     alone,
	}
	if err := g.game.ApplyAction(action); err != nil {
		g.message = err.Error()
		return g, nil
	}

	g.message = fmt.Sprintf("You called %s!", suit)
	g.suitSelector = nil // Reset suit selector
	g.updateTableView()
	return g, g.processAITurns()
}

// processAITurns processes a single AI player turn and returns a message to continue
func (g *GamePlay) processAITurns() tea.Cmd {
	return func() tea.Msg {
		if g.game.IsOver() || g.game.NeedsNewRound() {
			return roundCompleteMsg{}
		}

		current := g.game.CurrentPlayer()
		if current == g.humanPlayer || current < 0 {
			return humanTurnMsg{} // Human's turn - update display
		}

		aiPlayer := g.aiPlayers[current]
		if aiPlayer == nil {
			return humanTurnMsg{} // Fallback to human turn
		}

		state := engine.NewGameState(g.game)
		phase := g.game.Phase()

		switch phase {
		case engine.PhaseBidRound1, engine.PhaseBidRound2:
			round := 1
			if phase == engine.PhaseBidRound2 {
				round = 2
			}
			decision := aiPlayer.DecideBid(state, round)
			if err := g.applyAIBidDecision(current, decision, phase); err != nil {
				return aiErrorMsg{err: err, player: current, action: "bid"}
			}

			// Build message about what AI decided
			playerName := g.tableView.PlayerNames[current]
			var bidMsg string
			if decision.Pass {
				bidMsg = fmt.Sprintf("%s passes", playerName)
			} else if decision.OrderUp {
				if decision.Alone {
					bidMsg = fmt.Sprintf("%s orders it up alone!", playerName)
				} else {
					bidMsg = fmt.Sprintf("%s orders it up", playerName)
				}
			} else {
				if decision.Alone {
					bidMsg = fmt.Sprintf("%s calls %s alone!", playerName, decision.CallSuit)
				} else {
					bidMsg = fmt.Sprintf("%s calls %s", playerName, decision.CallSuit)
				}
			}
			g.updateTableView()
			return aiBidMsg{message: bidMsg}

		case engine.PhaseDiscard:
			hand := g.game.Hand(current)
			card := aiPlayer.DecideDiscard(state, hand)
			action := engine.DiscardAction{PlayerIdx: current, Card: card}
			if err := g.game.ApplyAction(action); err != nil {
				return aiErrorMsg{err: err, player: current, action: "discard"}
			}

		case engine.PhasePlay:
			// Track trick history to detect completion
			round := g.game.Round()
			historyLen := 0
			if round != nil {
				historyLen = len(round.TrickHistory())
			}

			card := aiPlayer.DecidePlay(state)
			action := engine.PlayCardAction{PlayerIdx: current, Card: card}
			if err := g.game.ApplyAction(action); err != nil {
				return aiErrorMsg{err: err, player: current, action: "play"}
			}

			// Start card play animation for AI
			g.tableView.CardPlayAnim = &components.CardPlayAnim{
				Card:        card,
				FromPlayer:  current,
				Frame:       0,
				TotalFrames: cardPlayFrames,
			}

			// Check if trick just completed
			if round != nil && len(round.TrickHistory()) > historyLen {
				history := round.TrickHistory()
				result := history[len(history)-1]
				// Set CurrentTrick to the completed trick so it displays during animation
				g.tableView.CurrentTrick = result.Cards
				return aiCardPlayWithTrickMsg{card: card, player: current, result: result}
			}
			return aiCardPlayMsg{card: card, player: current}
		}

		g.updateTableView()
		return aiTurnMsg{} // Signal that an AI turn was completed, continue processing
	}
}

// applyAIBidDecision applies an AI's bidding decision
func (g *GamePlay) applyAIBidDecision(playerIdx int, decision engine.BidDecision, phase engine.GamePhase) error {
	if decision.Pass {
		action := engine.PassAction{PlayerIdx: playerIdx}
		return g.game.ApplyAction(action)
	}

	if phase == engine.PhaseBidRound1 && decision.OrderUp {
		action := engine.OrderUpAction{
			PlayerIdx: playerIdx,
			Alone:     decision.Alone,
		}
		return g.game.ApplyAction(action)
	} else if phase == engine.PhaseBidRound2 {
		action := engine.CallTrumpAction{
			PlayerIdx: playerIdx,
			Suit:      decision.CallSuit,
			Alone:     decision.Alone,
		}
		return g.game.ApplyAction(action)
	}
	return nil
}

// selectBestTrump selects the best trump suit for the player
func (g *GamePlay) selectBestTrump(hand []engine.Card, excludeSuit engine.Suit) engine.Suit {
	bestSuit := engine.Clubs
	bestCount := 0

	for _, suit := range []engine.Suit{engine.Clubs, engine.Diamonds, engine.Hearts, engine.Spades} {
		if suit == excludeSuit {
			continue
		}

		count := 0
		for _, card := range hand {
			if card.Suit == suit || card.IsLeftBower(suit) {
				count++
			}
		}

		if count > bestCount {
			bestCount = count
			bestSuit = suit
		}
	}

	return bestSuit
}

// showTempMessage shows a temporary message that reverts after a delay
func (g *GamePlay) showTempMessage(msg string) (tea.Model, tea.Cmd) {
	originalMsg := g.message
	g.message = msg
	return g, tea.Tick(1*time.Second, func(t time.Time) tea.Msg {
		return tempMessageMsg{originalMsg: originalMsg}
	})
}

// updateTableView updates the table view with current game state
func (g *GamePlay) updateTableView() {
	round := g.game.Round()
	if round == nil {
		return
	}

	g.tableView.Trump = g.game.Trump()
	g.tableView.TurnedCard = g.game.TurnedCard()
	g.tableView.Dealer = g.game.Dealer()
	g.tableView.CurrentPlayer = g.game.CurrentPlayer()
	g.tableView.Maker = round.Maker()
	g.tableView.MakerAlone = round.IsAlone()
	g.tableView.TurnPulseFrame = g.turnPulseFrame
	g.tableView.RoundNumber = len(g.game.RoundHistory()) + 1 // Current round = completed rounds + 1

	// Update player hand counts
	for i := 0; i < 4; i++ {
		g.tableView.PlayerHands[i] = len(g.game.Hand(i))
		g.tableView.TricksWon[i] = round.TricksWon(i)
	}

	// Update current trick
	g.tableView.CurrentTrick = round.CurrentTrick()
}

// nextDealCard returns a command to deal the next card after a delay
func (g *GamePlay) nextDealCard() tea.Cmd {
	return tea.Tick(dealCardDelay, func(t time.Time) tea.Msg {
		return dealCardMsg{}
	})
}

// updateDealingView updates the table view during dealing animation
func (g *GamePlay) updateDealingView() {
	dealer := g.game.Dealer()

	// Calculate how many cards each player has been dealt so far
	// Cards are dealt in rotation starting left of dealer
	// Cap dealStep at 20 to ensure max 5 cards per player
	step := g.dealStep
	if step > 20 {
		step = 20
	}

	cardCounts := [4]int{0, 0, 0, 0}
	for i := 0; i < step; i++ {
		// Player index: rotate starting from left of dealer
		playerIdx := (dealer + 1 + (i % 4)) % 4
		cardCounts[playerIdx]++
	}

	// Update table view with animated card counts (cap at 5)
	for i := 0; i < 4; i++ {
		if cardCounts[i] > 5 {
			cardCounts[i] = 5
		}
		g.tableView.PlayerHands[i] = cardCounts[i]
		g.tableView.TricksWon[i] = 0
	}

	g.tableView.Dealer = dealer
	g.tableView.CurrentPlayer = -1 // No one's turn during dealing
	g.tableView.CurrentTrick = nil
	g.tableView.Trump = engine.NoSuit
	g.tableView.TurnPulseFrame = g.turnPulseFrame

	// Show turned card after all cards are dealt (step 20)
	if g.dealStep >= 20 {
		g.tableView.TurnedCard = g.game.TurnedCard()
	} else {
		g.tableView.TurnedCard = engine.Card{}
		g.message = "Dealing..."
	}
}

// View implements tea.Model
func (g *GamePlay) View() string {
	width := g.width
	height := g.height
	if width == 0 {
		width = 80
	}
	if height == 0 {
		height = 30
	}

	// Show shuffle animation
	if g.isShuffling {
		return g.renderShuffleAnimation(width, height)
	}

	// Score & Tricks cards
	scores := g.game.Scores()

	// Get tricks for each team
	var yourTricks, oppTricks int
	round := g.game.Round()
	if round != nil {
		yourTricks = round.TricksWon(0) + round.TricksWon(2) // You + Partner
		oppTricks = round.TricksWon(1) + round.TricksWon(3)  // West + East
	}

	// Helper to render trick dots
	renderTrickDots := func(tricks int) string {
		filled := "●"
		empty := "○"
		result := ""
		for i := 0; i < 5; i++ {
			if i < tricks {
				result += filled
			} else {
				result += empty
			}
		}
		return result
	}

	cardWidth := 15

	// Build score values with optional animation
	yourScoreStr := fmt.Sprintf("%d pts", scores[0])
	oppScoreStr := fmt.Sprintf("%d pts", scores[1])

	if g.scoreAnimFrames > 0 {
		if g.scoreDelta[0] > 0 {
			yourScoreStr = fmt.Sprintf("%d pts (+%d)", scores[0], g.scoreDelta[0])
		}
		if g.scoreDelta[1] > 0 {
			oppScoreStr = fmt.Sprintf("%d pts (+%d)", scores[1], g.scoreDelta[1])
		}
	}

	// Build cards manually with box drawing
	borderColor := lipgloss.Color("#3498DB")
	bc := lipgloss.NewStyle().Foreground(borderColor)

	// Card building helper
	buildCard := func(header, headerColor, score, scoreColor string, tricks int, highlight bool) string {
		hc := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color(headerColor)).Width(cardWidth).Align(lipgloss.Center)
		sc := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color(scoreColor)).Width(cardWidth).Align(lipgloss.Center)
		if highlight {
			sc = sc.Background(lipgloss.Color(scoreColor)).Foreground(lipgloss.Color("#FFFFFF"))
		}
		dc := lipgloss.NewStyle().Width(cardWidth).Align(lipgloss.Center)

		dots := renderTrickDots(tricks)

		lines := []string{
			bc.Render("┌" + strings.Repeat("─", cardWidth) + "┐"),
			bc.Render("│") + hc.Render(header) + bc.Render("│"),
			bc.Render("├" + strings.Repeat("─", cardWidth) + "┤"),
			bc.Render("│") + sc.Render(score) + bc.Render("│"),
			bc.Render("│") + dc.Render(dots) + bc.Render("│"),
			bc.Render("└" + strings.Repeat("─", cardWidth) + "┘"),
		}
		return strings.Join(lines, "\n")
	}

	yourHighlight := g.scoreAnimFrames > 0 && g.scoreDelta[0] > 0
	oppHighlight := g.scoreAnimFrames > 0 && g.scoreDelta[1] > 0

	yourCard := buildCard("YOUR TEAM", "#2ECC71", yourScoreStr, "#2ECC71", yourTricks, yourHighlight)
	oppCard := buildCard("OPPONENTS", "#E74C3C", oppScoreStr, "#E74C3C", oppTricks, oppHighlight)

	scoreStr := lipgloss.JoinHorizontal(lipgloss.Top, yourCard, "  ", oppCard)

	// Table view
	tableStr := g.tableView.Render()

	// Dealer badge style
	dealerStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#000")).
		Background(lipgloss.Color("#f1c40f")).
		Bold(true).
		Padding(0, 1)

	// Player's hand with tricks counter
	var handStr string
	if g.isDealing {
		// During dealing, show face-down cards based on animation step
		cardCount := g.tableView.PlayerHands[g.humanPlayer]
		header := theme.Current.Primary.Render("You")
		if g.game.Dealer() == g.humanPlayer {
			header += " " + dealerStyle.Render("DEALER")
		}
		faceDown := components.RenderFaceDown(cardCount)
		handStr = lipgloss.JoinVertical(lipgloss.Center, header, faceDown)
	} else {
		hand := g.game.Hand(g.humanPlayer)
		playerTricks := 0
		round := g.game.Round()
		if round != nil {
			playerTricks = round.TricksWon(g.humanPlayer)
		}

		// Build player header with name and dealer badge
		playerName := theme.Current.Primary.Render("You")
		if g.game.Dealer() == g.humanPlayer {
			playerName += " " + dealerStyle.Render("DEALER")
		}

		// Show tricks table (or suit selector during round 2 bidding)
		tricksTable := components.RenderTricksTable(playerTricks)

		// Build header - center everything for consistent layout
		var playerHeader string
		phase := g.game.Phase()
		if phase == engine.PhaseBidRound2 && g.game.CurrentPlayer() == g.humanPlayer && g.suitSelector != nil {
			// Show suit selector in place of tricks during round 2 bidding
			suitSelectorWidget := g.suitSelector.Render()
			playerHeader = lipgloss.JoinVertical(lipgloss.Center, playerName, suitSelectorWidget)
		} else if phase == engine.PhaseDiscard && len(hand) == 6 {
			discardMsg := theme.Current.Muted.Render("(select one to discard)")
			playerHeader = lipgloss.JoinVertical(lipgloss.Center, playerName, tricksTable, discardMsg)
		} else {
			playerHeader = lipgloss.JoinVertical(lipgloss.Center, playerName, tricksTable)
		}

		legalPlays := make([]engine.Card, 0)
		if phase == engine.PhasePlay && g.game.CurrentPlayer() == g.humanPlayer {
			if round != nil && round.Trick() != nil {
				legalPlays = engine.LegalPlays(engine.NewHandWith(hand), round.Trick())
			}
		}

		// Debug: Check for 6-card bug during play phase
		if phase == engine.PhasePlay && len(hand) > 5 {
			// Show detailed debug info
			cardNames := make([]string, len(hand))
			for i, c := range hand {
				cardNames[i] = c.String()
			}
			bugMsg := theme.Current.CardRed.Render(fmt.Sprintf("BUG: You have %d cards: %v", len(hand), cardNames))
			playerHeader = lipgloss.JoinVertical(lipgloss.Center, bugMsg, tricksTable)
		}

		// Only show selection when it's your turn to select a card
		// Must be in discard/play phase, your turn, and not waiting for acknowledgment
		selectedIdx := -1
		isYourTurn := g.game.CurrentPlayer() == g.humanPlayer
		canSelect := (phase == engine.PhaseDiscard || phase == engine.PhasePlay) &&
			isYourTurn && !g.waitingForTrickAck && !g.waitingForRoundAck
		if canSelect {
			selectedIdx = g.selectedCard
		}

		handCards := components.RenderHand(hand, selectedIdx, legalPlays)
		handStr = lipgloss.JoinVertical(lipgloss.Center, playerHeader, handCards)
	}

	// Fixed height for hand area (1 name + 3 tricks table + 1 blank + 5 cards + 1 raised = 11)
	handStr = lipgloss.NewStyle().Height(11).Render(handStr)

	// Phase indicator and message
	phaseStr := g.getPhaseMessage()
	if g.message != "" {
		// If it's the human's turn during bidding, combine the AI's message with the prompt
		phase := g.game.Phase()
		isYourTurn := g.game.CurrentPlayer() == g.humanPlayer
		if isYourTurn && (phase == engine.PhaseBidRound1 || phase == engine.PhaseBidRound2) {
			phaseStr = g.message + " — " + phaseStr
		} else {
			phaseStr = g.message
		}
	}

	// Help text
	helpStr := g.getHelpText()

	innerContent := scoreStr + "\n\n" +
		tableStr + "\n" +
		handStr + "\n\n" +
		theme.Current.Accent.Render(phaseStr) + "\n\n" +
		theme.Current.Help.Render(helpStr)

	// Add celebration overlay if active
	if g.celebrationFrames > 0 {
		innerContent = g.addCelebrationEffect(innerContent)
	}

	// Center content and wrap in screen border
	centeredContent := lipgloss.Place(width-4, height-4, lipgloss.Center, lipgloss.Center, innerContent)

	// Gold border during celebration
	var screenBox string
	if g.celebrationFrames > 0 {
		celebrationBorder := lipgloss.NewStyle().
			Border(lipgloss.DoubleBorder()).
			BorderForeground(lipgloss.Color("#FFD700")).
			Width(width - 2).
			Height(height - 2)
		screenBox = celebrationBorder.Render(centeredContent)
	} else {
		screenBox = theme.Current.ScreenBorder.
			Width(width - 2).
			Height(height - 2).
			Render(centeredContent)
	}

	return lipgloss.Place(width, height, lipgloss.Center, lipgloss.Center, screenBox)
}

// renderShuffleAnimation renders the deck shuffling animation
func (g *GamePlay) renderShuffleAnimation(width, height int) string {
	// Simple cut & merge with tall back cards
	frames := []string{
		// Start - single deck with depth
		"┌─────┐┐\n│░░░░░││\n│░░░░░││\n│░░░░░││\n└─────┘┘",
		// Thicken
		"┌─────┐┐┐\n│░░░░░│││\n│░░░░░│││\n│░░░░░│││\n└─────┘┘┘",
		// Full thickness
		"┌─────┐┐┐┐\n│░░░░░││││\n│░░░░░││││\n│░░░░░││││\n└─────┘┘┘┘",
		// Cut - split
		"┌─────┐┐    ┌─────┐┐\n│░░░░░││    │░░░░░││\n│░░░░░││    │░░░░░││\n│░░░░░││    │░░░░░││\n└─────┘┘    └─────┘┘",
		// Wide split
		"┌─────┐┐      ┌─────┐┐\n│░░░░░││      │░░░░░││\n│░░░░░││      │░░░░░││\n│░░░░░││      │░░░░░││\n└─────┘┘      └─────┘┘",
		// Coming together
		"┌─────┐┐ ┌─────┐┐\n│░░░░░││ │░░░░░││\n│░░░░░││ │░░░░░││\n│░░░░░││ │░░░░░││\n└─────┘┘ └─────┘┘",
		// Merged
		"┌─────┐┐┐┐┐\n│░░░░░│││││\n│░░░░░│││││\n│░░░░░│││││\n└─────┘┘┘┘┘",
		// Settling
		"┌─────┐┐┐┐\n│░░░░░││││\n│░░░░░││││\n│░░░░░││││\n└─────┘┘┘┘",
		// More settling
		"┌─────┐┐┐\n│░░░░░│││\n│░░░░░│││\n│░░░░░│││\n└─────┘┘┘",
		// Almost done
		"┌─────┐┐\n│░░░░░││\n│░░░░░││\n│░░░░░││\n└─────┘┘",
		// Final
		"┌─────┐\n│░░░░░│\n│░░░░░│\n│░░░░░│\n└─────┘",
		// Done - highlight
		"┌─────┐\n│▓▓▓▓▓│\n│▓▓▓▓▓│\n│▓▓▓▓▓│\n└─────┘",
	}

	frame := g.shuffleStep % len(frames)
	deckArt := frames[frame]

	// Animate the title
	titles := []string{
		"Shuffling.",
		"Shuffling..",
		"Shuffling...",
	}
	titleFrame := (g.shuffleStep / 4) % len(titles)

	title := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#3498DB")).
		Bold(true).
		Render(titles[titleFrame])

	content := title + "\n\n" + theme.Current.Muted.Render(deckArt)

	centeredContent := lipgloss.Place(width-4, height-4, lipgloss.Center, lipgloss.Center, content)
	screenBox := theme.Current.ScreenBorder.
		Width(width - 2).
		Height(height - 2).
		Render(centeredContent)

	return lipgloss.Place(width, height, lipgloss.Center, lipgloss.Center, screenBox)
}

// addCelebrationEffect adds confetti-like decorations to the content
func (g *GamePlay) addCelebrationEffect(content string) string {
	// Add celebratory symbols around the content
	confetti := []string{"🎉", "✨", "🎊", "⭐", "🏆"}

	// Simple approach: add a celebration header
	frame := g.celebrationFrames % 5
	symbol := confetti[frame]

	celebration := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#FFD700")).
		Bold(true).
		Render(symbol + " WINNER! " + symbol)

	return celebration + "\n" + content
}

// getPhaseMessage returns a message describing the current phase
func (g *GamePlay) getPhaseMessage() string {
	phase := g.game.Phase()
	current := g.game.CurrentPlayer()
	isYourTurn := current == g.humanPlayer

	switch phase {
	case engine.PhaseBidRound1:
		if isYourTurn {
			return fmt.Sprintf("Your turn: Order up %s or pass?", g.game.TurnedCard())
		}
		return fmt.Sprintf("Waiting for %s to bid...", g.tableView.PlayerNames[current])

	case engine.PhaseBidRound2:
		if isYourTurn {
			return "Round 2: Select a suit or pass"
		}
		return fmt.Sprintf("Waiting for %s to bid...", g.tableView.PlayerNames[current])

	case engine.PhaseDiscard:
		if isYourTurn {
			return "You picked up the trump card. Select a card to discard."
		}
		return "Dealer is picking up trump and discarding..."

	case engine.PhasePlay:
		if isYourTurn {
			return "Your turn: Select a card to play"
		}
		return fmt.Sprintf("Waiting for %s to play...", g.tableView.PlayerNames[current])

	case engine.PhaseGameEnd:
		winner := g.game.Winner()
		if winner == 0 {
			return "Congratulations! Your team wins!"
		}
		return "Game over. Opponents win."

	default:
		return ""
	}
}

// getHelpText returns context-appropriate help text
func (g *GamePlay) getHelpText() string {
	// Check waiting states first
	if g.waitingForRoundAck {
		if g.game.IsOver() {
			return "Enter: Return to menu"
		}
		return "Enter: Next round • Esc: Quit"
	}
	if g.waitingForTrickAck {
		return "Enter: Continue"
	}

	phase := g.game.Phase()

	switch phase {
	case engine.PhaseBidRound1:
		return "Enter: Order up • P: Pass • A: Order up alone • Esc: Quit"
	case engine.PhaseBidRound2:
		return "←/→: Select suit • Enter: Call • P: Pass • Esc: Quit"
	case engine.PhaseDiscard:
		return "←/→: Select card • Enter: Discard • Esc: Quit"
	case engine.PhasePlay:
		return "←/→: Select card • Enter: Play • Esc: Quit"
	default:
		return "Esc: Return to menu"
	}
}

// Messages for async operations
type aiTurnMsg struct{}
type aiContinueMsg struct{}
type aiBidMsg struct {
	message string
}
type roundCompleteMsg struct{}
type dealCardMsg struct{} // Animation tick for dealing
type trickDoneMsg struct {
	result engine.TrickResult
}
type aiErrorMsg struct {
	err    error
	player int
	action string
}
type aiCardPlayMsg struct {
	card   engine.Card
	player int
}
type aiCardPlayWithTrickMsg struct {
	card   engine.Card
	player int
	result engine.TrickResult
}
type humanTurnMsg struct{}
type tempMessageMsg struct {
	originalMsg string
}

// Animation messages
type shuffleTickMsg struct{}
type trumpFlashTickMsg struct{}
type cardFlipTickMsg struct{}
type scoreAnimTickMsg struct{}
type cardPlayTickMsg struct{}
type trickCollectTickMsg struct{}
type turnPulseTickMsg struct{}
type celebrationTickMsg struct{}
