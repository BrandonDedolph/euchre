package content

import (
	"github.com/bran/euchre/internal/engine"
	"github.com/bran/euchre/internal/tutorial"
)

func init() {
	registerTrumpLessons()
}

func registerTrumpLessons() {
	// Lesson 5: The Bower System
	tutorial.Register(&tutorial.Lesson{
		ID:            "rules-1",
		Title:         "The Bower System",
		Description:   "Learn about the unique trump hierarchy in Euchre",
		Category:      tutorial.CategoryRules,
		Order:         5,
		Prerequisites: []string{"basics-4"},
		VisualSections: []tutorial.VisualSection{
			{
				Title:      "Right Bower",
				TextBefore: "The Jack of trump is the HIGHEST card in the game!",
				Visual: tutorial.NewSingleCardVisual(
					engine.Card{Suit: engine.Hearts, Rank: engine.Jack},
					"Right Bower - HIGHEST",
					engine.Hearts,
				),
				TextAfter: "When Hearts is trump, J♥ beats everything.",
			},
			{
				Title:      "Left Bower",
				TextBefore: "The Jack of the same COLOR is the second-highest!",
				Visual: &tutorial.VisualElement{
					Type: tutorial.VisualCardComparison,
					Cards: []engine.Card{
						{Suit: engine.Hearts, Rank: engine.Jack},
						{Suit: engine.Diamonds, Rank: engine.Jack},
					},
					Trump:      engine.Hearts,
					LeftLabel:  "Right Bower",
					RightLabel: "Left Bower",
					Caption:    "Hearts & Diamonds are both red",
					Annotations: []tutorial.CardAnnotation{
						{CardIndex: 0, Label: "1st", Style: tutorial.AnnotationHighlight},
						{CardIndex: 1, Label: "2nd", Style: tutorial.AnnotationHighlight},
					},
				},
				TextAfter: "The Left Bower BELONGS to trump during play!",
			},
			{
				Title:      "Color Pairs",
				TextBefore: "Remember which suits are paired by color:",
				Visual: &tutorial.VisualElement{
					Type: tutorial.VisualCardRow,
					Cards: []engine.Card{
						{Suit: engine.Hearts, Rank: engine.Jack},
						{Suit: engine.Diamonds, Rank: engine.Jack},
						{Suit: engine.Spades, Rank: engine.Jack},
						{Suit: engine.Clubs, Rank: engine.Jack},
					},
					Caption: "Red: Hearts ↔ Diamonds | Black: Spades ↔ Clubs",
				},
			},
			{
				Title:      "Trump Hierarchy",
				TextBefore: "When Hearts is trump, cards rank:",
				Visual:     tutorial.NewTrumpHierarchyVisual(engine.Hearts),
				TextAfter:  "Even the 9♥ beats any non-trump card!",
			},
		},
	})

	// Lesson 6: Following Suit
	tutorial.Register(&tutorial.Lesson{
		ID:            "rules-2",
		Title:         "Following Suit",
		Description:   "The most important rule of trick-taking",
		Category:      tutorial.CategoryRules,
		Order:         6,
		Prerequisites: []string{"rules-1"},
		VisualSections: []tutorial.VisualSection{
			{
				Title:      "The Rule",
				TextBefore: "If you CAN follow suit, you MUST!",
				Visual: &tutorial.VisualElement{
					Type: tutorial.VisualHand,
					Cards: []engine.Card{
						{Suit: engine.Spades, Rank: engine.Ace},
						{Suit: engine.Spades, Rank: engine.Ten},
						{Suit: engine.Hearts, Rank: engine.King},
						{Suit: engine.Diamonds, Rank: engine.Queen},
						{Suit: engine.Clubs, Rank: engine.Nine},
					},
					HighlightIndices: []int{0, 1}, // Highlight spades
					Caption:          "Spades led - you MUST play one of your spades",
				},
				TextAfter: "Only if you have NONE may you play something else.",
			},
			{
				Title:      "Left Bower Exception",
				TextBefore: "The Left Bower belongs to trump, not its printed suit!",
				Visual: &tutorial.VisualElement{
					Type: tutorial.VisualHand,
					Cards: []engine.Card{
						{Suit: engine.Diamonds, Rank: engine.King},
						{Suit: engine.Diamonds, Rank: engine.Ten},
						{Suit: engine.Diamonds, Rank: engine.Jack}, // Left bower when Hearts trump
						{Suit: engine.Spades, Rank: engine.Queen},
						{Suit: engine.Clubs, Rank: engine.Nine},
					},
					Trump:            engine.Hearts,
					HighlightIndices: []int{0, 1}, // Only K♦ and 10♦ are diamonds
					Caption:          "Hearts is trump, Diamonds led",
					Annotations: []tutorial.CardAnnotation{
						{CardIndex: 2, Label: "Not a Diamond!", Style: tutorial.AnnotationDim},
					},
				},
				TextAfter: "J♦ is the Left Bower - it's a Heart now!",
			},
			{
				Title:      "Can't Follow Suit",
				TextBefore: "When you have no cards of the led suit:",
				Visual: &tutorial.VisualElement{
					Type: tutorial.VisualCardComparison,
					Cards: []engine.Card{
						{Suit: engine.Hearts, Rank: engine.Nine},
						{Suit: engine.Diamonds, Rank: engine.Nine},
					},
					Trump:      engine.Hearts,
					LeftLabel:  "Trump In",
					RightLabel: "Discard",
					Caption:    "Trump to win, or discard to save trump for later",
				},
				TextAfter: "Strategic choice: Trump if you can win!",
			},
		},
	})

	// Lesson 7: Scoring in Detail
	tutorial.Register(&tutorial.Lesson{
		ID:            "rules-3",
		Title:         "Scoring in Detail",
		Description:   "How points are awarded in Euchre",
		Category:      tutorial.CategoryRules,
		Order:         7,
		Prerequisites: []string{"basics-4"},
		VisualSections: []tutorial.VisualSection{
			{
				Title: "Making Your Bid",
				TextBefore: `When your team calls trump, you need 3+ tricks:

  ●●●○○  3-4 tricks = 1 point  (you "made it")
  ●●●●●  5 tricks   = 2 points (march!)

You called trump, so you're expected to win.`,
			},
			{
				Title: "Getting Euchred",
				TextBefore: `Fail to make 3 tricks? You're EUCHRED!

  Your team:  ●●○○○  (only 2 tricks)
  Opponents:  ●●●○○  (they got 3)

  Result: Opponents score 2 points!

This is a big swing - you expected to score but
instead your opponents score double.`,
			},
			{
				Title:      "Going Alone",
				TextBefore: "Brave enough to play without your partner?",
				Visual: tutorial.NewTableLayoutVisual(
					[4]tutorial.PlayerPosition{
						{Label: "You (Alone)", Style: tutorial.AnnotationHighlight},
						{Label: "Opponent", Style: tutorial.AnnotationDim},
						{Label: "Partner", Style: tutorial.AnnotationDim},
						{Label: "Opponent", Style: tutorial.AnnotationDim},
					},
					"Partner sits out - you play alone!",
				),
				TextAfter: "Win all 5 tricks alone = 4 POINTS!\nRegular make = 1 point, Euchred = opponents get 2",
			},
		},
	})
}
