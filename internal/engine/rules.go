package engine

// Rules holds the rule configuration consumed by the engine.
//
// NOTE: This struct lives in the engine package (not variants) on purpose.
// The variants package imports engine, so engine must not import variants
// (that would be a circular import). The app layer maps a selected variant
// onto this plain struct.
// Rules invariant for resolving an all-pass round 2:
//
//	For a valid game, exactly ONE of StickTheDealer / AllowMisdeal resolves an
//	all-pass round 2. Under StickTheDealer the dealer cannot pass, so the round
//	never reaches an all-pass (someone must name trump). With StickTheDealer off,
//	AllowMisdeal governs the throw-in. The standard variant keeps the two in sync:
//	AllowMisdeal == !StickTheDealer.
//
// If both are false (a misconfiguration), the engine defensively falls back to a
// misdeal so an all-pass round 2 cannot dead-end bidding (see Round.handlePass).
type Rules struct {
	StickTheDealer   bool // round 2: dealer may not pass; must call trump
	AllowDefendAlone bool // defenders may go alone for 4 points on a euchre
	AllowMisdeal     bool // if all pass round 2 (and not stick-the-dealer), re-deal with SAME dealer, no score
}

// DefaultRules returns the standard rule configuration.
func DefaultRules() Rules {
	return Rules{AllowMisdeal: true}
}
