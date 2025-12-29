package content

import "github.com/bran/euchre/internal/tutorial"

func init() {
	registerTrumpLessons()
}

func registerTrumpLessons() {
	tutorial.Register(&tutorial.Lesson{
		ID:            "rules-1",
		Title:         "The Bower System",
		Description:   "Learn about the unique trump hierarchy in Euchre",
		Category:      tutorial.CategoryRules,
		Order:         5,
		Prerequisites: []string{"basics-4"},
		Sections: []tutorial.Section{
			{
				Type:  tutorial.SectionText,
				Title: "Right Bower - The Highest Card",
				Content: `The Jack of the trump suit is called the "Right Bower"
and it's the HIGHEST card in the game.

Example: If Hearts is trump:
  J♥ = Right Bower = Highest card!

Nothing beats the Right Bower. If you have it, you have
a guaranteed trick winner.`,
			},
			{
				Type:  tutorial.SectionText,
				Title: "Left Bower - The Second Highest",
				Content: `Here's where Euchre gets interesting!

The Jack of the SAME COLOR as trump becomes the "Left Bower"
and is the second-highest card.

Color pairs:
  ♥ Hearts (red) ↔ ♦ Diamonds (red)
  ♠ Spades (black) ↔ ♣ Clubs (black)

Example: If Hearts is trump:
  J♥ = Right Bower (highest)
  J♦ = Left Bower (second highest)

The Left Bower BELONGS to the trump suit during play!
If someone leads hearts and you have J♦, you must play it!`,
			},
			{
				Type:  tutorial.SectionText,
				Title: "Complete Trump Hierarchy",
				Content: `When Hearts is trump, cards rank (highest to lowest):

TRUMP CARDS:
  J♥ Right Bower (highest)
  J♦ Left Bower
  A♥
  K♥
  Q♥
  10♥
  9♥ (lowest trump)

OFF-SUIT CARDS:
  Rank normally: A > K > Q > J > 10 > 9

Remember: Even the 9 of trump beats the Ace of any
other suit!`,
			},
		},
	})

	tutorial.Register(&tutorial.Lesson{
		ID:            "rules-2",
		Title:         "Following Suit",
		Description:   "The most important rule of trick-taking",
		Category:      tutorial.CategoryRules,
		Order:         6,
		Prerequisites: []string{"rules-1"},
		Sections: []tutorial.Section{
			{
				Type:  tutorial.SectionText,
				Title: "The Follow Suit Rule",
				Content: `The most fundamental rule in Euchre:

  IF YOU CAN FOLLOW SUIT, YOU MUST!

When a card is led:
1. Look at its EFFECTIVE suit
2. If you have any cards of that suit, play one
3. Only if you have NONE, may you play anything else

This rule applies to everyone except the leader,
who can play any card to start the trick.`,
			},
			{
				Type:  tutorial.SectionText,
				Title: "The Left Bower Exception",
				Content: `Remember: The Left Bower belongs to the trump suit!

Example (Hearts is trump, Diamonds led):
• You have: K♦, 10♦, J♦ (Left Bower)
• You MUST play K♦ or 10♦
• The J♦ is NOT a diamond anymore - it's a heart!

Example (Hearts is trump, Hearts led):
• You have: K♦, 10♦, J♦ (Left Bower)
• You MUST play J♦
• It's the only heart you have!

This catches many new players off guard!`,
			},
			{
				Type:  tutorial.SectionText,
				Title: "When You Can't Follow",
				Content: `If you have no cards of the led suit, you have options:

1. TRUMP IN - Play a trump card to win the trick
   • Any trump beats any non-trump
   • Be careful not to waste high trumps!

2. DISCARD - Play any off-suit card
   • This card cannot win
   • Use it to get rid of low cards
   • Signal to your partner

Strategic choice: Trump in if you can win, discard if
your partner is already winning the trick!`,
			},
		},
	})

	tutorial.Register(&tutorial.Lesson{
		ID:            "rules-3",
		Title:         "Scoring in Detail",
		Description:   "How points are awarded in Euchre",
		Category:      tutorial.CategoryRules,
		Order:         7,
		Prerequisites: []string{"basics-4"},
		Sections: []tutorial.Section{
			{
				Type:  tutorial.SectionText,
				Title: "Making Your Bid",
				Content: `When your team calls trump ("makes"), you need to win
at least 3 tricks:

Making (3-4 tricks): 1 point
  "We made it!" - minimum successful bid

March (all 5 tricks): 2 points
  "We marched!" - swept the hand

Your team called trump, so you're expected to win.
Getting only 1 point is the baseline.`,
			},
			{
				Type:  tutorial.SectionText,
				Title: "Getting Euchred",
				Content: `If your team calls trump but wins only 0, 1, or 2 tricks:

  YOU'RE EUCHRED!
  The OTHER team scores 2 points.

This is a big swing - you expected to score but instead
your opponents score double!

This is why you should only call trump when you have
a strong hand.`,
			},
			{
				Type:  tutorial.SectionText,
				Title: "Going Alone",
				Content: `A player can choose to "go alone" when calling trump.
Their partner sits out for that hand.

Alone March (all 5 tricks): 4 points!
  This is the biggest score in Euchre.

Alone but less than 5 tricks: 1 point
  Same as regular making

Euchred while alone: 2 points to opponents
  Same as regular euchre

Going alone is risky but can win games quickly!`,
			},
		},
	})
}
