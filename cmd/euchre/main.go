package main

import (
	"fmt"
	"os"

	"github.com/bran/euchre/internal/app"
	_ "github.com/bran/euchre/internal/variants/standard" // Register standard variant
	tea "github.com/charmbracelet/bubbletea"
	"github.com/urfave/cli/v2"
)

func main() {
	cliApp := &cli.App{
		Name:    "euchre",
		Usage:   "Learn and play the classic Euchre card game",
		Version: "0.1.0",
		Action:  runTUI,
		Commands: []*cli.Command{
			{
				Name:    "rules",
				Aliases: []string{"r"},
				Usage:   "Display Euchre rules",
				Action:  showRules,
				Subcommands: []*cli.Command{
					{
						Name:   "trump",
						Usage:  "Show trump card hierarchy",
						Action: showTrumpRules,
					},
					{
						Name:   "scoring",
						Usage:  "Show scoring rules",
						Action: showScoring,
					},
					{
						Name:   "bidding",
						Usage:  "Show bidding rules",
						Action: showBidding,
					},
				},
			},
			{
				Name:   "play",
				Usage:  "Start a game immediately",
				Action: runTUI,
			},
		},
	}

	if err := cliApp.Run(os.Args); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

// runTUI starts the TUI application
func runTUI(c *cli.Context) error {
	p := tea.NewProgram(app.New(), tea.WithAltScreen())
	_, err := p.Run()
	return err
}

// showRules displays general Euchre rules
func showRules(c *cli.Context) error {
	fmt.Print(`
EUCHRE RULES
============

Euchre is a trick-taking card game for 4 players in 2 teams.
Partners sit across from each other.

THE DECK
--------
24 cards: 9, 10, J, Q, K, A of each suit (Clubs, Diamonds, Hearts, Spades)

OBJECTIVE
---------
Be the first team to score 10 points by winning tricks.

DEALING
-------
Each player receives 5 cards. One card is turned face-up to suggest trump.

TRUMP
-----
The trump suit beats all other suits. Within trump:
  • Right Bower (Jack of trump) - HIGHEST
  • Left Bower (Jack of same color) - Second highest
  • A, K, Q, 10, 9 of trump

Note: The Left Bower is considered part of the trump suit!

PLAY
----
1. Player left of dealer leads first trick
2. Must follow suit if able
3. Highest card of led suit wins (unless trumped)
4. Winner of trick leads next

SCORING
-------
Making team (3-4 tricks): 1 point
March (all 5 tricks): 2 points
Alone march: 4 points
Euchred (< 3 tricks): Defenders get 2 points

Use 'euchre rules trump', 'euchre rules scoring', or 'euchre rules bidding' for more details.
`)
	return nil
}

// showTrumpRules displays trump hierarchy
func showTrumpRules(c *cli.Context) error {
	fmt.Print(`
TRUMP CARD HIERARCHY
====================

When a suit is trump, cards rank in this order (highest to lowest):

1. RIGHT BOWER - Jack of the trump suit
   Example: J♥ when Hearts is trump

2. LEFT BOWER - Jack of the same color
   Example: J♦ when Hearts is trump
   NOTE: The Left Bower BELONGS to the trump suit during play!

3. ACE of trump (A♥)
4. KING of trump (K♥)
5. QUEEN of trump (Q♥)
6. TEN of trump (10♥)
7. NINE of trump (9♥)

SAME COLOR PAIRS:
  • Hearts (red) ↔ Diamonds (red)
  • Spades (black) ↔ Clubs (black)

EXAMPLES:
---------
If HEARTS is trump:
  J♥ > J♦ > A♥ > K♥ > Q♥ > 10♥ > 9♥

If SPADES is trump:
  J♠ > J♣ > A♠ > K♠ > Q♠ > 10♠ > 9♠

IMPORTANT: If Hearts is trump and J♦ is led, you must follow with
HEARTS (not diamonds), because the Left Bower belongs to trump!
`)
	return nil
}

// showScoring displays scoring rules
func showScoring(c *cli.Context) error {
	fmt.Print(`
EUCHRE SCORING
==============

MAKING TEAM (called trump):
---------------------------
• Win 3 or 4 tricks: 1 point
• Win all 5 tricks (MARCH): 2 points
• Win all 5 tricks ALONE: 4 points

DEFENDING TEAM:
---------------
• EUCHRE (makers win < 3 tricks): 2 points

GAME
----
First team to reach 10 points wins.

GOING ALONE:
------------
A player may choose to "go alone" when ordering up or calling trump.
Their partner sits out for that hand.

If the lone player wins:
  • 3-4 tricks: 1 point (same as regular)
  • All 5 tricks: 4 points (double the march)

Risk: If euchred while alone, defenders still only get 2 points.

TRADITIONAL SCORING (Optional):
-------------------------------
Some players use cards to track score:
  • Two 5s or 6s per team
  • Overlap cards to show score 1-10
`)
	return nil
}

// showBidding displays bidding rules
func showBidding(c *cli.Context) error {
	fmt.Print(`
EUCHRE BIDDING
==============

After dealing, one card is turned face-up. This suggests a trump suit.

ROUND 1 - The Turn-Up
---------------------
Starting left of dealer, each player may:

  • ORDER UP - Accept the turned card's suit as trump
    - Dealer picks up the turned card
    - Dealer discards one card from their hand
    - The ordering player's team must win 3+ tricks

  • PASS - Decline to order up

If someone orders up, bidding ends and play begins.

ROUND 2 - Naming Trump
----------------------
If everyone passes in Round 1, the turned card is set aside.
Starting left of dealer, each player may:

  • NAME TRUMP - Call any suit EXCEPT the turned card's suit
    - The caller's team must win 3+ tricks

  • PASS - Decline to name trump

STICK THE DEALER (Optional Rule)
--------------------------------
If everyone passes in Round 2, the dealer MUST name trump.
This prevents endless redeals.

GOING ALONE
-----------
When ordering up or naming trump, you may declare "alone."
Your partner sits out, and you play 1 vs 2.
Reward: 4 points for winning all 5 tricks (instead of 2).

BIDDING STRATEGY
----------------
• Count your potential trump cards (including Left Bower!)
• 3+ trump is usually safe to bid
• Consider your partner's position
• Dealer has an advantage (gets the turn-up card)
`)
	return nil
}
