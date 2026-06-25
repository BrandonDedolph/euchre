package app

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/BrandonDedolph/euchre/internal/ai"
	"github.com/BrandonDedolph/euchre/internal/ai/rule_based"
	"github.com/BrandonDedolph/euchre/internal/engine"
	"github.com/BrandonDedolph/euchre/internal/ui/components"
	"github.com/BrandonDedolph/euchre/internal/ui/theme"
	"github.com/BrandonDedolph/euchre/internal/variants"
	"github.com/BrandonDedolph/euchre/internal/variants/standard"
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
	trickCollectFrames = 6
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
	tutorial           bool            // interactive-tutorial mode: show per-move coaching
	coach              ai.Player       // strong AI used only to suggest the human's best move
	shownConcepts      map[string]bool // teachable concepts already shown this game
	pendingPopup       *concept        // teachable-moment modal currently displayed (nil = none)
	gradeMsg           string          // feedback on the human's last move vs the coach (cleared next turn)
	gradeGood          bool            // whether that move matched the coach
	selectedCard       int
	message            string
	playerAction       [4]string // latest per-seat action label (0=You,1=West,2=Partner,3=East)
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

	// showHelp toggles the full keybind sheet overlaid on the board (the "?"
	// key). It is an in-place overlay rather than a screen swap so the game in
	// progress is preserved; any key dismisses it.
	showHelp bool
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
	return newGamePlay(rulesFromVariant(standard.New()), false, ai.DifficultyMedium)
}

// NewGamePlayWithSettings creates a new game play screen using the rule toggles
// chosen on the setup screen. When s.Tutorial is set the interactive coach is
// enabled (hands are still randomly dealt — only the per-move tips are added).
func NewGamePlayWithSettings(s GameSettings) *GamePlay {
	return newGamePlay(rulesFromVariant(variantFromSettings(s)), s.Tutorial, s.Difficulty)
}

// newGamePlay is the shared constructor body. It builds the game from the given
// engine rules and wires up the human/AI players, animation state, and starts
// the first round.
func newGamePlay(rules engine.Rules, tutorial bool, difficulty ai.Difficulty) *GamePlay {
	config := engine.DefaultGameConfig()
	config.Rules = rules

	game := engine.NewGame(config)

	gp := &GamePlay{
		game:         game,
		humanPlayer:  0, // Player 0 is the human
		aiPlayers:    rule_based.CreateAIPlayers(0, difficulty),
		tutorial:     tutorial,
		selectedCard: 0,
		tableView:    components.NewTableView(),
		isShuffling:  true, // Start with shuffle animation
		shuffleStep:  0,
		isDealing:    false,
		dealStep:     0,
	}

	// In tutorial mode a strong AI sitting in the human's seat supplies the
	// suggested best move for each decision.
	if tutorial {
		gp.coach = rule_based.New("Coach", gp.humanPlayer, ai.DifficultyHard)
		gp.shownConcepts = make(map[string]bool)
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
	// A teachable-moment popup captures keyboard input until dismissed. Non-key
	// messages (animation ticks) still fall through so the board keeps ticking
	// underneath; popups are only ever queued at idle points, so nothing is lost.
	if g.pendingPopup != nil {
		if key, ok := msg.(tea.KeyMsg); ok {
			switch key.String() {
			case "enter", " ", "esc":
				g.dismissPopup()
			}
			return g, nil
		}
	}

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
		// AI made a bid, use longer delay so user can follow. The per-seat action
		// label was already set in processAITurns when the decision was applied.
		g.message = msg.message
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
		g.gradeMsg = "" // last move's feedback has run its course
		g.updateTableView()
		g.maybeShowTeachable() // idle point: safe to surface a teachable popup
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
		// Crown the winning card while the player reviews the finished trick.
		g.tableView.TrickWinner = msg.result.Winner
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
		// During play only the trick winner's "won" label shows; clear the others
		// (incl. any lingering bidding labels) so the board reads cleanly.
		g.clearActions()
		g.setAction(msg.result.Winner, "won")
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

		// Surface a teachable popup for a euchre or march now that the round
		// result is in history (idle point: we're waiting on the round ack).
		g.maybeShowTeachable()

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

	// The help sheet is a modal overlay: while it is open, any key dismisses it
	// and is otherwise swallowed so it can't also act on the board behind it.
	if g.showHelp {
		g.showHelp = false
		return g, nil
	}
	// "?" opens the full keybind sheet over the board (state preserved).
	if msg.String() == "?" {
		g.showHelp = true
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
			g.clearActions() // fresh seat labels for the new deal
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

			// The static crown gives way to the directional sweep, which draws
			// its own winner highlight from the collect anim.
			g.tableView.TrickWinner = -1

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
			g.setAction(g.humanPlayer, "orders up")
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
			g.setAction(g.humanPlayer, "calls "+suit.Symbol())
			g.suitSelector = nil // Reset for next time
			g.updateTableView()
		}

	case engine.PhaseDiscard:
		// Discard the selected card
		hand := g.game.Hand(g.humanPlayer)
		if g.selectedCard >= 0 && g.selectedCard < len(hand) {
			card := hand[g.selectedCard]
			coachCard := g.coachWould(func(s *engine.GameState) engine.Card {
				return g.coach.DecideDiscard(s, hand)
			})
			action := engine.DiscardAction{
				PlayerIdx: g.humanPlayer,
				Card:      card,
			}
			if err := g.game.ApplyAction(action); err != nil {
				g.message = err.Error()
			} else {
				g.message = fmt.Sprintf("Discarded %s", card)
				g.setAction(g.humanPlayer, "picks it up")
				g.gradeCard("discard", card, coachCard)
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

			// Leading a new trick: clear stale seat labels (any lingering bidding
			// labels on the first trick, or the prior winner's "won").
			if round != nil && len(round.CurrentTrick()) == 0 {
				g.clearActions()
			}

			coachCard := g.coachWould(func(s *engine.GameState) engine.Card {
				return g.coach.DecidePlay(s)
			})

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
			g.gradeCard("play", card, coachCard)
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
				// Sync card-back stacks so the seat that just emptied its hand
				// stops showing a played card during the trick-ack window.
				g.refreshSeatCounts()
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
		g.setAction(g.humanPlayer, "passes")
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
			g.setAction(g.humanPlayer, "orders up, alone!")
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
			g.setAction(g.humanPlayer, "defends alone!")
		} else {
			g.message = "You decline to defend alone"
			g.setAction(g.humanPlayer, "no defense")
		}
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
	g.setAction(g.humanPlayer, "calls "+suit.Symbol())
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

			// Build message about what AI decided, and set the seat's short
			// action label (shown under that seat's name).
			playerName := g.tableView.PlayerNames[current]
			var bidMsg string
			if decision.Pass {
				bidMsg = fmt.Sprintf("%s passes", playerName)
				g.setAction(current, "passes")
			} else if decision.OrderUp {
				if decision.Alone {
					bidMsg = fmt.Sprintf("%s orders it up alone!", playerName)
					g.setAction(current, "orders up, alone!")
				} else {
					bidMsg = fmt.Sprintf("%s orders it up", playerName)
					g.setAction(current, "orders up")
				}
			} else {
				if decision.Alone {
					bidMsg = fmt.Sprintf("%s calls %s alone!", playerName, decision.CallSuit)
					g.setAction(current, "calls "+decision.CallSuit.Symbol()+", alone!")
				} else {
					bidMsg = fmt.Sprintf("%s calls %s", playerName, decision.CallSuit)
					g.setAction(current, "calls "+decision.CallSuit.Symbol())
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
			// The dealer (current player here) just picked up the turn card.
			g.setAction(current, "picks it up")

		case engine.PhaseDefendAlone:
			playerName := g.tableView.PlayerNames[current]
			if aiPlayer.DecideDefendAlone(state) {
				action := engine.DefendAloneAction{PlayerIdx: current}
				if err := g.game.ApplyAction(action); err != nil {
					return aiErrorMsg{err: err, player: current, action: "defend-alone"}
				}
				g.message = fmt.Sprintf("%s defends alone!", playerName)
				g.setAction(current, "defends alone!")
			} else {
				action := engine.PassAction{PlayerIdx: current}
				if err := g.game.ApplyAction(action); err != nil {
					return aiErrorMsg{err: err, player: current, action: "defend-alone-pass"}
				}
				g.message = fmt.Sprintf("%s declines to defend alone", playerName)
				g.setAction(current, "no defense")
			}

		case engine.PhasePlay:
			// Track trick history to detect completion
			round := g.game.Round()
			historyLen := 0
			if round != nil {
				historyLen = len(round.TrickHistory())
			}

			// Leading a new trick: clear stale seat labels (any lingering bidding
			// labels on the first trick, or the prior winner's "won").
			if round != nil && len(round.CurrentTrick()) == 0 {
				g.clearActions()
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
				// Sync card-back stacks so the seat that just emptied its hand
				// stops showing a played card during the trick-ack window.
				g.refreshSeatCounts()
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

// setAction records a player's latest action so it shows on the reserved line
// beneath that seat's name. Index is the player seat (0=You,1=West,2=Partner,
// 3=East). Kept short so it fits under the seat label.
func (g *GamePlay) setAction(player int, label string) {
	if player < 0 || player >= len(g.playerAction) {
		return
	}
	g.playerAction[player] = label
}

// clearActions blanks every seat's action label. Called at the start of a new
// deal and when play begins so stale bidding labels don't linger.
func (g *GamePlay) clearActions() {
	for i := range g.playerAction {
		g.playerAction[i] = ""
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
	g.tableView.PlayerActions = g.playerAction

	// Update player hand counts
	for i := 0; i < 4; i++ {
		g.tableView.PlayerHands[i] = len(g.game.Hand(i))
		g.tableView.TricksWon[i] = round.TricksWon(i)
	}

	// Update current trick
	g.tableView.CurrentTrick = round.CurrentTrick()

	// The crown is owned by the trick-ack window (set in the trickDoneMsg
	// handler, which does not route through here). Any normal table refresh
	// means we're no longer reviewing a finished trick, so clear it.
	g.tableView.TrickWinner = -1
}

// refreshSeatCounts syncs the per-seat card-back stacks and trick counts with
// the engine without touching CurrentTrick. The trick-completion paths can't
// call updateTableView (it would replace the finished trick on the table with
// the new empty one), but the seat that played the trick-winning card still
// needs its card-back stack decremented — otherwise it keeps showing a card it
// already played, most visibly on the final trick where the seat should be
// empty.
func (g *GamePlay) refreshSeatCounts() {
	round := g.game.Round()
	if round == nil {
		return
	}
	for i := 0; i < 4; i++ {
		g.tableView.PlayerHands[i] = len(g.game.Hand(i))
		g.tableView.TricksWon[i] = round.TricksWon(i)
	}
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
	g.tableView.TrickWinner = -1
	g.tableView.Trump = engine.NoSuit
	g.tableView.TurnPulseFrame = g.turnPulseFrame
	g.tableView.PlayerActions = g.playerAction

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

	// A teachable-moment popup takes over the screen until dismissed.
	if g.pendingPopup != nil {
		return g.renderPopup(width, height)
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

		phase := g.game.Phase()
		isYourTurn := g.game.CurrentPlayer() == g.humanPlayer

		legalPlays := make([]engine.Card, 0)
		if phase == engine.PhasePlay && isYourTurn {
			if round != nil && round.Trick() != nil {
				legalPlays = engine.LegalPlays(engine.NewHandWith(hand), round.Trick())
			}
		}

		// Only show selection when it's your turn to select a card
		// Must be in discard/play phase, your turn, and not waiting for acknowledgment
		selectedIdx := -1
		canSelect := (phase == engine.PhaseDiscard || phase == engine.PhasePlay) &&
			isYourTurn && !g.waitingForTrickAck && !g.waitingForRoundAck
		if canSelect {
			selectedIdx = g.selectedCard
		}

		handCards := components.RenderHand(hand, selectedIdx, legalPlays, g.tableView.Trump, g.coachPickIndex())

		// Diegetic controls: the keys live on the thing they act on rather than in
		// a separate footer legend. During card selection (play/discard) the hand
		// is flanked by ◄/► move arrows and the chosen verb tags the raised card;
		// cursor-less choices (bidding, defend) render as a chip row below it.
		handStr = g.renderHandArea(playerName, phase, isYourTurn, selectedIdx, len(hand), handCards)
	}

	// Fixed height keeps the table above the hand from shifting as the per-phase
	// controls (verb tag, chip row) appear and disappear. See handAreaHeight.
	handStr = lipgloss.NewStyle().Height(handAreaHeight).Render(handStr)

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

	// Build center content (table + hand)
	// Center the hand to match table width
	tableWidth := lipgloss.Width(tableStr)
	centeredHand := lipgloss.PlaceHorizontal(tableWidth, lipgloss.Center, handStr)
	centerContent := tableStr + centeredHand

	// Compose the main area. The side cards are pure team scoreboards flanking
	// the table: YOU on the left, OPP on the right. Global game state (round,
	// trump, contract) goes in a banner above the table, and each player's latest
	// action shows under their seat. Compact layout folds the scores into a single
	// scoreboard line instead of the flanking cards.
	var mainArea string
	if showPanels {
		centerHeight := lipgloss.Height(centerContent)
		// Flank the table: YOU card on the left, OPP card on the right, each
		// filled to the table height so the table stays vertically centered
		// between them. Each panel is 13 wide (panelInnerWidth + border).
		leftPanel := lipgloss.PlaceVertical(centerHeight, lipgloss.Top, g.renderYouCard())
		rightPanel := lipgloss.PlaceVertical(centerHeight, lipgloss.Top, g.renderOppCard())

		mainArea = lipgloss.JoinHorizontal(lipgloss.Top, leftPanel, centerContent, rightPanel)
	} else {
		scoreBar := lipgloss.PlaceHorizontal(tableWidth, lipgloss.Center, g.renderScoreBar())
		mainArea = lipgloss.JoinVertical(lipgloss.Center, scoreBar, centerContent)
	}
	// The main area is a constant width per layout mode (fixed panel + table
	// widths), so use it as the single content width for everything below.
	contentWidth := lipgloss.Width(mainArea)

	// slot renders s into a fixed-width, fixed-height box. Every region below the
	// main area is variable in length (the status message can wrap, the ticker
	// and banner appear over time, the coach box changes per tip), so pinning
	// each to a reserved size keeps the whole block a constant size. That in turn
	// stops the final centering from shifting the table and panels around — the
	// elements that are meant to stay fixed in place.
	slot := func(s string, h int) string {
		return lipgloss.NewStyle().
			Width(contentWidth).
			Height(h).
			MaxHeight(h).
			Align(lipgloss.Center).
			Render(s)
	}

	// Assemble vertical sections: [banner] · main area · status · [coach] · help.
	// Optional regions still reserve their rows when empty so that their first
	// appearance doesn't push the table down. Per-player actions now live under
	// each seat in the table, so there is no separate play-log region.
	var sections []string
	if showPanels {
		// Banner row + spacer below it.
		sections = append(sections, slot(g.renderContractBanner(contentWidth), 2))
		sections = append(sections, mainArea)
	} else {
		// Compact layout: a single score bar above the table (added below).
		sections = append(sections, mainArea)
	}
	// Pre-truncate the status to a single line so a long combined message can't
	// wrap and grow the slot vertically (which would shove the block up/down).
	// Centered within the fixed-width slot; MaxHeight keeps it from growing.
	statusLine := lipgloss.NewStyle().MaxWidth(contentWidth).Render(theme.Current.Accent.Render(phaseStr))
	sections = append(sections, slot(statusLine, 2))
	if g.tutorial {
		sections = append(sections, slot(g.renderCoachBox(contentWidth), coachBoxHeight))
	}

	innerContent := lipgloss.JoinVertical(lipgloss.Center, sections...)

	// Add celebration overlay if active
	if g.celebrationFrames > 0 {
		innerContent = g.addCelebrationEffect(innerContent)
	}

	// Pin the controls/help as a footer on the very bottom row (mirroring the
	// banner header at the top), and center the rest of the content in the space
	// above it. footer spans the full content width so it reads as a bottom bar.
	// Minimal, always-present corner controls. Everything situational now lives
	// on the board (see renderHandArea); only the two global keys sit here.
	footer := lipgloss.NewStyle().Width(width - 4).Align(lipgloss.Right).
		Render(theme.Current.Help.Render("esc quit · ? help"))
	footerHeight := lipgloss.Height(footer)
	// The "?" help sheet replaces the board content as a centered modal; the
	// frame and corner footer stay so it reads as an overlay, not a new screen.
	if g.showHelp {
		innerContent = g.renderHelpSheet()
	}
	body := lipgloss.Place(width-4, height-4-footerHeight, lipgloss.Center, lipgloss.Center, innerContent)
	centeredContent := body + "\n" + footer

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

	// The title ("Shuffling." → "...") and the deck art change width every frame
	// (single deck → split decks → merged). Center each within a constant width so
	// the whole block stays put instead of jittering as the animation plays.
	titleLine := lipgloss.PlaceHorizontal(shuffleArtWidth, lipgloss.Center, title)
	deckBlock := lipgloss.PlaceHorizontal(shuffleArtWidth, lipgloss.Center, deckArt)
	content := titleLine + "\n\n" + deckBlock

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

// Hand-area geometry. The block is a constant height so the table above it
// never shifts as per-phase controls appear and disappear:
// verb tag(1) + header(name + sub-line = 2) + cards(7) + chip row(1) = 11.
const (
	cardCellWidth  = 7  // matches a column width in components.RenderHand
	arrowCellWidth = 3  // gutter the ◄/► move arrows occupy beside the hand
	handAreaHeight = 11 // total reserved rows; see breakdown above
)

// keyCap renders a control hint as a highlighted key glyph followed by a muted
// label, e.g. "⏎ Play". Shared by the on-board controls, the chip row, and the
// help sheet so keys read consistently everywhere.
func keyCap(key, label string) string {
	k := lipgloss.NewStyle().Foreground(theme.ColGold).Bold(true).Render(key)
	return k + theme.Current.Muted.Render(" "+label)
}

// renderHandArea composes the diegetic control layer around the player's hand:
//
//	            ⏎ Play          verb tag over the raised card (play/discard)
//	  You (2) DEALER            name + optional sub-line (suit selector / hint)
//	◄ [ the hand of cards ] ►   move arrows when a card cursor is active
//	   P Pass · A Alone         chip row for cursor-less choices (bidding, etc.)
//
// Every row is reserved (blank when unused) so the block stays handAreaHeight
// tall and the table above never jumps between phases.
func (g *GamePlay) renderHandArea(playerName string, phase engine.GamePhase, isYourTurn bool, selectedIdx, handLen int, handCards string) string {
	arrowStyle := lipgloss.NewStyle().Foreground(theme.ColGold).Bold(true)

	// Header sub-line: suit selector during round-2 bidding, the discard hint
	// during the dealer's discard, else blank (still reserved for height).
	subLine := ""
	switch {
	case phase == engine.PhaseBidRound2 && isYourTurn && g.suitSelector != nil:
		subLine = g.suitSelector.Render()
	case phase == engine.PhaseDiscard && handLen == 6:
		subLine = theme.Current.Muted.Render("(select one to discard)")
	}
	header := lipgloss.JoinVertical(lipgloss.Center, playerName, subLine)

	// Move arrows flank the hand whenever a card cursor is active and there is
	// more than one card to move between.
	arrowsShown := selectedIdx >= 0 && handLen > 1
	handRow := handCards
	if arrowsShown {
		gutter := func(s string) string {
			return lipgloss.NewStyle().Width(arrowCellWidth).Align(lipgloss.Center).Render(s)
		}
		// Dim the boundary arrow: the cursor clamps (no wrap), so a lit arrow at
		// either end would imply a move that isn't possible.
		arrow := func(glyph string, active bool) string {
			if active {
				return arrowStyle.Render(glyph)
			}
			return theme.Current.Muted.Render(glyph)
		}
		handRow = lipgloss.JoinHorizontal(lipgloss.Center,
			gutter(arrow("◄", selectedIdx > 0)), handCards,
			gutter(arrow("►", selectedIdx < handLen-1)))
	}
	blockWidth := lipgloss.Width(handRow)

	// Verb tag floats over the raised (selected) card during play/discard. It is
	// left-padded to the selected card's column, so the whole block must be
	// assembled left-aligned (the table centers it as a unit afterwards).
	verbRow := ""
	if selectedIdx >= 0 {
		verb := map[engine.GamePhase]string{engine.PhasePlay: "Play", engine.PhaseDiscard: "Discard"}[phase]
		if verb != "" {
			tag := keyCap("⏎", verb)
			leftPad := selectedIdx * cardCellWidth
			if arrowsShown {
				leftPad += arrowCellWidth
			}
			leftPad += (cardCellWidth - lipgloss.Width(tag)) / 2 // nudge toward card center
			if leftPad < 0 {
				leftPad = 0
			}
			verbRow = strings.Repeat(" ", leftPad) + tag
		}
	}

	// Center the header and chip row over the hand; the verb tag stays
	// column-aligned. JoinVertical(Left) preserves the verb tag's left padding.
	center := func(s string) string { return lipgloss.PlaceHorizontal(blockWidth, lipgloss.Center, s) }
	return lipgloss.JoinVertical(lipgloss.Left,
		verbRow, center(header), handRow, center(g.handChips(isYourTurn)))
}

// handChips renders the cursor-less choices for the current state as a row of
// key caps (bidding, defend-alone, and the acknowledgement prompts). Play and
// discard return "" — their controls are the arrows and verb tag on the hand.
func (g *GamePlay) handChips(isYourTurn bool) string {
	sep := theme.Current.Muted.Render("   ")
	if g.waitingForRoundAck {
		if g.game.IsOver() {
			return keyCap("⏎", "Return to menu")
		}
		return keyCap("⏎", "Next round")
	}
	if g.waitingForTrickAck {
		return keyCap("⏎", "Continue")
	}
	if !isYourTurn {
		return ""
	}
	switch g.game.Phase() {
	case engine.PhaseBidRound1:
		return strings.Join([]string{keyCap("⏎", "Order up"), keyCap("P", "Pass"), keyCap("A", "Alone")}, sep)
	case engine.PhaseBidRound2:
		return strings.Join([]string{keyCap("⏎", "Call"), keyCap("P", "Pass")}, sep)
	case engine.PhaseDefendAlone:
		return strings.Join([]string{keyCap("Y", "Defend alone"), keyCap("N", "Decline")}, sep)
	}
	return ""
}

// renderHelpSheet is the full keybind reference shown as an in-place overlay
// when "?" is pressed (g.showHelp). It lists every control grouped by phase so
// players can see the whole scheme without leaving the game in progress.
func (g *GamePlay) renderHelpSheet() string {
	title := theme.Current.Accent.Bold(true).Render("Controls")
	row := func(keys, what string) string {
		return lipgloss.JoinHorizontal(lipgloss.Left,
			lipgloss.NewStyle().Width(12).Foreground(theme.ColGold).Bold(true).Render(keys),
			theme.Current.Muted.Render(what))
	}
	lines := []string{
		title,
		"",
		row("←/→  h/l", "Move card / suit cursor"),
		row("Enter", "Play · Discard · Order up · Call"),
		row("P", "Pass"),
		row("A", "Order up alone"),
		row("Y / N", "Defend alone / decline"),
		row("Enter", "Continue to next trick / round"),
		row("?", "Toggle this help"),
		row("Esc  q", "Quit to menu"),
		"",
		theme.Current.Muted.Italic(true).Render("Press any key to close"),
	}
	body := lipgloss.JoinVertical(lipgloss.Left, lines...)
	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(theme.ColBlue).
		Padding(1, 3).
		Render(body)
}

// renderYouCard renders the YOU scoreboard card at its NATURAL height; View()
// fills it to the table height so it flanks the left side of the table.
func (g *GamePlay) renderYouCard() string {
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
		panelTricks(yourTricks, theme.ColGreen),
	}

	return boxFrame("YOU", theme.ColGreen, lipgloss.JoinVertical(lipgloss.Center, body...), panelInnerWidth)
}

// Layout sizing. panelInnerWidth is the content width inside each side-panel box
// (outer = 13 incl. the rounded border). The min/full thresholds decide between
// the too-small guard, the compact (panel-less) layout, and the full HUD layout.
const (
	panelInnerWidth   = 11 // content width inside a scoreboard panel box (outer = 13 incl. border)
	minPlayableWidth  = 64 // table (~61 wide) + screen-border margin
	minPlayableHeight = 20 // enough rows for the table core

	// fullLayoutMinWidth gates the full HUD vs the compact fallback. It must clear
	// the table (~61) + both flanking scoreboard panels (13 each incl. border) +
	// the screen-border margin (2); below it we drop to the compact score-bar
	// layout. (= 89.)
	fullLayoutMinWidth = 61 + 2*13 + 2 // table + both flanking panels + screen border

	// shuffleArtWidth is a constant width the shuffle title and deck art are
	// centered within so the animation doesn't jitter as their widths change
	// frame to frame (widest frame is the split decks at ~22 cols).
	shuffleArtWidth = 24

	// coachBoxBodyLines is the reserved height of the coach callout's body; the
	// box renders at coachBoxHeight = header(1) + body + border(2) rows so it
	// stays a constant size between tips.
	coachBoxBodyLines = 4
	coachBoxHeight    = coachBoxBodyLines + 3
)

// boxFrame wraps inner content in the shared panel frame: a centered, bold,
// colored title bar atop a rounded border, at the given inner content width.
// Used by the YOU/OPP scoreboard cards (at panelInnerWidth).
func boxFrame(title string, accent lipgloss.TerminalColor, inner string, innerWidth int) string {
	header := lipgloss.NewStyle().
		Width(innerWidth).
		Align(lipgloss.Center).
		Bold(true).
		Background(accent).
		Foreground(lipgloss.Color("#FFFFFF")).
		Render(title)

	content := lipgloss.JoinVertical(lipgloss.Center, header, inner)
	return lipgloss.NewStyle().
		Width(innerWidth).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(theme.ColBlue).
		Render(content)
}

// panelCenter renders s centered to the panel inner width using the given style.
func panelCenter(style lipgloss.Style, s string) string {
	return style.Width(panelInnerWidth).Align(lipgloss.Center).Render(s)
}

// trickDots returns the 5-pip tracker string (filled in the team accent, empty
// muted), with no surrounding placement.
func trickDots(n int, accent lipgloss.TerminalColor) string {
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
	return s
}

// panelTricks stacks the "Tricks" label directly above the pip tracker and
// centers the pair as one unit, so the label sits centered over the bubbles
// (rather than each centering independently and ending up a column apart).
func panelTricks(n int, accent lipgloss.TerminalColor) string {
	label := theme.Current.Muted.Render("Tricks")
	block := lipgloss.JoinVertical(lipgloss.Center, label, trickDots(n, accent))
	return lipgloss.PlaceHorizontal(panelInnerWidth, lipgloss.Center, block)
}

// renderOppCard renders the opponents' scoreboard card (score + tricks) at its
// NATURAL height, mirroring renderYouCard; View() fills it to the table height
// so it flanks the right side of the table. Global state lives in the banner.
func (g *GamePlay) renderOppCard() string {
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
		panelTricks(oppTricks, theme.ColRed),
	}

	return boxFrame("OPP", theme.ColRed, lipgloss.JoinVertical(lipgloss.Center, body...), panelInnerWidth)
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
