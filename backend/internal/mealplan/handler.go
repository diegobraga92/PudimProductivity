package mealplan

import (
	"context"
	"time"
)

// --- Requests ---

type SlotInput struct {
	Date     string  `json:"date"` // YYYY-MM-DD
	MealType string  `json:"meal_type"`
	RecipeID *string `json:"recipe_id"`
	Notes    string  `json:"notes"`
}

type CreateMealPlanRequest struct {
	Name      string      `json:"name"`
	StartDate string      `json:"start_date"` // YYYY-MM-DD
	EndDate   string      `json:"end_date"`   // YYYY-MM-DD
	Slots     []SlotInput `json:"slots"`
}

func (req CreateMealPlanRequest) toInput() (CreateInput, error) {
	start, err := time.Parse("2006-01-02", req.StartDate)
	if err != nil {
		return CreateInput{}, err
	}
	end, err := time.Parse("2006-01-02", req.EndDate)
	if err != nil {
		return CreateInput{}, err
	}
	slots := make([]MealSlot, 0, len(req.Slots))
	for _, s := range req.Slots {
		d, err := time.Parse("2006-01-02", s.Date)
		if err != nil {
			return CreateInput{}, err
		}
		slots = append(slots, MealSlot{
			Date:     d,
			MealType: MealType(s.MealType),
			RecipeID: s.RecipeID,
			Notes:    s.Notes,
		})
	}
	return CreateInput{Name: req.Name, StartDate: start, EndDate: end, Slots: slots}, nil
}

type AssignSlotRequest struct {
	RecipeID *string `json:"recipe_id"`
	Notes    *string `json:"notes"`
}

// --- Responses ---

type SlotResponse struct {
	ID       string  `json:"id"`
	Date     string  `json:"date"`
	MealType string  `json:"meal_type"`
	RecipeID *string `json:"recipe_id"`
	Notes    string  `json:"notes,omitempty"`
}

type MealPlanResponse struct {
	ID          string         `json:"id"`
	Name        string         `json:"name"`
	StartDate   string         `json:"start_date"`
	EndDate     string         `json:"end_date"`
	IsPublished bool           `json:"is_published"`
	Slots       []SlotResponse `json:"slots,omitempty"`
	CreatedAt   string         `json:"created_at"`
	UpdatedAt   string         `json:"updated_at"`
}

func toResponse(p *MealPlan) MealPlanResponse {
	resp := MealPlanResponse{
		ID:          p.ID,
		Name:        p.Name,
		StartDate:   p.StartDate.Format("2006-01-02"),
		EndDate:     p.EndDate.Format("2006-01-02"),
		IsPublished: p.IsPublished,
		CreatedAt:   p.CreatedAt.Format(time.RFC3339),
		UpdatedAt:   p.UpdatedAt.Format(time.RFC3339),
	}
	for _, s := range p.Slots {
		resp.Slots = append(resp.Slots, SlotResponse{
			ID:       s.ID,
			Date:     s.Date.Format("2006-01-02"),
			MealType: string(s.MealType),
			RecipeID: s.RecipeID,
			Notes:    s.Notes,
		})
	}
	return resp
}

func toResponses(plans []*MealPlan) []MealPlanResponse {
	out := make([]MealPlanResponse, 0, len(plans))
	for _, p := range plans {
		out = append(out, toResponse(p))
	}
	return out
}

type ShoppingItemResponse struct {
	ID             string `json:"id"`
	IngredientName string `json:"ingredient_name"`
	QuantityAgg    string `json:"quantity_agg,omitempty"`
	Unit           string `json:"unit,omitempty"`
	IsChecked      bool   `json:"is_checked"`
}

func toShoppingResponses(items []ShoppingItem) []ShoppingItemResponse {
	out := make([]ShoppingItemResponse, 0, len(items))
	for _, item := range items {
		out = append(out, ShoppingItemResponse(item))
	}
	return out
}

// --- Service interface (consumer-side, handler level) ---

type Service interface {
	Create(ctx context.Context, in CreateInput) (*MealPlan, error)
	Get(ctx context.Context, id string) (*MealPlan, error)
	List(ctx context.Context) ([]*MealPlan, error)
	Update(ctx context.Context, id string, in UpdateInput) (*MealPlan, error)
	Delete(ctx context.Context, id string) error
	AssignSlot(ctx context.Context, planID, slotID string, recipeID *string, notes *string) error
	GenerateShoppingList(ctx context.Context, planID string) ([]ShoppingItem, error)
	GetShoppingList(ctx context.Context, planID string) ([]ShoppingItem, error)
	ToggleShoppingItem(ctx context.Context, planID, itemID string) error
	Publish(ctx context.Context, planID string) error
	// RenderPDF generates the printable meal-plan PDF (Phase 9b).
	RenderPDF(ctx context.Context, id string) ([]byte, error)
}
