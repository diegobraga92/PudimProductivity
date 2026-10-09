package mealplan

import (
	"context"
	"fmt"

	"github.com/rs/zerolog/log"

	"github.com/diegobraga92/pudimproductivity/backend/internal/contexts/audit"
	"github.com/diegobraga92/pudimproductivity/backend/internal/platform/eventbus"
	"github.com/diegobraga92/pudimproductivity/backend/pkg/uuid"
)

// CreateInput adds one item across one or more days.
type CreateInput struct {
	Days     []Weekday
	Meal     Meal
	Name     string
	Amount   string
	Calories int
	RecipeID *string
}

// UpdateInput replaces a single item.
type UpdateInput struct {
	Day      Weekday
	Meal     Meal
	Name     string
	Amount   string
	Calories int
	RecipeID *string
}

// MealPlanService coordinates persistence, audit logging and event publication.
type MealPlanService struct {
	repo  Repository
	audit audit.Logger
	bus   eventbus.Bus
}

func NewService(repo Repository, auditLogger audit.Logger, bus eventbus.Bus) *MealPlanService {
	if auditLogger == nil {
		auditLogger = audit.NoopLogger{}
	}
	return &MealPlanService{repo: repo, audit: auditLogger, bus: bus}
}

// Create fans the item out into one independent row per selected day.
func (s *MealPlanService) Create(ctx context.Context, in CreateInput) ([]*Item, error) {
	days := uniqueDays(in.Days)
	if len(days) == 0 {
		return nil, fmt.Errorf("select at least one day")
	}

	items := make([]*Item, 0, len(days))
	for _, day := range days {
		item, err := NewItem(uuid.NewUUID(), day, in.Meal, in.Name, in.Amount, in.Calories, in.RecipeID)
		if err != nil {
			return nil, fmt.Errorf("create item: %w", err)
		}
		items = append(items, item)
	}

	if err := s.repo.CreateBatch(ctx, items); err != nil {
		return nil, fmt.Errorf("persist items: %w", err)
	}

	log.Info().Ctx(ctx).Int("count", len(items)).Str("meal", string(in.Meal)).Msg("meal plan items created")
	s.audit.Log(ctx, audit.ActionMealPlanItemCreated, audit.ResourceMealPlanItems, "", nil, map[string]any{
		"meal":  in.Meal,
		"name":  in.Name,
		"days":  dayStrings(days),
		"count": len(items),
	})
	for _, item := range items {
		s.publish(ctx, eventbus.EventMealPlanItemCreated, toResponse(item))
	}
	return items, nil
}

func (s *MealPlanService) List(ctx context.Context) ([]*Item, error) {
	return s.repo.List(ctx)
}

func (s *MealPlanService) Update(ctx context.Context, id string, in UpdateInput) (*Item, error) {
	item, err := NewItem(id, in.Day, in.Meal, in.Name, in.Amount, in.Calories, in.RecipeID)
	if err != nil {
		return nil, fmt.Errorf("update item: %w", err)
	}

	if err := s.repo.Update(ctx, item); err != nil {
		return nil, err // ErrNotFound passes through
	}

	log.Info().Ctx(ctx).Str("item_id", item.ID).Msg("meal plan item updated")
	s.audit.Log(ctx, audit.ActionMealPlanItemUpdated, audit.ResourceMealPlanItems, item.ID, nil, map[string]any{
		"meal": item.Meal,
		"name": item.Name,
		"day":  item.Day,
	})
	s.publish(ctx, eventbus.EventMealPlanItemUpdated, toResponse(item))
	return item, nil
}

// Clear removes the whole template, publishing one delete event per item.
func (s *MealPlanService) Clear(ctx context.Context) error {
	ids, err := s.repo.DeleteAll(ctx)
	if err != nil {
		return fmt.Errorf("clear meal plan: %w", err)
	}
	if len(ids) == 0 {
		return nil
	}

	log.Info().Ctx(ctx).Int("count", len(ids)).Msg("meal plan cleared")
	s.audit.Log(ctx, audit.ActionMealPlanItemsCleared, audit.ResourceMealPlanItems, "", nil, map[string]any{
		"count": len(ids),
	})
	for _, id := range ids {
		s.publish(ctx, eventbus.EventMealPlanItemDeleted, map[string]any{"id": id})
	}
	return nil
}

func (s *MealPlanService) Delete(ctx context.Context, id string) error {
	if err := s.repo.Delete(ctx, id); err != nil {
		return err // ErrNotFound passes through
	}
	s.audit.Log(ctx, audit.ActionMealPlanItemDeleted, audit.ResourceMealPlanItems, id, nil, nil)
	s.publish(ctx, eventbus.EventMealPlanItemDeleted, map[string]any{"id": id})
	return nil
}

func (s *MealPlanService) publish(ctx context.Context, typ eventbus.EventType, payload interface{}) {
	if s.bus == nil {
		return
	}
	if err := s.bus.Publish(ctx, typ, payload); err != nil {
		log.Warn().Err(err).Str("event_type", string(typ)).Msg("failed to publish meal plan event")
	}
}

func uniqueDays(days []Weekday) []Weekday {
	seen := make(map[Weekday]struct{}, len(days))
	out := make([]Weekday, 0, len(days))
	for _, d := range days {
		if _, ok := seen[d]; ok {
			continue
		}
		seen[d] = struct{}{}
		out = append(out, d)
	}
	return out
}

func dayStrings(days []Weekday) []string {
	out := make([]string, len(days))
	for i, d := range days {
		out[i] = string(d)
	}
	return out
}
