package app

import "testing"

func TestTeachableFiresOnceForGoingAlone(t *testing.T) {
	g := NewGamePlayWithSettings(GameSettings{Variant: "Standard", Tutorial: true})

	// No alone declared yet → nothing to teach.
	g.maybeShowTeachable()
	if g.pendingPopup != nil {
		t.Fatalf("popup queued with no trigger: %v", g.pendingPopup.id)
	}

	// Someone goes alone → the going-alone concept fires.
	g.tableView.MakerAlone = true
	g.maybeShowTeachable()
	if g.pendingPopup == nil || g.pendingPopup.id != "going_alone" {
		t.Fatalf("expected going_alone popup, got %v", g.pendingPopup)
	}

	// Dismissing chains to the next concept; there is none, so it clears and
	// the same concept must not fire again.
	g.dismissPopup()
	if g.pendingPopup != nil {
		t.Fatalf("going_alone re-fired after dismissal: %v", g.pendingPopup.id)
	}
}

func TestNonTutorialNeverPops(t *testing.T) {
	g := NewGamePlay() // tutorial = false
	g.tableView.MakerAlone = true
	g.maybeShowTeachable()
	if g.pendingPopup != nil {
		t.Fatalf("non-tutorial game queued a popup: %v", g.pendingPopup.id)
	}
}
