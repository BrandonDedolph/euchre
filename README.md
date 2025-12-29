# Euchre

A terminal-based Euchre card game written in Go using the [Bubble Tea](https://github.com/charmbracelet/bubbletea) TUI framework.

```
    ███████╗██╗   ██╗ ██████╗██╗  ██╗██████╗ ███████╗
    ██╔════╝██║   ██║██╔════╝██║  ██║██╔══██╗██╔════╝
    █████╗  ██║   ██║██║     ███████║██████╔╝█████╗
    ██╔══╝  ██║   ██║██║     ██╔══██║██╔══██╗██╔══╝
    ███████╗╚██████╔╝╚██████╗██║  ██║██║  ██║███████╗
    ╚══════╝ ╚═════╝  ╚═════╝╚═╝  ╚═╝╚═╝  ╚═╝╚══════╝
```

## Features

- **Interactive TUI** - Beautiful terminal interface with card rendering and animations
- **AI Opponents** - Play against rule-based AI opponents with strategic bidding and play
- **Learn to Play** - Interactive tutorials covering basics, trump mechanics, bidding, and strategy
- **Quick Reference** - In-game rules and card rankings reference
- **Variant Support** - Extensible variants system for different rule sets (standard rules included)

## Installation

### Prerequisites

- [asdf](https://asdf-vm.com/) version manager (recommended) - `make run` will automatically install the correct Go version
- Or Go 1.21+ installed manually

### Build from Source

```bash
git clone https://github.com/BrandonDedolph/euchre.git
cd euchre
make build
```

The binary will be created at `bin/euchre`.

## Usage

### Launch the TUI

```bash
./bin/euchre
# or
make run
```

### CLI Commands

View rules directly from the command line:

```bash
# General rules overview
./bin/euchre rules

# Specific rule sections
./bin/euchre rules trump     # Trump card hierarchy
./bin/euchre rules scoring   # Scoring rules
./bin/euchre rules bidding   # Bidding rules
```

## How to Play Euchre

Euchre is a trick-taking card game for 4 players in 2 teams. Partners sit across from each other.

### The Deck

24 cards: 9, 10, J, Q, K, A of each suit

### Objective

Be the first team to score 10 points by winning tricks.

### Trump Hierarchy

When a suit is trump, cards rank (highest to lowest):

1. **Right Bower** - Jack of trump suit
2. **Left Bower** - Jack of same color (belongs to trump suit!)
3. A, K, Q, 10, 9 of trump

### Scoring

| Outcome | Points |
|---------|--------|
| Win 3-4 tricks (makers) | 1 point |
| Win all 5 tricks (march) | 2 points |
| Win all 5 tricks alone | 4 points |
| Euchred (makers win < 3) | Defenders get 2 points |

## Development

### Build Commands

```bash
make build      # Build binary to bin/euchre
make run        # Run the application
make test       # Run all tests with verbose output
make lint       # Run golangci-lint
make coverage   # Generate test coverage report
make clean      # Remove build artifacts
```

### Run a Single Test

```bash
go test -v -run TestName ./internal/engine/
```

### Project Structure

```
├── cmd/euchre/        # Main application entry point
├── internal/
│   ├── ai/            # AI player implementations
│   │   └── rule_based/  # Rule-based AI strategy
│   ├── app/           # TUI screens (menu, game, tutorials)
│   ├── engine/        # Core game logic (cards, tricks, rounds)
│   ├── tutorial/      # Tutorial content and lesson system
│   ├── ui/            # UI components and theming
│   │   ├── components/  # Reusable TUI components
│   │   └── theme/       # Color schemes and styling
│   └── variants/      # Game rule variants
│       └── standard/    # Standard Euchre rules
```

## Controls

| Key | Action |
|-----|--------|
| `↑`/`k` | Navigate up |
| `↓`/`j` | Navigate down |
| `←`/`h` | Navigate left |
| `→`/`l` | Navigate right |
| `Enter`/`Space` | Select |
| `Esc`/`q` | Back/Quit |

## License

MIT
