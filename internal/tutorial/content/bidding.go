package content

import (
	"github.com/BrandonDedolph/euchre/internal/engine"
	"github.com/BrandonDedolph/euchre/internal/tutorial"
)

func init() {
	registerBiddingLessons()
}

func registerBiddingLessons() {
	// Lesson: Bidding for Trump
	//
	// Sits between the bower/hierarchy lesson (rules-1, Order 5) and the
	// "Following Suit" lesson (rules-2, Order 6). It teaches the two-round
	// bidding process, the dealer pickup/discard, and the round-2 restriction.
	tutorial.Register(&tutorial.Lesson{
		ID:            "rules-bidding",
		Title:         "Bidding for Trump",
		Description:   "How trump is chosen: ordering up, calling, and the dealer's pickup",
		Category:      tutorial.CategoryBidding,
		Order:         6, // After "The Bower System" (5), before "Following Suit" (now 7)
		Prerequisites: []string{"rules-1"},
		VisualSections: []tutorial.VisualSection{
			{
				Title:      "The Turned-Up Card",
				TextBefore: "After the deal, the top card of the kitty is\nturned face-up. Its suit is the candidate for trump.",
				Visual: tutorial.NewSingleCardVisual(
					engine.Card{Suit: engine.Hearts, Rank: engine.Ten},
					"Turned up",
					engine.Hearts,
				),
				TextAfter: "Here the 10♥ is up, so Hearts is on offer as trump.",
			},
			{
				Title:      "Round 1: Order It Up",
				TextBefore: "Starting LEFT of the dealer and going clockwise,\neach player either orders it up or passes.",
				Visual: tutorial.NewTableLayoutVisual(
					[4]tutorial.PlayerPosition{
						{Label: "Dealer", Style: tutorial.AnnotationDim},
						{Label: "1st (left)", Style: tutorial.AnnotationHighlight},
						{Label: "2nd", Style: tutorial.AnnotationDim},
						{Label: "3rd", Style: tutorial.AnnotationDim},
					},
					"Bidding starts left of the dealer, clockwise",
				),
				TextAfter: "\"Order it up\" makes the turned suit trump\nfor the player's team.",
			},
			{
				Title:      "Dealer Picks Up",
				TextBefore: "When the card is ordered up, the DEALER takes it\ninto their hand, then discards one card.",
				Visual: tutorial.NewCardComparisonVisual(
					[]engine.Card{
						{Suit: engine.Hearts, Rank: engine.Ten},
					},
					[]engine.Card{
						{Suit: engine.Spades, Rank: engine.Nine},
					},
					"Pick up",
					"Discard",
					engine.Hearts,
				),
				TextAfter: "The dealer is back to 5 cards, now holding the 10♥.",
			},
			{
				Title:      "Round 2: Name a Suit",
				TextBefore: "If everyone passes, the card is turned down.\nNow each player may name a DIFFERENT suit—or pass.",
				Visual: &tutorial.VisualElement{
					Type: tutorial.VisualCardComparison,
					Cards: []engine.Card{
						{Suit: engine.Hearts, Rank: engine.Ten}, // turned-down suit
						{Suit: engine.Spades, Rank: engine.Ace}, // a legal call
					},
					LeftLabel:  "Turned down",
					RightLabel: "May call",
					Caption:    "Hearts was turned down—you may NOT call Hearts",
					Annotations: []tutorial.CardAnnotation{
						{CardIndex: 0, Label: "Illegal", Style: tutorial.AnnotationLoser},
						{CardIndex: 1, Label: "Legal", Style: tutorial.AnnotationWinner},
					},
				},
				TextAfter: "You can call Spades, Clubs, or Diamonds—\nany suit except the turned-down Hearts.",
			},
			{
				Title: "Everyone Passes Again",
				TextBefore: `If all four players pass in round 2 as well,
the hand is a misdeal: the cards are thrown in
and re-dealt by the same dealer.`,
				TextAfter: `Optional "stick the dealer" rule: the dealer is
not allowed to pass in round 2, so they must
name a suit—there is never a misdeal.`,
			},
		},
	})
}
