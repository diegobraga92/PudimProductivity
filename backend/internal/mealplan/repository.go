package mealplan

import (
	"context"
	"errors"

	"github.com/diegobraga92/pudimproductivity/backend/internal/recipe"
)

// ErrNotFound is returned when a meal plan does not exist.
var ErrNotFound = errors.New("meal plan not found")

// RecipeReader is the minimal slice of the recipes module the meal planner
// needs to generate shopping lists. Satisfied by *recipe.RecipeService.
type RecipeReader interface {
	Get(ctx context.Context, id string) (*recipe.Recipe, error)
}

// Repository persists meal plans, their slots, and the shopping list.
type Repository interface {
	Create(ctx context.Context, plan *MealPlan) error
	GetByID(ctx context.Context, id string) (*MealPlan, error)
	List(ctx context.Context) ([]*MealPlan, error)
	Update(ctx context.Context, plan *MealPlan) error
	Delete(ctx context.Context, id string) error
	// UpdateSlot assigns a recipe and notes to a slot (recipeID nil clears).
	UpdateSlot(ctx context.Context, planID, slotID string, recipeID *string, notes *string) error
	// ReplaceShoppingList deletes the existing list for a plan and inserts the
	// given items.
	ReplaceShoppingList(ctx context.Context, planID string, items []ShoppingItem) error
	GetShoppingList(ctx context.Context, planID string) ([]ShoppingItem, error)
	// ToggleShoppingItem flips the checked state of a list item.
	ToggleShoppingItem(ctx context.Context, planID, itemID string) error
	// SetPublished marks a plan published.
	SetPublished(ctx context.Context, id string) error
}
