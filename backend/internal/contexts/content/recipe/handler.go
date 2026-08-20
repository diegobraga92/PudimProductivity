package recipe

import (
	"context"
	"time"

	"github.com/diegobraga92/pudimproductivity/backend/internal/contexts/content/media"
)

// --- Requests ---

type IngredientInput struct {
	Name     string `json:"name"`
	Quantity string `json:"quantity"`
	Unit     string `json:"unit"`
}

type StepInput struct {
	Instruction string `json:"instruction"`
}

type CreateRecipeRequest struct {
	Title       string            `json:"title"`
	Description string            `json:"description"`
	Difficulty  string            `json:"difficulty"`
	PrepMinutes int               `json:"prep_time_minutes"`
	CookMinutes int               `json:"cook_time_minutes"`
	Servings    int               `json:"servings"`
	ImageURL    *string           `json:"image_url"`
	SourceURL   *string           `json:"source_url"`
	Tags        []string          `json:"tags"`
	Ingredients []IngredientInput `json:"ingredients"`
	Steps       []StepInput       `json:"steps"`
}

func (in CreateRecipeRequest) toInput() CreateInput {
	ingredients := make([]Ingredient, 0, len(in.Ingredients))
	for _, ing := range in.Ingredients {
		ingredients = append(ingredients, Ingredient{Name: ing.Name, Quantity: ing.Quantity, Unit: ing.Unit})
	}
	steps := make([]Step, 0, len(in.Steps))
	for _, s := range in.Steps {
		steps = append(steps, Step{Instruction: s.Instruction})
	}
	return CreateInput{
		Title:       in.Title,
		Description: in.Description,
		Difficulty:  Difficulty(in.Difficulty),
		PrepMinutes: in.PrepMinutes,
		CookMinutes: in.CookMinutes,
		Servings:    in.Servings,
		ImageURL:    in.ImageURL,
		SourceURL:   in.SourceURL,
		Tags:        in.Tags,
		Ingredients: ingredients,
		Steps:       steps,
	}
}

type UploadURLRequest struct {
	ContentType string `json:"content_type"`
	Filename    string `json:"filename"`
}

// --- Responses ---

type IngredientResponse struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Quantity  string `json:"quantity,omitempty"`
	Unit      string `json:"unit,omitempty"`
	SortOrder int    `json:"sort_order"`
}

type StepResponse struct {
	ID          string `json:"id"`
	StepNumber  int    `json:"step_number"`
	Instruction string `json:"instruction"`
}

type RecipeResponse struct {
	ID          string               `json:"id"`
	Title       string               `json:"title"`
	Description string               `json:"description"`
	Difficulty  string               `json:"difficulty"`
	PrepMinutes int                  `json:"prep_time_minutes"`
	CookMinutes int                  `json:"cook_time_minutes"`
	Servings    int                  `json:"servings"`
	ImageURL    *string              `json:"image_url,omitempty"`
	SourceURL   *string              `json:"source_url,omitempty"`
	Tags        []string             `json:"tags,omitempty"`
	Ingredients []IngredientResponse `json:"ingredients,omitempty"`
	Steps       []StepResponse       `json:"steps,omitempty"`
	CreatedAt   string               `json:"created_at"`
	UpdatedAt   string               `json:"updated_at"`
}

func toResponse(r *Recipe) RecipeResponse {
	resp := RecipeResponse{
		ID:          r.ID,
		Title:       r.Title,
		Description: r.Description,
		Difficulty:  string(r.Difficulty),
		PrepMinutes: r.PrepMinutes,
		CookMinutes: r.CookMinutes,
		Servings:    r.Servings,
		ImageURL:    r.ImageURL,
		SourceURL:   r.SourceURL,
		Tags:        r.Tags,
		CreatedAt:   r.CreatedAt.Format(time.RFC3339),
		UpdatedAt:   r.UpdatedAt.Format(time.RFC3339),
	}
	for _, ing := range r.Ingredients {
		resp.Ingredients = append(resp.Ingredients, IngredientResponse(ing))
	}
	for _, s := range r.Steps {
		resp.Steps = append(resp.Steps, StepResponse(s))
	}
	return resp
}

func toResponses(recipes []*Recipe) []RecipeResponse {
	out := make([]RecipeResponse, 0, len(recipes))
	for _, r := range recipes {
		out = append(out, toResponse(r))
	}
	return out
}

// --- Service interface (consumer-side, handler level) ---

type Service interface {
	Create(ctx context.Context, in CreateInput) (*Recipe, error)
	Get(ctx context.Context, id string) (*Recipe, error)
	List(ctx context.Context, filter ListFilter) ([]*Recipe, error)
	Update(ctx context.Context, id string, in UpdateInput) (*Recipe, error)
	Delete(ctx context.Context, id string) error
	GenerateUploadURL(ctx context.Context, recipeID, contentType, filename string) (*media.UploadURL, error)
}
