// Package recipe implements the Recipes module (Phase 5a): a full recipe
// manager with ingredients, steps, tags, search, and optional image upload via
// presigned S3 URLs. Recipes are the composable unit for the Phase 5 meal
// planner — a meal-plan slot references a recipe and the shopping list is
// aggregated from recipe ingredients.
package recipe

import (
	"fmt"
	"sort"
	"strings"
	"time"
)

// Difficulty is the recipe difficulty level.
type Difficulty string

const (
	DifficultyEasy   Difficulty = "easy"
	DifficultyMedium Difficulty = "medium"
	DifficultyHard   Difficulty = "hard"
)

func (d Difficulty) Valid() bool {
	switch d {
	case DifficultyEasy, DifficultyMedium, DifficultyHard:
		return true
	default:
		return false
	}
}

// Ingredient is one line of a recipe's ingredient list.
type Ingredient struct {
	ID        string
	Name      string
	Quantity  string
	Unit      string
	SortOrder int
}

// Step is one numbered instruction.
type Step struct {
	ID          string
	StepNumber  int
	Instruction string
}

// Recipe is the aggregate root. Children (tags, ingredients, steps) are always
// loaded together for a single recipe and replaced wholesale on update.
type Recipe struct {
	ID          string
	Title       string
	Description string
	Difficulty  Difficulty
	PrepMinutes int
	CookMinutes int
	Servings    int
	ImageURL    *string
	SourceURL   *string
	Tags        []string
	Ingredients []Ingredient
	Steps       []Step
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// NewRecipe validates and builds a recipe with the given children.
func NewRecipe(id, title, description string, difficulty Difficulty, prepMinutes, cookMinutes, servings int, imageURL, sourceURL *string, tags []string, ingredients []Ingredient, steps []Step) (*Recipe, error) {
	if id == "" {
		return nil, fmt.Errorf("recipe id cannot be empty")
	}
	if strings.TrimSpace(title) == "" {
		return nil, fmt.Errorf("recipe title cannot be empty")
	}
	if !difficulty.Valid() {
		return nil, fmt.Errorf("invalid difficulty %q", difficulty)
	}
	if prepMinutes < 0 || cookMinutes < 0 {
		return nil, fmt.Errorf("prep/cook minutes cannot be negative")
	}
	if servings < 1 {
		return nil, fmt.Errorf("servings must be at least 1")
	}
	for _, ing := range ingredients {
		if strings.TrimSpace(ing.Name) == "" {
			return nil, fmt.Errorf("ingredient name cannot be empty")
		}
	}
	for _, s := range steps {
		if strings.TrimSpace(s.Instruction) == "" {
			return nil, fmt.Errorf("step instruction cannot be empty")
		}
	}

	return &Recipe{
		ID:          id,
		Title:       title,
		Description: description,
		Difficulty:  difficulty,
		PrepMinutes: prepMinutes,
		CookMinutes: cookMinutes,
		Servings:    servings,
		ImageURL:    imageURL,
		SourceURL:   sourceURL,
		Tags:        normalizedTags(tags),
		Ingredients: ingredients,
		Steps:       steps,
	}, nil
}

// normalizedTags trims, lowercases and dedupes tags.
func normalizedTags(tags []string) []string {
	seen := make(map[string]struct{}, len(tags))
	out := make([]string, 0, len(tags))
	for _, t := range tags {
		t = strings.ToLower(strings.TrimSpace(t))
		if t == "" {
			continue
		}
		if _, ok := seen[t]; ok {
			continue
		}
		seen[t] = struct{}{}
		out = append(out, t)
	}
	sort.Strings(out)
	return out
}

// TotalMinutes returns prep + cook time.
func (r *Recipe) TotalMinutes() int { return r.PrepMinutes + r.CookMinutes }
