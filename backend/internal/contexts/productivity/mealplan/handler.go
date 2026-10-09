package mealplan

import (
	"context"
	"time"
)

type CreateItemRequest struct {
	Days     []string `json:"days"`
	Meal     string   `json:"meal"`
	Name     string   `json:"name"`
	Amount   string   `json:"amount"`
	Calories int      `json:"calories"`
	RecipeID *string  `json:"recipe_id"`
}

func (in CreateItemRequest) toInput() CreateInput {
	days := make([]Weekday, 0, len(in.Days))
	for _, d := range in.Days {
		days = append(days, Weekday(d))
	}
	return CreateInput{
		Days:     days,
		Meal:     Meal(in.Meal),
		Name:     in.Name,
		Amount:   in.Amount,
		Calories: in.Calories,
		RecipeID: in.RecipeID,
	}
}

type UpdateItemRequest struct {
	Day      string  `json:"day"`
	Meal     string  `json:"meal"`
	Name     string  `json:"name"`
	Amount   string  `json:"amount"`
	Calories int     `json:"calories"`
	RecipeID *string `json:"recipe_id"`
}

func (in UpdateItemRequest) toInput() UpdateInput {
	return UpdateInput{
		Day:      Weekday(in.Day),
		Meal:     Meal(in.Meal),
		Name:     in.Name,
		Amount:   in.Amount,
		Calories: in.Calories,
		RecipeID: in.RecipeID,
	}
}

type ItemResponse struct {
	ID        string  `json:"id"`
	Day       string  `json:"day"`
	Meal      string  `json:"meal"`
	Name      string  `json:"name"`
	Amount    string  `json:"amount"`
	Calories  int     `json:"calories"`
	RecipeID  *string `json:"recipe_id,omitempty"`
	CreatedAt string  `json:"created_at"`
	UpdatedAt string  `json:"updated_at"`
}

func toResponse(i *Item) ItemResponse {
	return ItemResponse{
		ID:        i.ID,
		Day:       string(i.Day),
		Meal:      string(i.Meal),
		Name:      i.Name,
		Amount:    i.Amount,
		Calories:  i.Calories,
		RecipeID:  i.RecipeID,
		CreatedAt: i.CreatedAt.Format(time.RFC3339),
		UpdatedAt: i.UpdatedAt.Format(time.RFC3339),
	}
}

func toResponses(items []*Item) []ItemResponse {
	out := make([]ItemResponse, 0, len(items))
	for _, i := range items {
		out = append(out, toResponse(i))
	}
	return out
}

type Service interface {
	Create(ctx context.Context, in CreateInput) ([]*Item, error)
	List(ctx context.Context) ([]*Item, error)
	Update(ctx context.Context, id string, in UpdateInput) (*Item, error)
	Delete(ctx context.Context, id string) error
}
