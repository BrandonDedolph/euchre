# Euchre

![Go Version](https://img.shields.io/badge/Go-1.21+-00ADD8?logo=go)
![License](https://img.shields.io/badge/License-MIT-blue)
![Platform](https://img.shields.io/badge/Platform-macOS%20%7C%20Linux%20%7C%20Windows-lightgrey)

A terminal-based Euchre card game with AI opponents, built with [Bubble Tea](https://github.com/charmbracelet/bubbletea).

```
    ███████╗██╗   ██╗ ██████╗██╗  ██╗██████╗ ███████╗
    ██╔════╝██║   ██║██╔════╝██║  ██║██╔══██╗██╔════╝
    █████╗  ██║   ██║██║     ███████║██████╔╝█████╗
    ██╔══╝  ██║   ██║██║     ██╔══██║██╔══██╗██╔══╝
    ███████╗╚██████╔╝╚██████╗██║  ██║██║  ██║███████╗
    ╚══════╝ ╚═════╝  ╚═════╝╚═╝  ╚═╝╚═╝  ╚═╝╚══════╝
```

## Quick Start

```bash
git clone https://github.com/BrandonDedolph/euchre.git
cd euchre
make run
```

## Features

- **Interactive TUI** - Card rendering, animations, and visual suit selector
- **AI Opponents** - Rule-based AI with strategic bidding and play
- **Learn to Play** - Interactive tutorials for beginners
- **Quick Reference** - In-game rules with visual card examples

## Controls

| Key | Action |
|-----|--------|
| `↑↓` or `jk` | Navigate |
| `←→` or `hl` | Select card/suit |
| `Enter` | Confirm |
| `p` | Pass (bidding) |
| `Esc` | Back/Quit |

## Euchre Basics

4 players, 2 teams, 24 cards (9-A). First to 10 points wins.

**Trump Hierarchy:** Right Bower (J of trump) > Left Bower (J of same color) > A > K > Q > 10 > 9

**Scoring:** 3-4 tricks = 1pt | March (5 tricks) = 2pts | Alone march = 4pts | Euchred = 2pts to defenders

## Development

```bash
make build      # Build to bin/euchre
make test       # Run tests
make lint       # Run linter
```

<details>
<summary>Project Structure</summary>

```
cmd/euchre/          # Entry point
internal/
  ai/rule_based/     # AI strategy
  app/               # TUI screens
  engine/            # Game logic
  tutorial/          # Tutorial system
  ui/components/     # UI components
  variants/          # Rule variants
```
</details>

## Acknowledgments

Built with the [Charm](https://charm.sh) ecosystem: [Bubble Tea](https://github.com/charmbracelet/bubbletea), [Lip Gloss](https://github.com/charmbracelet/lipgloss).

## License

MIT
