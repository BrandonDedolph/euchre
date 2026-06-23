package app

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/bran/euchre/internal/ai"
	"github.com/bran/euchre/internal/ai/rule_based"
	"github.com/bran/euchre/internal/engine"
	"github.com/bran/euchre/internal/ui/components"
	"github.com/bran/euchre/internal/ui/theme"
	"github.com/bran/euchre/internal/variants"
	"github.com/bran/euchre/internal/variants/standard"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

const aiTurnDelay = 500 * time.Millisecond
const aiBidDelay = 1200 * time.Millisecond   // Slower for bidding so user can follow
const dealCardDelay = 170 * time.Millisecond // per dealt packet (2 or 3 cards)

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
	eventLog           []string // recent player-facing events (most recent last)
	tableView          *components.TableView
	width              int
	height             int
	waitingForTrickAck bool                // Waiting for user to acknowledge trick result
	completedTrick     *engine.TrickResult // The completed trick to display
	isDealing          bool                // Currently animating the deal
	dealStep           int                 // Current deal packet dealt (0-8: Euchre 2s & 3s)
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

// rulesFromVariant maps a selected variant's options to the engine's Rules
// struct. The variant is the single source of truth for rule invariants (e.g.
// AllowMisdeal == !StickTheDealer), so all callers route through here rather
// than hand-building Rules.
func rulesFromVariant(v variants.Variant) engine.Rules {
	return engine.Rules{
		StickTheDealer:   v.HasStickTheDealer(),
		AllowMisdeal:     v.AllowMisdeal(),
		AllowDefendAlone: v.GetBoolOption("defend_alone", false),
	}
}

// variantFromSettings builds a fresh, configured variant from the setup
// screen's toggles. Only the standard variant exists today, so we construct it
// directly; when more variants are added this should look up s.Variant in the
// registry (constructing a fresh instance to avoid mutating shared singletons).
func variantFromSettings(s GameSettings) variants.Variant {
	v := standard.New()
	_ = v.SetOption("stick_the_dealer", s.StickTheDealer)
	_ = v.SetOption("defend_alone", s.DefendAlone)
	return v
}

// NewGamePlay creates a new game play screen with default rules:
// the standard variant with all optional rules off. This preserves the
// behavior of the original constructor for callers that have no settings.
func NewGamePlay() *GamePlay {
	// Map the standard variant's default options onto the engine's plain Rules
	// struct. The engine cannot import variants (that would be a circular
	// import), so the app layer does this translation.
	return newGamePlay(rulesFromVariant(standard.New()))
}

// NewGamePlayWithSettings creates a new game play screen using the rule toggles
// chosen on the setup screen.
func NewGamePlayWithSettings(s GameSettings) *GamePlay {
	return newGamePlay(rulesFromVariant(variantFromSettings(s)))
}

// newGamePlay is the shared constructor body. It builds the game from the given
// engine rules and wires up the human/AI players, animation state, and starts
// the first round.
func newGamePlay(rules engine.Rules) *GamePlay {
	config := engine.DefaultGameConfig()
	config.Rules = rules

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
		g.pushEvent(msg.message)
		g.updateTableView()
		return g, tea.Tick(aiBidDelay, func(t time.Time) tea.Msg {
			return aiContinueMsg{}
		})

	case aiContinueMsg:
		// Continue processing AI turns after delay
		return g, g.processAITurns()

	case humanTurnMsg:
		// It's the human's turn - update the display and pre-select a legal card
		// so pressing Enter always plays a valid card by default.
		g.selectedCard = g.firstLegalCardIndex()
		g.updateTableView()
		return g, nil

	case tempMessageMsg:
		// Restore the original message after showing a temporary message
		g.message = msg.originalMsg
		return g, nil

	case dealCardMsg:
		// Animate dealing one packet (Euchre deals in 2s and 3s, not one at a time)
		if !g.isDealing {
			// Already done dealing, ignore stale message
			return g, nil
		}
		g.dealStep++
		g.updateDealingView()
		if g.dealStep >= len(dealPacketPlan(g.game.Dealer())) { // all packets dealt
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
		g.pushEvent(trickMsg)
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

		// A misdeal (round-2 throw-in) appends nothing to history and changes no
		// scores, so the history-based result below would show a stale/zero
		// result. Handle it explicitly with a clear re-deal message.
		if g.game.IsMisdeal() {
			g.message = "Throw-in — everyone passed. Re-dealing…"
			return g, nil
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
			g.eventLog = nil // fresh log for the new deal
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

	// Defend-alone declaration window: only the polled human defender acts here.
	if phase == engine.PhaseDefendAlone && g.game.CurrentPlayer() == g.humanPlayer {
		switch msg.String() {
		case "q", "esc":
			return g, Navigate(ScreenMainMenu)
		case "y":
			return g.handleDefendAlone(true)
		case "n", "p", "enter", " ":
			return g.handleDefendAlone(false)
		}
		return g, nil
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
			g.pushEvent(g.message)
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
			g.pushEvent(g.message)
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
				// Illegal play: snap selection to a legal card and point the
				// player at the highlighted options. Tailor the message to the
				// failure so a non-follow-suit error isn't mislabeled.
				g.selectedCard = g.firstLegalCardIndex()
				msg := "Can't play that card — playable cards are highlighted in green."
				if errors.Is(err, engine.ErrMustFollowSuit) {
					msg = "Must follow suit — playable cards are highlighted in green."
				}
				return g.showTempMessage(msg)
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
		g.pushEvent(g.message)
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
			g.pushEvent("You're going alone!")
			g.updateTableView()
		}
	}

	return g, g.processAITurns()
}

// handleDefendAlone handles the human declaring (or declining) a lone defense
// during the PhaseDefendAlone declaration window.
func (g *GamePlay) handleDefendAlone(declare bool) (tea.Model, tea.Cmd) {
	if g.game.Phase() != engine.PhaseDefendAlone {
		return g, nil
	}
	if g.game.CurrentPlayer() != g.humanPlayer {
		return g.showTempMessage("Not your turn")
	}

	var action engine.Action
	if declare {
		action = engine.DefendAloneAction{PlayerIdx: g.humanPlayer}
	} else {
		action = engine.PassAction{PlayerIdx: g.humanPlayer}
	}

	if err := g.game.ApplyAction(action); err != nil {
		g.message = err.Error()
	} else {
		if declare {
			g.message = "You defend alone!"
		} else {
			g.message = "You decline to defend alone"
		}
		g.pushEvent(g.message)
		g.updateTableView()
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
	g.pushEvent(g.message)
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

		case engine.PhaseDefendAlone:
			playerName := g.tableView.PlayerNames[current]
			if aiPlayer.DecideDefendAlone(state) {
				action := engine.DefendAloneAction{PlayerIdx: current}
				if err := g.game.ApplyAction(action); err != nil {
					return aiErrorMsg{err: err, player: current, action: "defend-alone"}
				}
				g.message = fmt.Sprintf("%s defends alone!", playerName)
				g.pushEvent(g.message)
			} else {
				action := engine.PassAction{PlayerIdx: current}
				if err := g.game.ApplyAction(action); err != nil {
					return aiErrorMsg{err: err, player: current, action: "defend-alone-pass"}
				}
				g.message = fmt.Sprintf("%s declines to defend alone", playerName)
				g.pushEvent(g.message)
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

// firstLegalCardIndex returns the hand index of the first legal card the human
// can play in the current trick. It returns 0 when not in the human's play turn
// or when legality can't be determined (the caller's default selection).
func (g *GamePlay) firstLegalCardIndex() int {
	if g.game.Phase() != engine.PhasePlay || g.game.CurrentPlayer() != g.humanPlayer {
		return 0
	}
	round := g.game.Round()
	if round == nil || round.Trick() == nil {
		return 0
	}
	hand := g.game.Hand(g.humanPlayer)
	legal := engine.LegalPlays(engine.NewHandWith(hand), round.Trick())
	if len(legal) == 0 {
		return 0
	}
	for i, c := range hand {
		for _, lc := range legal {
			if c.Suit == lc.Suit && c.Rank == lc.Rank {
				return i
			}
		}
	}
	return 0
}

// pushEvent appends a player-facing event to the recent-events log shown in the
// side panel, keeping only the most recent few. Empty/duplicate-of-last events
// are ignored to avoid noise.
func (g *GamePlay) pushEvent(msg string) {
	const maxEvents = 4
	if msg == "" {
		return
	}
	if n := len(g.eventLog); n > 0 && g.eventLog[n-1] == msg {
		return
	}
	g.eventLog = append(g.eventLog, msg)
	if len(g.eventLog) > maxEvents {
		g.eventLog = g.eventLog[len(g.eventLog)-maxEvents:]
	}
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

// dealPacketPlan returns the Euchre deal as an ordered list of {player, count}
// packets, mirroring the engine's dealPasses for 4 players: packets of 2,3,2,3
// on the first pass then 3,2,3,2 on the second, starting to the dealer's left.
// Each player ends with 5 cards.
func dealPacketPlan(dealer int) [][2]int {
	passes := [2][4]int{{2, 3, 2, 3}, {3, 2, 3, 2}}
	plan := make([][2]int, 0, 8)
	for _, pass := range passes {
		for i := 0; i < 4; i++ {
			player := (dealer + 1 + i) % 4
			plan = append(plan, [2]int{player, pass[i]})
		}
	}
	return plan
}

// updateDealingView updates the table view during the deal animation, revealing
// one packet (2 or 3 cards) per step rather than a single card at a time.
func (g *GamePlay) updateDealingView() {
	dealer := g.game.Dealer()
	plan := dealPacketPlan(dealer)

	step := g.dealStep
	if step > len(plan) {
		step = len(plan)
	}

	cardCounts := [4]int{0, 0, 0, 0}
	for i := 0; i < step; i++ {
		player, count := plan[i][0], plan[i][1]
		cardCounts[player] += count
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

	// Show the turned card only once every packet has been dealt.
	if g.dealStep >= len(plan) {
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

	// Below this the fixed-size table can't render at all — ask for a resize
	// rather than spilling a broken layout.
	if width < minPlayableWidth || height < minPlayableHeight {
		return g.renderTooSmall(width, height)
	}

	// The side HUD panels only fit on wider terminals; below that we fall back
	// to a compact layout (table + a single scoreboard line, no panels).
	showPanels := width >= fullLayoutMinWidth

	// Show shuffle animation
	if g.isShuffling {
		return g.renderShuffleAnimation(width, height)
	}

	// Table view
	tableStr := g.tableView.Render()

	// Dealer badge style
	dealerStyle := theme.Current.DealerBadge

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

		// Build player header with name, inline tricks, and dealer badge
		tricksStyle := theme.Current.Muted
		tricksStr := tricksStyle.Render(fmt.Sprintf("(%d)", playerTricks))
		playerName := theme.Current.Primary.Render("You") + " " + tricksStr
		if g.game.Dealer() == g.humanPlayer {
			playerName += " " + dealerStyle.Render("DEALER")
		}

		// Build header - center everything for consistent layout
		var playerHeader string
		phase := g.game.Phase()
		if phase == engine.PhaseBidRound2 && g.game.CurrentPlayer() == g.humanPlayer && g.suitSelector != nil {
			// Show suit selector below name during round 2 bidding
			suitSelectorWidget := g.suitSelector.Render()
			playerHeader = lipgloss.JoinVertical(lipgloss.Center, playerName, suitSelectorWidget)
		} else if phase == engine.PhaseDiscard && len(hand) == 6 {
			discardMsg := theme.Current.Muted.Render("(select one to discard)")
			playerHeader = lipgloss.JoinVertical(lipgloss.Center, playerName, discardMsg)
		} else {
			playerHeader = playerName
		}

		legalPlays := make([]engine.Card, 0)
		if phase == engine.PhasePlay && g.game.CurrentPlayer() == g.humanPlayer {
			if round != nil && round.Trick() != nil {
				legalPlays = engine.LegalPlays(engine.NewHandWith(hand), round.Trick())
			}
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

		handCards := components.RenderHand(hand, selectedIdx, legalPlays, g.tableView.Trump)
		handStr = lipgloss.JoinVertical(lipgloss.Center, playerHeader, handCards)
	}

	// Fixed height for hand area (1 name + 1 playable marker + 5 cards + 1 raised = 8)
	handStr = lipgloss.NewStyle().Height(8).Render(handStr)

	// Build status bar with phase message (trump info now in side panel)
	phaseStr := g.getPhaseMessage()
	if g.message != "" {
		// If it's the human's turn during bidding, combine the AI's message with the prompt
		phase := g.game.Phase()
		isYourTurn := g.game.CurrentPlayer() == g.humanPlayer
		if isYourTurn && (phase == engine.PhaseBidRound1 || phase == engine.PhaseBidRound2 || phase == engine.PhaseDefendAlone) {
			phaseStr = g.message + " — " + phaseStr
		} else {
			phaseStr = g.message
		}
	}

	// Help text
	helpStr := g.getHelpText()

	// Build center content (table + hand)
	// Center the hand to match table width
	tableWidth := lipgloss.Width(tableStr)
	centeredHand := lipgloss.PlaceHorizontal(tableWidth, lipgloss.Center, handStr)
	centerContent := tableStr + centeredHand

	// Compose the main area. The side cards are pure team scoreboards; global
	// game state (round, trump, contract) goes in a banner above the table and
	// the running play log goes in a ticker below it, so neither reads as
	// belonging to one team. Compact layout folds the global state into a single
	// scoreboard line instead of the banner/cards.
	var mainArea string
	if showPanels {
		centerHeight := lipgloss.Height(centerContent)
		leftPanel := g.renderLeftPanel(centerHeight)
		rightPanel := g.renderRightPanel(centerHeight)
		mainArea = lipgloss.JoinHorizontal(lipgloss.Top, leftPanel, centerContent, rightPanel)
	} else {
		scoreBar := lipgloss.PlaceHorizontal(tableWidth, lipgloss.Center, g.renderScoreBar())
		mainArea = lipgloss.JoinVertical(lipgloss.Center, scoreBar, centerContent)
	}
	mainWidth := lipgloss.Width(mainArea)

	// Assemble vertical sections: [banner] · main area · [recent ticker].
	var sections []string
	if showPanels {
		if banner := g.renderContractBanner(mainWidth); banner != "" {
			sections = append(sections, lipgloss.PlaceHorizontal(mainWidth, lipgloss.Center, banner), "")
		}
	}
	sections = append(sections, mainArea)
	if ticker := g.renderRecentTicker(mainWidth); ticker != "" {
		sections = append(sections, lipgloss.PlaceHorizontal(mainWidth, lipgloss.Center, ticker))
	}
	block := lipgloss.JoinVertical(lipgloss.Center, sections...)

	// Build final layout (phase + help kept as their own trailing lines).
	innerContent := block + "\n" +
		theme.Current.Accent.Render(phaseStr) + "\n" +
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
	borderStyle := theme.Current.Muted
	patternStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#2563EB"))

	border := borderStyle.Render
	pattern := patternStyle.Render

	// Helper to build a deck frame with colored pattern
	// p = pattern interior, d = depth chars (border colored)
	buildDeck := func(p string, depth int) string {
		depthStr := border(repeatChar('┐', depth))
		depthPipe := border(repeatChar('│', depth))
		depthBot := border(repeatChar('┘', depth))

		return border("┌─────┐") + depthStr + "\n" +
			border("│") + pattern(p) + border("│") + depthPipe + "\n" +
			border("│") + pattern(p) + border("│") + depthPipe + "\n" +
			border("│") + pattern(p) + border("│") + depthPipe + "\n" +
			border("└─────┘") + depthBot
	}

	buildSplitDecks := func(p string, depth int, gap string) string {
		depthStr := border(repeatChar('┐', depth))
		depthPipe := border(repeatChar('│', depth))
		depthBot := border(repeatChar('┘', depth))

		deck := border("┌─────┐") + depthStr
		row := border("│") + pattern(p) + border("│") + depthPipe
		bot := border("└─────┘") + depthBot

		return deck + gap + deck + "\n" +
			row + gap + row + "\n" +
			row + gap + row + "\n" +
			row + gap + row + "\n" +
			bot + gap + bot
	}

	// Animation frames
	var deckArt string
	frameNum := g.shuffleStep % 12

	switch frameNum {
	case 0: // Start - single deck with depth
		deckArt = buildDeck("░░░░░", 1)
	case 1: // Thicken
		deckArt = buildDeck("░░░░░", 2)
	case 2: // Full thickness
		deckArt = buildDeck("░░░░░", 3)
	case 3: // Cut - split
		deckArt = buildSplitDecks("░░░░░", 1, "    ")
	case 4: // Wide split
		deckArt = buildSplitDecks("░░░░░", 1, "      ")
	case 5: // Coming together
		deckArt = buildSplitDecks("░░░░░", 1, " ")
	case 6: // Merged
		deckArt = buildDeck("░░░░░", 4)
	case 7: // Settling
		deckArt = buildDeck("░░░░░", 3)
	case 8: // More settling
		deckArt = buildDeck("░░░░░", 2)
	case 9: // Almost done
		deckArt = buildDeck("░░░░░", 1)
	case 10: // Final
		deckArt = buildDeck("░░░░░", 0)
	case 11: // Done - highlight (brighter pattern)
		highlightPattern := lipgloss.NewStyle().Foreground(lipgloss.Color("#60A5FA")).Bold(true)
		hp := highlightPattern.Render
		deckArt = border("┌─────┐") + "\n" +
			border("│") + hp("░░░░░") + border("│") + "\n" +
			border("│") + hp("░░░░░") + border("│") + "\n" +
			border("│") + hp("░░░░░") + border("│") + "\n" +
			border("└─────┘")
	}

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

	content := title + "\n\n" + deckArt

	centeredContent := lipgloss.Place(width-4, height-4, lipgloss.Center, lipgloss.Center, content)
	screenBox := theme.Current.ScreenBorder.
		Width(width - 2).
		Height(height - 2).
		Render(centeredContent)

	return lipgloss.Place(width, height, lipgloss.Center, lipgloss.Center, screenBox)
}

// repeatChar repeats a rune n times
func repeatChar(r rune, n int) string {
	if n <= 0 {
		return ""
	}
	result := make([]rune, n)
	for i := range result {
		result[i] = r
	}
	return string(result)
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

	case engine.PhaseDefendAlone:
		if isYourTurn {
			return "Opponent goes alone! Defend alone? (y = yes, n = no)"
		}
		return fmt.Sprintf("Waiting for %s to decide on a lone defense...", g.tableView.PlayerNames[current])

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
	case engine.PhaseDefendAlone:
		if g.game.CurrentPlayer() == g.humanPlayer {
			return "Y: Defend alone • N: Decline • Esc: Quit"
		}
		return "Esc: Quit"
	case engine.PhasePlay:
		return "←/→: Select card • Enter: Play • Esc: Quit"
	default:
		return "Esc: Return to menu"
	}
}

// renderLeftPanel renders your team's scoreboard card (score + tricks). Global
// state (round, trump, contract, log) lives in the banner/ticker, not here, so
// the card reads purely as your team's standing.
func (g *GamePlay) renderLeftPanel(height int) string {
	scores := g.game.Scores()
	var yourTricks int
	if r := g.game.Round(); r != nil {
		yourTricks = r.TricksWon(0) + r.TricksWon(2)
	}

	scoreStyle := lipgloss.NewStyle().Foreground(theme.ColGreen).Bold(true)
	scoreStr := fmt.Sprintf("%d pts", scores[0])
	if g.scoreAnimFrames > 0 && g.scoreDelta[0] > 0 {
		scoreStr = fmt.Sprintf("%d (+%d) pts", scores[0], g.scoreDelta[0])
		scoreStyle = scoreStyle.Background(theme.ColGreen).Foreground(lipgloss.Color("#FFF"))
	}

	body := []string{
		panelCenter(scoreStyle, scoreStr),
		"",
		panelCenter(theme.Current.Muted, "Tricks"),
		panelTrickDots(yourTricks, theme.ColGreen),
	}

	return panelBox("YOU", theme.ColGreen, body, height)
}

// Layout sizing. panelInnerWidth is the content width inside each side-panel box
// (outer = 13 incl. the rounded border). The min/full thresholds decide between
// the too-small guard, the compact (panel-less) layout, and the full HUD layout.
const (
	panelInnerWidth    = 11 // content width inside a panel box (outer = 13 incl. border)
	minPlayableWidth   = 64 // table (~61 wide) + screen-border margin
	minPlayableHeight  = 20 // enough rows for the table core
	fullLayoutMinWidth = 89 // table (61) + both panel boxes (13 each) + screen border
)

// panelBox wraps a panel's body in a rounded border with a colored title bar,
// then centers it vertically in the given column height so it reads as a
// self-contained card rather than a full-height divider.
func panelBox(title string, accent lipgloss.TerminalColor, body []string, height int) string {
	header := lipgloss.NewStyle().
		Width(panelInnerWidth).
		Align(lipgloss.Center).
		Bold(true).
		Background(accent).
		Foreground(lipgloss.Color("#FFFFFF")).
		Render(title)

	inner := lipgloss.JoinVertical(lipgloss.Center, append([]string{header}, body...)...)
	box := lipgloss.NewStyle().
		Width(panelInnerWidth).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(theme.ColBlue).
		Render(inner)

	// Top-align both panels so their title bars line up; pad below to fill the
	// column height beside the table.
	return lipgloss.PlaceVertical(height, lipgloss.Top, box)
}

// panelCenter renders s centered to the panel inner width using the given style.
func panelCenter(style lipgloss.Style, s string) string {
	return style.Width(panelInnerWidth).Align(lipgloss.Center).Render(s)
}

// panelTrickDots renders 5 trick pips (filled in the team accent, empty muted),
// centered to the panel width.
func panelTrickDots(n int, accent lipgloss.TerminalColor) string {
	filled := lipgloss.NewStyle().Foreground(accent)
	empty := theme.Current.Muted
	s := ""
	for i := 0; i < 5; i++ {
		if i < n {
			s += filled.Render("●")
		} else {
			s += empty.Render("○")
		}
	}
	return lipgloss.PlaceHorizontal(panelInnerWidth, lipgloss.Center, s)
}

// renderRightPanel renders the opponents' scoreboard card (score + tricks),
// mirroring renderLeftPanel. Global state lives in the banner/ticker.
func (g *GamePlay) renderRightPanel(height int) string {
	scores := g.game.Scores()
	var oppTricks int
	if r := g.game.Round(); r != nil {
		oppTricks = r.TricksWon(1) + r.TricksWon(3)
	}

	scoreStyle := lipgloss.NewStyle().Foreground(theme.ColRed).Bold(true)
	scoreStr := fmt.Sprintf("%d pts", scores[1])
	if g.scoreAnimFrames > 0 && g.scoreDelta[1] > 0 {
		scoreStr = fmt.Sprintf("%d (+%d) pts", scores[1], g.scoreDelta[1])
		scoreStyle = scoreStyle.Background(theme.ColRed).Foreground(lipgloss.Color("#FFF"))
	}

	body := []string{
		panelCenter(scoreStyle, scoreStr),
		"",
		panelCenter(theme.Current.Muted, "Tricks"),
		panelTrickDots(oppTricks, theme.ColRed),
	}

	return panelBox("OPP", theme.ColRed, body, height)
}

// trumpBadge renders a small filled trump chip (suit symbol + name) colored to
// match the suit, for inline use in the contract banner.
func (g *GamePlay) trumpBadge() string {
	bg := lipgloss.Color("#000000") // spades/clubs
	if g.tableView.Trump == engine.Hearts || g.tableView.Trump == engine.Diamonds {
		bg = lipgloss.Color("#E74C3C")
	}
	return lipgloss.NewStyle().
		Bold(true).
		Background(bg).
		Foreground(lipgloss.Color("#FFFFFF")).
		Padding(0, 1).
		Render(g.tableView.Trump.Symbol() + " " + g.tableView.Trump.String())
}

// renderContractBanner builds the centered global banner above the table:
// round, trump, and who holds the contract. None of this belongs to one team,
// so it sits between the two scoreboards rather than inside either. Returns ""
// during dealing (before there's anything to show).
func (g *GamePlay) renderContractBanner(maxWidth int) string {
	if g.isDealing {
		return ""
	}
	sep := theme.Current.Muted.Render("   ·   ")
	parts := []string{theme.Current.Muted.Render(fmt.Sprintf("Round %d", g.tableView.RoundNumber))}

	if g.tableView.Trump != engine.NoSuit {
		parts = append(parts, theme.Current.Muted.Render("Trump ")+g.trumpBadge())
		if m := g.tableView.Maker; m >= 0 && m < len(g.tableView.PlayerNames) {
			tag := "called by " + g.tableView.PlayerNames[m]
			if g.tableView.MakerAlone {
				tag += " (alone)"
			}
			parts = append(parts, theme.Current.Muted.Render(tag))
		}
	} else {
		parts = append(parts, theme.Current.Muted.Italic(true).Render("bidding…"))
	}

	return lipgloss.NewStyle().MaxWidth(maxWidth).Render(strings.Join(parts, sep))
}

// renderRecentTicker builds the centered global play log shown below the table:
// the last few events as a single horizontal line. Truncated to maxWidth so it
// can never overflow. Returns "" when there are no events yet.
func (g *GamePlay) renderRecentTicker(maxWidth int) string {
	if len(g.eventLog) == 0 {
		return ""
	}
	const showN = 3
	events := g.eventLog
	if len(events) > showN {
		events = events[len(events)-showN:]
	}
	styled := make([]string, len(events))
	for i, e := range events {
		styled[i] = theme.Current.Muted.Render(e)
	}
	label := theme.Current.Muted.Bold(true).Render("Recent  ")
	line := label + strings.Join(styled, theme.Current.Muted.Render("  ·  "))
	return lipgloss.NewStyle().MaxWidth(maxWidth).Render(line)
}

// renderScoreBar builds the single-line scoreboard shown above the table in the
// compact layout, where the side HUD panels don't fit. It folds the panels'
// key facts (scores, tricks, trump, contract, round) into one centered line.
func (g *GamePlay) renderScoreBar() string {
	scores := g.game.Scores()
	var youTr, oppTr int
	if r := g.game.Round(); r != nil {
		youTr = r.TricksWon(0) + r.TricksWon(2)
		oppTr = r.TricksWon(1) + r.TricksWon(3)
	}

	parts := []string{
		theme.Current.TeamYou.Render(fmt.Sprintf("YOU %d", scores[0])),
		theme.Current.TeamOpp.Render(fmt.Sprintf("OPP %d", scores[1])),
		theme.Current.Muted.Render(fmt.Sprintf("Tricks %d-%d", youTr, oppTr)),
	}

	if g.tableView.Trump != engine.NoSuit {
		trumpStyle := theme.Current.CardBlack
		if g.tableView.Trump == engine.Hearts || g.tableView.Trump == engine.Diamonds {
			trumpStyle = theme.Current.CardRed
		}
		contract := trumpStyle.Render(g.tableView.Trump.Symbol() + " " + g.tableView.Trump.String())
		if m := g.tableView.Maker; m >= 0 && m < len(g.tableView.PlayerNames) {
			tag := g.tableView.PlayerNames[m]
			if g.tableView.MakerAlone {
				tag += ", alone"
			}
			contract += theme.Current.Muted.Render(" (" + tag + ")")
		}
		parts = append(parts, contract)
	}

	parts = append(parts, theme.Current.Muted.Render(fmt.Sprintf("Rd %d", g.tableView.RoundNumber)))

	return strings.Join(parts, theme.Current.Muted.Render("  •  "))
}

// renderTooSmall shows a friendly resize prompt when the terminal is below the
// minimum playable size, instead of rendering a broken/overflowing layout.
func (g *GamePlay) renderTooSmall(width, height int) string {
	innerW := width - 4
	innerH := height - 4
	if innerW < 1 {
		innerW = 1
	}
	if innerH < 1 {
		innerH = 1
	}

	msg := lipgloss.JoinVertical(lipgloss.Center,
		theme.Current.Title.Render("Terminal too small"),
		"",
		theme.Current.Body.Render(fmt.Sprintf("Resize to at least %d×%d", minPlayableWidth, minPlayableHeight)),
		theme.Current.Muted.Render(fmt.Sprintf("(%d wide for the full layout)", fullLayoutMinWidth)),
		"",
		theme.Current.Muted.Render(fmt.Sprintf("current: %d×%d", width, height)),
	)

	box := theme.Current.ScreenBorder.
		Width(width - 2).
		Height(height - 2).
		Render(lipgloss.Place(innerW, innerH, lipgloss.Center, lipgloss.Center, msg))

	return lipgloss.Place(width, height, lipgloss.Center, lipgloss.Center, box)
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
