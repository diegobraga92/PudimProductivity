package recipe

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ErrNotFound is returned when a recipe does not exist.
var ErrNotFound = errors.New("recipe not found")

type PostgresRepository struct {
	pool *pgxpool.Pool
}

func NewPostgresRepository(pool *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{pool: pool}
}

const recipeColumns = `id, title, description, difficulty, prep_time_minutes, cook_time_minutes, servings, image_url, source_url, created_at, updated_at`

func scanRecipe(scanner interface{ Scan(dest ...any) error }) (*Recipe, error) {
	r := &Recipe{}
	var imageURL, sourceURL *string
	err := scanner.Scan(
		&r.ID, &r.Title, &r.Description, (*string)(&r.Difficulty),
		&r.PrepMinutes, &r.CookMinutes, &r.Servings,
		&imageURL, &sourceURL, &r.CreatedAt, &r.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	r.ImageURL = imageURL
	r.SourceURL = sourceURL
	return r, nil
}

// scanRecipeRow scans the List projection: the recipe columns plus the
// aggregated tags array in a single Scan call.
func scanRecipeRow(scanner interface{ Scan(dest ...any) error }) (*Recipe, error) {
	r := &Recipe{}
	var imageURL, sourceURL *string
	err := scanner.Scan(
		&r.ID, &r.Title, &r.Description, (*string)(&r.Difficulty),
		&r.PrepMinutes, &r.CookMinutes, &r.Servings,
		&imageURL, &sourceURL, &r.CreatedAt, &r.UpdatedAt,
		&r.Tags,
	)
	if err != nil {
		return nil, err
	}
	r.ImageURL = imageURL
	r.SourceURL = sourceURL
	return r, nil
}

func (r *PostgresRepository) Create(ctx context.Context, recipe *Recipe) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if _, err := tx.Exec(ctx, `
		INSERT INTO recipes (id, title, description, difficulty, prep_time_minutes, cook_time_minutes, servings, image_url, source_url)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`,
		recipe.ID, recipe.Title, recipe.Description, recipe.Difficulty,
		recipe.PrepMinutes, recipe.CookMinutes, recipe.Servings,
		recipe.ImageURL, recipe.SourceURL,
	); err != nil {
		return fmt.Errorf("insert recipe: %w", err)
	}

	if err := insertChildren(ctx, tx, recipe); err != nil {
		return err
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit: %w", err)
	}
	return nil
}

// insertChildren writes tags, ingredients and steps for an existing recipe row.
func insertChildren(ctx context.Context, tx pgx.Tx, recipe *Recipe) error {
	for _, tag := range recipe.Tags {
		if _, err := tx.Exec(ctx,
			`INSERT INTO recipe_tags (recipe_id, tag) VALUES ($1, $2)`,
			recipe.ID, tag,
		); err != nil {
			return fmt.Errorf("insert tag %q: %w", tag, err)
		}
	}
	for i, ing := range recipe.Ingredients {
		if ing.SortOrder == 0 {
			ing.SortOrder = i
		}
		if _, err := tx.Exec(ctx, `
			INSERT INTO recipe_ingredients (id, recipe_id, name, quantity, unit, sort_order)
			VALUES ($1, $2, $3, $4, $5, $6)`,
			ing.ID, recipe.ID, ing.Name, ing.Quantity, ing.Unit, ing.SortOrder,
		); err != nil {
			return fmt.Errorf("insert ingredient %q: %w", ing.Name, err)
		}
	}
	for _, s := range recipe.Steps {
		if _, err := tx.Exec(ctx, `
			INSERT INTO recipe_steps (id, recipe_id, step_number, instruction)
			VALUES ($1, $2, $3, $4)`,
			s.ID, recipe.ID, s.StepNumber, s.Instruction,
		); err != nil {
			return fmt.Errorf("insert step %d: %w", s.StepNumber, err)
		}
	}
	return nil
}

func (r *PostgresRepository) GetByID(ctx context.Context, id string) (*Recipe, error) {
	row := r.pool.QueryRow(ctx, `
		SELECT `+recipeColumns+` FROM recipes WHERE id = $1`, id)
	recipe, err := scanRecipe(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get recipe %s: %w", id, err)
	}

	tags, err := r.loadTags(ctx, id)
	if err != nil {
		return nil, err
	}
	recipe.Tags = tags

	ingredients, err := r.loadIngredients(ctx, id)
	if err != nil {
		return nil, err
	}
	recipe.Ingredients = ingredients

	steps, err := r.loadSteps(ctx, id)
	if err != nil {
		return nil, err
	}
	recipe.Steps = steps

	return recipe, nil
}

func (r *PostgresRepository) loadTags(ctx context.Context, recipeID string) ([]string, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT tag FROM recipe_tags WHERE recipe_id = $1 ORDER BY tag`, recipeID)
	if err != nil {
		return nil, fmt.Errorf("load tags: %w", err)
	}
	defer rows.Close()

	var tags []string
	for rows.Next() {
		var t string
		if err := rows.Scan(&t); err != nil {
			return nil, err
		}
		tags = append(tags, t)
	}
	return tags, rows.Err()
}

func (r *PostgresRepository) loadIngredients(ctx context.Context, recipeID string) ([]Ingredient, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, name, quantity, unit, sort_order
		FROM recipe_ingredients WHERE recipe_id = $1 ORDER BY sort_order, name`, recipeID)
	if err != nil {
		return nil, fmt.Errorf("load ingredients: %w", err)
	}
	defer rows.Close()

	var out []Ingredient
	for rows.Next() {
		var ing Ingredient
		if err := rows.Scan(&ing.ID, &ing.Name, &ing.Quantity, &ing.Unit, &ing.SortOrder); err != nil {
			return nil, err
		}
		out = append(out, ing)
	}
	return out, rows.Err()
}

func (r *PostgresRepository) loadSteps(ctx context.Context, recipeID string) ([]Step, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, step_number, instruction
		FROM recipe_steps WHERE recipe_id = $1 ORDER BY step_number`, recipeID)
	if err != nil {
		return nil, fmt.Errorf("load steps: %w", err)
	}
	defer rows.Close()

	var out []Step
	for rows.Next() {
		var s Step
		if err := rows.Scan(&s.ID, &s.StepNumber, &s.Instruction); err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

func (r *PostgresRepository) List(ctx context.Context, filter ListFilter) ([]*Recipe, error) {
	limit := filter.Limit
	if limit <= 0 || limit > 100 {
		limit = 50
	}

	var conds []string
	var args []any

	if strings.TrimSpace(filter.Search) != "" {
		args = append(args, "%"+strings.ToLower(filter.Search)+"%")
		conds = append(conds, `(LOWER(r.title) LIKE $`+itoa(len(args))+` OR LOWER(r.description) LIKE $`+itoa(len(args))+`)`)
	}
	if filter.Difficulty != "" {
		args = append(args, filter.Difficulty)
		conds = append(conds, `r.difficulty = $`+itoa(len(args)))
	}
	if len(filter.Tags) > 0 {
		args = append(args, filter.Tags)
		conds = append(conds, `EXISTS (SELECT 1 FROM recipe_tags rt WHERE rt.recipe_id = r.id AND rt.tag = ANY($`+itoa(len(args))+`))`)
	}
	if filter.Cursor != nil {
		args = append(args, *filter.Cursor, filter.CursorID)
		conds = append(conds, `(r.created_at, r.id) < ($`+itoa(len(args)-1)+`, $`+itoa(len(args))+`)`)
	}

	where := ""
	if len(conds) > 0 {
		where = "WHERE " + strings.Join(conds, " AND ")
	}
	args = append(args, limit)
	query := fmt.Sprintf(`
		SELECT %s, COALESCE(array_agg(DISTINCT t.tag) FILTER (WHERE t.tag IS NOT NULL), '{}')
		FROM recipes r
		LEFT JOIN recipe_tags t ON t.recipe_id = r.id
		%s
		GROUP BY r.id
		ORDER BY r.created_at DESC, r.id DESC
		LIMIT $%d`, recipeColumns, where, len(args))

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list recipes: %w", err)
	}
	defer rows.Close()

	var out []*Recipe
	for rows.Next() {
		recipe, err := scanRecipeRow(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, recipe)
	}
	return out, rows.Err()
}

func (r *PostgresRepository) Update(ctx context.Context, recipe *Recipe) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	tag, err := tx.Exec(ctx, `
		UPDATE recipes SET
			title = $2, description = $3, difficulty = $4,
			prep_time_minutes = $5, cook_time_minutes = $6, servings = $7,
			image_url = $8, source_url = $9, updated_at = NOW()
		WHERE id = $1`,
		recipe.ID, recipe.Title, recipe.Description, recipe.Difficulty,
		recipe.PrepMinutes, recipe.CookMinutes, recipe.Servings,
		recipe.ImageURL, recipe.SourceURL,
	)
	if err != nil {
		return fmt.Errorf("update recipe: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}

	// Replace children wholesale (delete then insert) — simpler and idempotent.
	for _, table := range []string{"recipe_tags", "recipe_ingredients", "recipe_steps"} {
		if _, err := tx.Exec(ctx, `DELETE FROM `+table+` WHERE recipe_id = $1`, recipe.ID); err != nil {
			return fmt.Errorf("clear %s: %w", table, err)
		}
	}
	if err := insertChildren(ctx, tx, recipe); err != nil {
		return err
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit: %w", err)
	}
	return nil
}

func (r *PostgresRepository) Delete(ctx context.Context, id string) error {
	tag, err := r.pool.Exec(ctx, `DELETE FROM recipes WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete recipe: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// itoa is a tiny helper to keep the dynamic SQL builder readable.
func itoa(n int) string {
	return fmt.Sprintf("%d", n)
}

