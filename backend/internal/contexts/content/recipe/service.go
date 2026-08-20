package recipe

import (
	"context"
	"fmt"
	"time"

	"github.com/rs/zerolog/log"

	"github.com/diegobraga92/pudimproductivity/backend/internal/contexts/audit"
	"github.com/diegobraga92/pudimproductivity/backend/internal/contexts/content/media"
	"github.com/diegobraga92/pudimproductivity/backend/internal/platform/eventbus"
	"github.com/diegobraga92/pudimproductivity/backend/pkg/uuid"
)

// CreateInput is the full set of fields for creating a recipe. Children
// (ingredients, steps) are provided as ordered lists; IDs are assigned by the
// service.
type CreateInput struct {
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
}

// UpdateInput uses full-replacement semantics: all fields are required and the
// existing children are replaced wholesale.
type UpdateInput = CreateInput

// Service coordinates the recipe module: persistence, audit logging, event
// publication, and optional media uploads.
type RecipeService struct {
	repo    Repository
	audit   audit.Logger
	bus     eventbus.Bus
	uploads media.Generator // nil when media is not configured (degraded mode)
}

func NewService(repo Repository, auditLogger audit.Logger, bus eventbus.Bus, uploads media.Generator) *RecipeService {
	if auditLogger == nil {
		auditLogger = audit.NoopLogger{}
	}
	return &RecipeService{repo: repo, audit: auditLogger, bus: bus, uploads: uploads}
}

func (s *RecipeService) Create(ctx context.Context, in CreateInput) (*Recipe, error) {
	id := uuid.NewUUID()
	recipe, err := NewRecipe(
		id, in.Title, in.Description, in.Difficulty,
		in.PrepMinutes, in.CookMinutes, in.Servings,
		in.ImageURL, in.SourceURL, in.Tags,
		assignIngredientIDs(in.Ingredients), assignStepIDs(in.Steps),
	)
	if err != nil {
		return nil, fmt.Errorf("create recipe: %w", err)
	}

	if err := s.repo.Create(ctx, recipe); err != nil {
		return nil, fmt.Errorf("persist recipe: %w", err)
	}

	log.Info().Ctx(ctx).Str("recipe_id", recipe.ID).Str("title", recipe.Title).Msg("recipe created")
	s.audit.Log(ctx, audit.ActionRecipeCreated, audit.ResourceRecipes, recipe.ID, nil, map[string]any{
		"title":      recipe.Title,
		"difficulty": recipe.Difficulty,
	})
	s.publish(ctx, eventbus.EventRecipeCreated, toResponse(recipe))
	return recipe, nil
}

func (s *RecipeService) Get(ctx context.Context, id string) (*Recipe, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *RecipeService) List(ctx context.Context, filter ListFilter) ([]*Recipe, error) {
	return s.repo.List(ctx, filter)
}

func (s *RecipeService) Update(ctx context.Context, id string, in UpdateInput) (*Recipe, error) {
	recipe, err := NewRecipe(
		id, in.Title, in.Description, in.Difficulty,
		in.PrepMinutes, in.CookMinutes, in.Servings,
		in.ImageURL, in.SourceURL, in.Tags,
		assignIngredientIDs(in.Ingredients), assignStepIDs(in.Steps),
	)
	if err != nil {
		return nil, fmt.Errorf("update recipe: %w", err)
	}

	if err := s.repo.Update(ctx, recipe); err != nil {
		return nil, err // ErrNotFound passes through
	}

	log.Info().Ctx(ctx).Str("recipe_id", recipe.ID).Msg("recipe updated")
	s.audit.Log(ctx, audit.ActionRecipeUpdated, audit.ResourceRecipes, recipe.ID, nil, map[string]any{
		"title": recipe.Title,
	})
	s.publish(ctx, eventbus.EventRecipeUpdated, toResponse(recipe))
	return recipe, nil
}

func (s *RecipeService) Delete(ctx context.Context, id string) error {
	if err := s.repo.Delete(ctx, id); err != nil {
		return err // ErrNotFound passes through
	}
	s.audit.Log(ctx, audit.ActionRecipeDeleted, audit.ResourceRecipes, id, nil, nil)
	s.publish(ctx, eventbus.EventRecipeDeleted, map[string]any{"id": id})
	return nil
}

// GenerateUploadURL returns a presigned PUT URL the client uploads the image to
// directly, plus the object key that should be stored as the recipe's
// image_url. Errors when media is not configured.
func (s *RecipeService) GenerateUploadURL(ctx context.Context, recipeID, contentType, filename string) (*media.UploadURL, error) {
	if s.uploads == nil {
		return nil, fmt.Errorf("media uploads are not configured")
	}
	if _, err := s.repo.GetByID(ctx, recipeID); err != nil {
		return nil, err // ErrNotFound passes through
	}
	key := uuid.NewUUID() + "/" + sanitizeFilename(filename)
	ttl := 15 * time.Minute
	return s.uploads.GenerateUploadURL(ctx, key, contentType, ttl)
}

func (s *RecipeService) publish(ctx context.Context, typ eventbus.EventType, payload interface{}) {
	if s.bus == nil {
		return
	}
	if err := s.bus.Publish(ctx, typ, payload); err != nil {
		log.Warn().Err(err).Str("event_type", string(typ)).Msg("failed to publish recipe event")
	}
}

func assignIngredientIDs(in []Ingredient) []Ingredient {
	out := make([]Ingredient, len(in))
	for i, ing := range in {
		ing.ID = uuid.NewUUID()
		ing.SortOrder = i
		out[i] = ing
	}
	return out
}

func assignStepIDs(in []Step) []Step {
	out := make([]Step, len(in))
	for i, s := range in {
		s.ID = uuid.NewUUID()
		s.StepNumber = i + 1
		out[i] = s
	}
	return out
}

func sanitizeFilename(name string) string {
	out := make([]byte, 0, len(name))
	for _, r := range name {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '.' || r == '-' || r == '_' {
			out = append(out, byte(r))
		}
	}
	if len(out) == 0 {
		return "image"
	}
	return string(out)
}
