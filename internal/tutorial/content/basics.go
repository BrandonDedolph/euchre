package content

import "github.com/bran/euchre/internal/tutorial"

func init() {
	registerBasicsLessons()
}

func registerBasicsLessons() {
	tutorial.Register(&tutorial.Lesson{
		ID:          "basics-1",
		Title:       "What is Euchre?",
		Description: "An introduction to the game of Euchre",
		Category:    tutorial.CategoryBasics,
		Order:       1,
		Sections: []tutorial.Section{
			{
				Type:  tutorial.SectionText,
				Title: "Welcome to Euchre!",
				Content: `Euchre is a classic trick-taking card game that's particularly popular in
the Midwestern United States, Canada, and parts of England.

It's a fast-paced, strategic game that's perfect for four players
working in two teams of partners.

Euchre is known for:
• Quick games (usually 15-30 minutes)
• Simple rules with strategic depth
• The unique "bower" system that makes Jacks special
• Being the game that introduced the Joker to playing cards!`,
			},
			{
				Type:  tutorial.SectionText,
				Title: "The Teams",
				Content: `Euchre is played with 4 players in 2 teams:
• Partners sit across from each other
• You and your partner work together to win tricks
• Communication is limited to the cards you play

Team 1: You (South) and Partner (North)
Team 2: West and East

       Partner
         │
    West─┼─East
         │
        You`,
			},
		},
	})

	tutorial.Register(&tutorial.Lesson{
		ID:          "basics-2",
		Title:       "The Cards",
		Description: "Learn about the Euchre deck",
		Category:    tutorial.CategoryBasics,
		Order:       2,
		Sections: []tutorial.Section{
			{
				Type:  tutorial.SectionText,
				Title: "The Euchre Deck",
				Content: `Euchre uses a 24-card deck, which is a standard deck with
cards 2 through 8 removed.

The cards in each suit (from high to low):
  A K Q J 10 9

That's only 6 cards per suit!

The four suits are:
  ♠ Spades (black)
  ♣ Clubs (black)
  ♥ Hearts (red)
  ♦ Diamonds (red)

The color groupings are important for the "bower" system
you'll learn about soon.`,
			},
			{
				Type:  tutorial.SectionText,
				Title: "Dealing",
				Content: `Each player receives 5 cards.

After dealing, one card is turned face-up in the center.
This card suggests what suit might become "trump."

The remaining cards form a small pile called the "kitty"
which usually isn't used.`,
			},
		},
	})

	tutorial.Register(&tutorial.Lesson{
		ID:          "basics-3",
		Title:       "The Objective",
		Description: "Learn how to win at Euchre",
		Category:    tutorial.CategoryBasics,
		Order:       3,
		Sections: []tutorial.Section{
			{
				Type:  tutorial.SectionText,
				Title: "Winning the Game",
				Content: `The goal in Euchre is to be the first team to score 10 points.

You score points by winning "tricks" - rounds where each player
plays one card and the highest card wins.

There are 5 tricks per hand, and your team needs to win at
least 3 of them to score.

Key concepts:
• Win 3-4 tricks: Score 1 point
• Win all 5 tricks: Score 2 points (called a "march")
• Fail to win 3 tricks when you called trump: Other team
  scores 2 points (called getting "euchred")`,
			},
			{
				Type:  tutorial.SectionText,
				Title: "What is a Trick?",
				Content: `A trick is one round of play where:
1. One player leads by playing a card
2. Each other player plays one card (clockwise)
3. The highest card wins the trick
4. The winner leads the next trick

Important rule: You must "follow suit" if you can!
If spades are led, you must play a spade if you have one.`,
			},
		},
	})

	tutorial.Register(&tutorial.Lesson{
		ID:            "basics-4",
		Title:         "What is Trump?",
		Description:   "Understanding the most important concept in Euchre",
		Category:      tutorial.CategoryBasics,
		Order:         4,
		Prerequisites: []string{"basics-3"},
		Sections: []tutorial.Section{
			{
				Type:  tutorial.SectionText,
				Title: "The Power of Trump",
				Content: `"Trump" is a suit that becomes more powerful than all other suits
for that hand.

When a suit is trump:
• Any trump card beats any non-trump card
• You can "trump in" when you can't follow suit

Example:
If Hearts is trump and someone leads the A♠ (Ace of Spades),
even the lowly 9♥ (Nine of Hearts) will beat it!

This is why choosing the right trump is so important.`,
			},
			{
				Type:  tutorial.SectionText,
				Title: "How Trump is Chosen",
				Content: `Trump is chosen through "bidding" at the start of each hand.

Round 1:
• A card is turned face-up (e.g., J♦)
• Players can accept that suit as trump ("order it up")
• Or pass to the next player

Round 2 (if everyone passed):
• Players can name ANY OTHER suit as trump
• Or pass again

If you or your partner calls trump, your team must win at
least 3 tricks or you'll be "euchred"!`,
			},
		},
	})
}
