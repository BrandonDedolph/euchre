package content

import (
	"github.com/bran/euchre/internal/engine"
	"github.com/bran/euchre/internal/tutorial"
)

func init() {
	registerBasicsLessons()
}

func registerBasicsLessons() {
	// Lesson 1: What is Euchre?
	tutorial.Register(&tutorial.Lesson{
		ID:          "basics-1",
		Title:       "What is Euchre?",
		Description: "An introduction to the game of Euchre",
		Category:    tutorial.CategoryBasics,
		Order:       1,
		VisualSections: []tutorial.VisualSection{
			{
				Title:      "The Teams",
				TextBefore: "Euchre is a 4-player partnership game.\nYou and your partner sit across from each other.",
				Visual: tutorial.NewTableLayoutVisual(
					[4]tutorial.PlayerPosition{
						{Label: "You", Style: tutorial.AnnotationHighlight},
						{Label: "Opponent", Style: tutorial.AnnotationDim},
						{Label: "Partner", Style: tutorial.AnnotationHighlight},
						{Label: "Opponent", Style: tutorial.AnnotationDim},
					},
					"Partners sit across from each other",
				),
				TextAfter: "Work together to win tricks and score points!",
			},
			{
				Title:      "The Goal",
				TextBefore: "Win tricks by playing the highest card.",
				Visual: &tutorial.VisualElement{
					Type: tutorial.VisualTrickDemo,
					Cards: []engine.Card{
						{Suit: engine.Hearts, Rank: engine.King},    // You
						{Suit: engine.Hearts, Rank: engine.Nine},    // Left
						{Suit: engine.Hearts, Rank: engine.Ace},     // Partner - winner
						{Suit: engine.Hearts, Rank: engine.Ten},     // Right
					},
					Caption: "The highest card wins the trick",
					Annotations: []tutorial.CardAnnotation{
						{CardIndex: 2, Label: "Winner!", Style: tutorial.AnnotationWinner},
					},
				},
				TextAfter: "First team to 10 points wins the game!",
			},
		},
	})

	// Lesson 2: The Cards
	tutorial.Register(&tutorial.Lesson{
		ID:          "basics-2",
		Title:       "The Cards",
		Description: "Learn about the Euchre deck",
		Category:    tutorial.CategoryBasics,
		Order:       2,
		VisualSections: []tutorial.VisualSection{
			{
				Title:      "The Euchre Deck",
				TextBefore: "Euchre uses only 24 cards (9 through Ace in each suit):",
				Visual: tutorial.NewCardGridVisual(
					[]engine.Card{
						// Spades row
						{Suit: engine.Spades, Rank: engine.Nine},
						{Suit: engine.Spades, Rank: engine.Ten},
						{Suit: engine.Spades, Rank: engine.Jack},
						{Suit: engine.Spades, Rank: engine.Queen},
						{Suit: engine.Spades, Rank: engine.King},
						{Suit: engine.Spades, Rank: engine.Ace},
						// Clubs row
						{Suit: engine.Clubs, Rank: engine.Nine},
						{Suit: engine.Clubs, Rank: engine.Ten},
						{Suit: engine.Clubs, Rank: engine.Jack},
						{Suit: engine.Clubs, Rank: engine.Queen},
						{Suit: engine.Clubs, Rank: engine.King},
						{Suit: engine.Clubs, Rank: engine.Ace},
						// Hearts row
						{Suit: engine.Hearts, Rank: engine.Nine},
						{Suit: engine.Hearts, Rank: engine.Ten},
						{Suit: engine.Hearts, Rank: engine.Jack},
						{Suit: engine.Hearts, Rank: engine.Queen},
						{Suit: engine.Hearts, Rank: engine.King},
						{Suit: engine.Hearts, Rank: engine.Ace},
						// Diamonds row
						{Suit: engine.Diamonds, Rank: engine.Nine},
						{Suit: engine.Diamonds, Rank: engine.Ten},
						{Suit: engine.Diamonds, Rank: engine.Jack},
						{Suit: engine.Diamonds, Rank: engine.Queen},
						{Suit: engine.Diamonds, Rank: engine.King},
						{Suit: engine.Diamonds, Rank: engine.Ace},
					},
					6, // 6 cards per row
					"Cards 2-8 are removed from a standard deck",
				),
			},
			{
				Title:      "The Four Suits",
				TextBefore: "There are four suits in two colors:",
				Visual: &tutorial.VisualElement{
					Type: tutorial.VisualCardRow,
					Cards: []engine.Card{
						{Suit: engine.Spades, Rank: engine.Ace},
						{Suit: engine.Clubs, Rank: engine.Ace},
						{Suit: engine.Hearts, Rank: engine.Ace},
						{Suit: engine.Diamonds, Rank: engine.Ace},
					},
					Caption: "Black: Spades & Clubs | Red: Hearts & Diamonds",
				},
				TextAfter: "The color pairs are important for the bower system!",
			},
			{
				Title:      "Your Hand",
				TextBefore: "Each player receives 5 cards.",
				Visual: tutorial.NewHandVisual(
					[]engine.Card{
						{Suit: engine.Hearts, Rank: engine.Jack},
						{Suit: engine.Hearts, Rank: engine.Ace},
						{Suit: engine.Spades, Rank: engine.King},
						{Suit: engine.Diamonds, Rank: engine.Queen},
						{Suit: engine.Clubs, Rank: engine.Nine},
					},
					nil, // no highlights
					engine.NoSuit,
					"A typical 5-card Euchre hand",
				),
				TextAfter: "One card is turned face-up to suggest trump.",
			},
		},
	})

	// Lesson 3: The Objective
	tutorial.Register(&tutorial.Lesson{
		ID:          "basics-3",
		Title:       "The Objective",
		Description: "Learn how to win at Euchre",
		Category:    tutorial.CategoryBasics,
		Order:       3,
		VisualSections: []tutorial.VisualSection{
			{
				Title:      "What is a Trick?",
				TextBefore: "A trick is one round where each player plays a card.",
				Visual: &tutorial.VisualElement{
					Type: tutorial.VisualTrickDemo,
					Cards: []engine.Card{
						{Suit: engine.Spades, Rank: engine.Ten},   // You
						{Suit: engine.Spades, Rank: engine.Nine},  // Left
						{Suit: engine.Spades, Rank: engine.King},  // Partner
						{Suit: engine.Spades, Rank: engine.Queen}, // Right
					},
					Caption: "Spades led - everyone must follow suit",
					Annotations: []tutorial.CardAnnotation{
						{CardIndex: 2, Label: "Highest", Style: tutorial.AnnotationWinner},
					},
				},
				TextAfter: "The highest card wins. Winner leads the next trick.",
			},
			{
				Title:      "Winning a Hand",
				TextBefore: "Your team needs to win at least 3 of 5 tricks.",
				Visual: tutorial.NewCardComparisonVisual(
					[]engine.Card{
						{Suit: engine.Hearts, Rank: engine.Ace},
						{Suit: engine.Hearts, Rank: engine.King},
						{Suit: engine.Hearts, Rank: engine.Queen},
					},
					[]engine.Card{
						{Suit: engine.Spades, Rank: engine.Ace},
						{Suit: engine.Spades, Rank: engine.King},
					},
					"Your Team: 3",
					"Opponents: 2",
					engine.NoSuit,
				),
				TextAfter: "3 tricks = You win the hand!",
			},
			{
				Title:      "Scoring",
				TextBefore: `Points depend on how many tricks you win:

  ●●●○○  3-4 tricks = 1 point
  ●●●●●  5 tricks   = 2 points (march!)

Your team calls trump, so you must win at least 3.`,
				TextAfter: "Fail to win 3? You're 'euchred' - opponents get 2 points!",
			},
		},
	})

	// Lesson 4: What is Trump?
	tutorial.Register(&tutorial.Lesson{
		ID:            "basics-4",
		Title:         "What is Trump?",
		Description:   "Understanding the most important concept in Euchre",
		Category:      tutorial.CategoryBasics,
		Order:         4,
		Prerequisites: []string{"basics-3"},
		VisualSections: []tutorial.VisualSection{
			{
				Title:      "Trump Beats All",
				TextBefore: "When a suit is 'trump', it beats all other suits!",
				Visual: &tutorial.VisualElement{
					Type: tutorial.VisualCardComparison,
					Cards: []engine.Card{
						{Suit: engine.Spades, Rank: engine.Ace},  // Left: high non-trump
						{Suit: engine.Hearts, Rank: engine.Nine}, // Right: low trump
					},
					Trump:      engine.Hearts,
					LeftLabel:  "A♠ (Not Trump)",
					RightLabel: "9♥ (Trump)",
					Caption:    "Even the lowest trump beats the highest non-trump!",
					Annotations: []tutorial.CardAnnotation{
						{CardIndex: 0, Label: "Loses", Style: tutorial.AnnotationLoser},
						{CardIndex: 1, Label: "Wins!", Style: tutorial.AnnotationWinner},
					},
				},
			},
			{
				Title:      "Trumping In",
				TextBefore: "If you can't follow suit, you can play trump to win.",
				Visual: &tutorial.VisualElement{
					Type: tutorial.VisualTrickDemo,
					Cards: []engine.Card{
						{Suit: engine.Spades, Rank: engine.Ace},   // You led
						{Suit: engine.Spades, Rank: engine.King},  // Left followed
						{Suit: engine.Hearts, Rank: engine.Nine},  // Partner trumped!
						{Suit: engine.Spades, Rank: engine.Queen}, // Right followed
					},
					Trump:   engine.Hearts,
					Caption: "Partner has no spades - trumps in with 9♥!",
					Annotations: []tutorial.CardAnnotation{
						{CardIndex: 2, Label: "Trump wins!", Style: tutorial.AnnotationWinner},
					},
				},
				TextAfter: "Hearts is trump. Partner's 9♥ beats all the spades!",
			},
			{
				Title:      "Choosing Trump",
				TextBefore: "Trump is chosen through bidding at the start.",
				Visual: tutorial.NewSingleCardVisual(
					engine.Card{Suit: engine.Diamonds, Rank: engine.Jack},
					"Turned-up card",
					engine.Diamonds,
				),
				TextAfter: "The turned-up card suggests a trump suit.\nPlayers can accept it or choose a different suit.",
			},
		},
	})
}
