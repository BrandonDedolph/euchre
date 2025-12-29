package variants

import "github.com/bran/euchre/internal/engine"

// Variant defines the rules and configuration for a Euchre variant
type Variant interface {
	// Identification
	Name() string
	Description() string

	// Player configuration
	PlayerCount() int
	TeamCount() int
	CardsPerHand() int
	TargetScore() int

	// Deck configuration
	CreateDeck() *engine.Deck
	HasJoker() bool

	// Trump rules
	TrumpHierarchy(trump engine.Suit) []engine.Card
	IsLeftBower(card engine.Card, trump engine.Suit) bool

	// Bidding rules
	BiddingRounds() int
	CanGoAlone() bool
	HasStickTheDealer() bool

	// Scoring rules
	ScoreRound(result engine.RoundResult) engine.ScoreUpdate

	// Special rules
	HasFarmersHand() bool
	AllowMisdeal() bool

	// Rule options (configurable per-variant)
	Options() []RuleOption
	SetOption(key string, value interface{}) error
	GetOption(key string) interface{}
}

// RuleOption represents a configurable rule setting
type RuleOption struct {
	Key         string
	Name        string
	Description string
	Type        OptionType
	Default     interface{}
	Choices     []interface{} // For choice types
}

// OptionType represents the type of a rule option
type OptionType int

const (
	OptionBool OptionType = iota
	OptionInt
	OptionChoice
)

// BaseVariant provides common functionality for variants
type BaseVariant struct {
	options map[string]interface{}
}

// NewBaseVariant creates a new base variant
func NewBaseVariant() BaseVariant {
	return BaseVariant{
		options: make(map[string]interface{}),
	}
}

// SetOption sets a rule option value
func (v *BaseVariant) SetOption(key string, value interface{}) error {
	v.options[key] = value
	return nil
}

// GetOption gets a rule option value
func (v *BaseVariant) GetOption(key string) interface{} {
	return v.options[key]
}

// GetBoolOption gets a boolean option with a default
func (v *BaseVariant) GetBoolOption(key string, defaultVal bool) bool {
	if val, ok := v.options[key].(bool); ok {
		return val
	}
	return defaultVal
}

// GetIntOption gets an integer option with a default
func (v *BaseVariant) GetIntOption(key string, defaultVal int) int {
	if val, ok := v.options[key].(int); ok {
		return val
	}
	return defaultVal
}

// Registry holds all registered variants
type Registry struct {
	variants map[string]Variant
}

// NewRegistry creates a new variant registry
func NewRegistry() *Registry {
	return &Registry{
		variants: make(map[string]Variant),
	}
}

// Register adds a variant to the registry
func (r *Registry) Register(v Variant) {
	r.variants[v.Name()] = v
}

// Get retrieves a variant by name
func (r *Registry) Get(name string) (Variant, bool) {
	v, ok := r.variants[name]
	return v, ok
}

// List returns all registered variant names
func (r *Registry) List() []string {
	names := make([]string, 0, len(r.variants))
	for name := range r.variants {
		names = append(names, name)
	}
	return names
}

// All returns all registered variants
func (r *Registry) All() []Variant {
	variants := make([]Variant, 0, len(r.variants))
	for _, v := range r.variants {
		variants = append(variants, v)
	}
	return variants
}

// DefaultRegistry is the global variant registry
var DefaultRegistry = NewRegistry()

// Register adds a variant to the default registry
func Register(v Variant) {
	DefaultRegistry.Register(v)
}

// Get retrieves a variant from the default registry
func Get(name string) (Variant, bool) {
	return DefaultRegistry.Get(name)
}

// List returns all variant names from the default registry
func List() []string {
	return DefaultRegistry.List()
}
