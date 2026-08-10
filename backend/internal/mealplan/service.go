package mealplan

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/rs/zerolog/log"

	"github.com/diegobraga92/pudimproductivity/backend/internal/audit"
	"github.com/diegobraga92/pudimproductivity/backend/internal/eventbus"
	"github.com/diegobraga92/pudimproductivity/backend/internal/shared"
)

// CreateInput is the full meal-plan payload (slots included).
type CreateInput struct {
	Name      string
	StartDate time.Time
	EndDate   time.Time
	Slots     []MealSlot
}

// UpdateInput reuses CreateInput (full replacement of slots).
type UpdateInput = CreateInput

// Service coordinates the meal-planning module.
type MealPlanService struct {
	repo    Repository
	recipes RecipeReader // nil = recipes module not wired (shopping list disabled)
	audit   audit.Logger
	bus     eventbus.Bus
}

func NewMealPlanService(repo Repository, recipes RecipeReader, auditLogger audit.Logger, bus eventbus.Bus) *MealPlanService {
	if auditLogger == nil {
		auditLogger = audit.NoopLogger{}
	}
	return &MealPlanService{repo: repo, recipes: recipes, audit: auditLogger, bus: bus}
}

func (s *MealPlanService) Create(ctx context.Context, in CreateInput) (*MealPlan, error) {
	id := shared.NewUUID()
	plan, err := NewMealPlan(id, in.Name, in.StartDate, in.EndDate, false, assignSlotIDs(in.Slots))
	if err != nil {
		return nil, fmt.Errorf("create meal plan: %w", err)
	}
	if err := s.repo.Create(ctx, plan); err != nil {
		return nil, fmt.Errorf("persist meal plan: %w", err)
	}
	s.audit.Log(ctx, audit.ActionMealPlanCreated, audit.ResourceMealPlans, plan.ID, nil, map[string]any{"name": plan.Name})
	s.publish(ctx, eventbus.EventMealPlanCreated, map[string]any{"id": plan.ID})
	return plan, nil
}

func (s *MealPlanService) Get(ctx context.Context, id string) (*MealPlan, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *MealPlanService) List(ctx context.Context) ([]*MealPlan, error) {
	return s.repo.List(ctx)
}

func (s *MealPlanService) Update(ctx context.Context, id string, in UpdateInput) (*MealPlan, error) {
	plan, err := NewMealPlan(id, in.Name, in.StartDate, in.EndDate, false, assignSlotIDs(in.Slots))
	if err != nil {
		return nil, fmt.Errorf("update meal plan: %w", err)
	}
	if err := s.repo.Update(ctx, plan); err != nil {
		return nil, err
	}
	s.audit.Log(ctx, audit.ActionMealPlanUpdated, audit.ResourceMealPlans, id, nil, map[string]any{"name": plan.Name})
	return plan, nil
}

func (s *MealPlanService) Delete(ctx context.Context, id string) error {
	if err := s.repo.Delete(ctx, id); err != nil {
		return err
	}
	s.audit.Log(ctx, audit.ActionMealPlanDeleted, audit.ResourceMealPlans, id, nil, nil)
	return nil
}

func (s *MealPlanService) AssignSlot(ctx context.Context, planID, slotID string, recipeID *string, notes *string) error {
	return s.repo.UpdateSlot(ctx, planID, slotID, recipeID, notes)
}

// GenerateShoppingList aggregates the ingredients of every slotted recipe into
// a single list, grouping by (name, unit) and summing free-text quantities
// when they are plain numbers. Replaces the plan's existing list.
func (s *MealPlanService) GenerateShoppingList(ctx context.Context, planID string) ([]ShoppingItem, error) {
	if s.recipes == nil {
		return nil, fmt.Errorf("recipes module not configured — cannot generate shopping list")
	}
	plan, err := s.repo.GetByID(ctx, planID)
	if err != nil {
		return nil, err
	}

	// Load each distinct recipe once.
	type recipeIngredients struct {
		name   string
		qty    string
		unit   string
	}
	var all []recipeIngredients
	seen := map[string]struct{}{}
	for _, slot := range plan.Slots {
		if slot.RecipeID == nil {
			continue
		}
		if _, ok := seen[*slot.RecipeID]; ok {
			continue
		}
		seen[*slot.RecipeID] = struct{}{}
		recipe, err := s.recipes.Get(ctx, *slot.RecipeID)
		if err != nil {
			log.Warn().Err(err).Str("recipe_id", *slot.RecipeID).Msg("shopping list: recipe load failed, skipping")
			continue
		}
		for _, ing := range recipe.Ingredients {
			all = append(all, recipeIngredients{name: strings.ToLower(ing.Name), qty: ing.Quantity, unit: ing.Unit})
		}
	}

	// Group by (name, unit).
	type bucket struct {
		name string
		qty  string
		unit string
	}
	merged := map[string]*bucket{}
	var order []string
	for _, ing := range all {
		key := ing.name + "\x00" + ing.unit
		if b, ok := merged[key]; ok {
			b.qty = mergeQuantities(b.qty, ing.qty)
		} else {
			merged[key] = &bucket{name: ing.name, qty: ing.qty, unit: ing.unit}
			order = append(order, key)
		}
	}

	items := make([]ShoppingItem, 0, len(order))
	for _, key := range order {
		b := merged[key]
		items = append(items, ShoppingItem{ID: shared.NewUUID(), IngredientName: b.name, QuantityAgg: b.qty, Unit: b.unit})
	}
	if err := s.repo.ReplaceShoppingList(ctx, planID, items); err != nil {
		return nil, err
	}
	return items, nil
}

func (s *MealPlanService) GetShoppingList(ctx context.Context, planID string) ([]ShoppingItem, error) {
	return s.repo.GetShoppingList(ctx, planID)
}

func (s *MealPlanService) ToggleShoppingItem(ctx context.Context, planID, itemID string) error {
	return s.repo.ToggleShoppingItem(ctx, planID, itemID)
}

// Publish marks a plan published and emits the mealplan.published event.
func (s *MealPlanService) Publish(ctx context.Context, planID string) error {
	if err := s.repo.SetPublished(ctx, planID); err != nil {
		return err
	}
	s.publish(ctx, eventbus.EventMealPlanPublished, map[string]any{"id": planID})
	return nil
}

func (s *MealPlanService) publish(ctx context.Context, typ eventbus.EventType, payload interface{}) {
	if s.bus == nil {
		return
	}
	if err := s.bus.Publish(ctx, typ, payload); err != nil {
		log.Warn().Err(err).Str("event_type", string(typ)).Msg("failed to publish mealplan event")
	}
}

func assignSlotIDs(slots []MealSlot) []MealSlot {
	out := make([]MealSlot, len(slots))
	for i, s := range slots {
		s.ID = shared.NewUUID()
		out[i] = s
	}
	return out
}
