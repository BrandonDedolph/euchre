package tutorial

import "sort"

// Category represents a lesson category
type Category int

const (
	CategoryBasics Category = iota
	CategoryRules
	CategoryBidding
	CategoryPlay
	CategoryAdvanced
)

func (c Category) String() string {
	switch c {
	case CategoryBasics:
		return "Basics"
	case CategoryRules:
		return "Rules"
	case CategoryBidding:
		return "Bidding Strategy"
	case CategoryPlay:
		return "Play Strategy"
	case CategoryAdvanced:
		return "Advanced"
	default:
		return "Unknown"
	}
}

// Lesson represents a single tutorial lesson
type Lesson struct {
	ID            string
	Title         string
	Description   string
	Category      Category
	Order         int
	Prerequisites []string // Lesson IDs that must be completed first
	Sections      []Section
	VisualSections []VisualSection // Visual demonstration sections (optional, takes precedence)
}

// HasVisuals returns true if this lesson uses visual sections
func (l *Lesson) HasVisuals() bool {
	return len(l.VisualSections) > 0
}

// SectionCount returns the number of sections in the lesson
func (l *Lesson) SectionCount() int {
	if l.HasVisuals() {
		return len(l.VisualSections)
	}
	return len(l.Sections)
}

// GetVisualSection returns a visual section by index, or nil if not available
func (l *Lesson) GetVisualSection(index int) *VisualSection {
	if !l.HasVisuals() || index < 0 || index >= len(l.VisualSections) {
		return nil
	}
	return &l.VisualSections[index]
}

// GetSection returns a text section by index, or nil if not available
func (l *Lesson) GetSection(index int) *Section {
	if l.HasVisuals() || index < 0 || index >= len(l.Sections) {
		return nil
	}
	return &l.Sections[index]
}

// Section represents a part of a lesson
type Section struct {
	Type    SectionType
	Title   string
	Content string // Markdown-compatible text
}

// SectionType represents the type of section
type SectionType int

const (
	SectionText SectionType = iota
	SectionExample
	SectionQuiz
	SectionInteractive
	SectionVisual // Visual demonstration section
)

// LessonRegistry holds all lessons
type LessonRegistry struct {
	lessons map[string]*Lesson
	order   []string // Ordered list of lesson IDs
}

// NewLessonRegistry creates a new lesson registry
func NewLessonRegistry() *LessonRegistry {
	return &LessonRegistry{
		lessons: make(map[string]*Lesson),
		order:   make([]string, 0),
	}
}

// Register adds a lesson to the registry
func (r *LessonRegistry) Register(lesson *Lesson) {
	r.lessons[lesson.ID] = lesson
	r.order = append(r.order, lesson.ID)
}

// Get retrieves a lesson by ID
func (r *LessonRegistry) Get(id string) (*Lesson, bool) {
	lesson, ok := r.lessons[id]
	return lesson, ok
}

// List returns all lessons in order
func (r *LessonRegistry) List() []*Lesson {
	lessons := make([]*Lesson, 0, len(r.order))
	for _, id := range r.order {
		if lesson, ok := r.lessons[id]; ok {
			lessons = append(lessons, lesson)
		}
	}
	return lessons
}

// ByCategory returns lessons filtered by category
func (r *LessonRegistry) ByCategory(category Category) []*Lesson {
	lessons := make([]*Lesson, 0)
	for _, id := range r.order {
		if lesson, ok := r.lessons[id]; ok {
			if lesson.Category == category {
				lessons = append(lessons, lesson)
			}
		}
	}
	return lessons
}

// AllInOrder returns all lessons sorted by their Order field
func (r *LessonRegistry) AllInOrder() []*Lesson {
	lessons := r.List()
	sort.Slice(lessons, func(i, j int) bool {
		return lessons[i].Order < lessons[j].Order
	})
	return lessons
}

// DefaultRegistry is the global lesson registry
var DefaultRegistry = NewLessonRegistry()

// Register adds a lesson to the default registry
func Register(lesson *Lesson) {
	DefaultRegistry.Register(lesson)
}

// Get retrieves a lesson from the default registry
func Get(id string) (*Lesson, bool) {
	return DefaultRegistry.Get(id)
}

// List returns all lessons from the default registry
func List() []*Lesson {
	return DefaultRegistry.List()
}

// ByCategory returns lessons by category from the default registry
func ByCategory(category Category) []*Lesson {
	return DefaultRegistry.ByCategory(category)
}

// AllInOrder returns all lessons sorted by Order from the default registry
func AllInOrder() []*Lesson {
	return DefaultRegistry.AllInOrder()
}
