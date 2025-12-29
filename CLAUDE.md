# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

A terminal-based Euchre card game written in Go using the Bubble Tea TUI framework. The game features a learning-oriented design with tutorials, AI opponents, and support for variant rules.

## Build Commands

```bash
make build      # Build binary to bin/euchre
make run        # Run the application
make test       # Run all tests with verbose output
make lint       # Run golangci-lint
make coverage   # Generate test coverage report
```

Run a single test:
```bash
go test -v -run TestName ./internal/engine/
```

## Architecture

### Core Engine (`internal/engine/`)

The game engine is stateless and action-based:
- **Game** - Manages overall game state, scores, and round lifecycle
- **Round** - Handles a single deal: bidding, discard, and trick play
- **Trick** - Manages cards played in a single trick
- **Card/Deck** - Card primitives with Euchre-specific logic (bowers, effective suits)

Game flow is driven by `Action` interface implementations:
- `PassAction`, `OrderUpAction`, `CallTrumpAction` - Bidding actions
- `DiscardAction` - Dealer discards after pickup
- `PlayCardAction` - Playing cards to tricks

Key engine concepts:
- **Bowers**: Jack of trump (Right Bower) and Jack of same color (Left Bower) are the highest trump cards
- **Effective Suit**: Left Bower belongs to trump suit, not its printed suit - use `Card.EffectiveSuit(trump)` for suit-following logic
- **Game Phases**: `PhaseBidRound1` -> `PhaseBidRound2` (if all pass) -> `PhaseDiscard` (if ordered up) -> `PhasePlay` -> `PhaseRoundEnd`

### Variants System (`internal/variants/`)

Variants are registered via init() and retrieved from a global registry:
```go
import _ "github.com/bran/euchre/internal/variants/standard" // Auto-registers
variants.Get("standard")
```

The `Variant` interface defines all configurable rules (stick-the-dealer, going alone, scoring, etc.).

### AI System (`internal/ai/`)

- `Player` interface defines AI decision points: `DecideBid`, `DecidePlay`, `DecideDiscard`
- `Strategy` interface provides pluggable algorithms for hand evaluation and card selection
- `rule_based/` contains the default rule-based AI implementation

### TUI Application (`internal/app/`)

Built on Bubble Tea's Elm architecture:
- `App` - Root model managing screen navigation via `NavigateMsg`
- Screen models: `MainMenu`, `GameSetup`, `GamePlay`, `QuickReference`, `LearningJourney`
- Navigate between screens: `app.Navigate(app.ScreenGamePlay)`

### UI Components (`internal/ui/`)

- `components/` - Reusable TUI components (card rendering, menus, tables)
- `theme/` - Color schemes and styling

## Testing

Tests use standard Go testing. Engine tests cover card mechanics, trick resolution, and round state transitions. AI tests verify bidding thresholds and play selection logic.
