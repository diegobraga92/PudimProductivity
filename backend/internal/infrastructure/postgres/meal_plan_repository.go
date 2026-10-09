package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	mealplandomain "github.com/diegobraga92/pudimproductivity/backend/internal/contexts/productivity/mealplan"
)

type MealPlanRepository struct {
	pool *pgxpool.Pool
}

func NewMealPlanRepository(pool *pgxpool.Pool) *MealPlanRepository {
	return &MealPlanRepository{pool: pool}
}

const mealPlanColumns = `id, day, meal, name, amount, calories, recipe_id, created_at, updated_at`

func scanMealPlanItem(scanner interface{ Scan(dest ...any) error }) (*mealplandomain.Item, error) {
	item := &mealplandomain.Item{}
	var recipeID *string
	err := scanner.Scan(
		&item.ID, (*string)(&item.Day), (*string)(&item.Meal),
		&item.Name, &item.Amount, &item.Calories, &recipeID,
		&item.CreatedAt, &item.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	item.RecipeID = recipeID
	return item, nil
}

// CreateBatch inserts every item inside a single transaction so a multi-day
// create either lands entirely or not at all.
func (r *MealPlanRepository) CreateBatch(ctx context.Context, items []*mealplandomain.Item) error {
	if len(items) == 0 {
		return nil
	}

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	for _, item := range items {
		if err := tx.QueryRow(ctx, `
			INSERT INTO meal_plan_items (id, day, meal, name, amount, calories, recipe_id)
			VALUES ($1, $2, $3, $4, $5, $6, $7)
			RETURNING created_at, updated_at`,
			item.ID, item.Day, item.Meal, item.Name, item.Amount, item.Calories, item.RecipeID,
		).Scan(&item.CreatedAt, &item.UpdatedAt); err != nil {
			return fmt.Errorf("insert meal plan item: %w", err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit: %w", err)
	}
	return nil
}

func (r *MealPlanRepository) List(ctx context.Context) ([]*mealplandomain.Item, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT `+mealPlanColumns+`
		FROM meal_plan_items
		ORDER BY meal, day, created_at, id`)
	if err != nil {
		return nil, fmt.Errorf("list meal plan items: %w", err)
	}
	defer rows.Close()

	var out []*mealplandomain.Item
	for rows.Next() {
		item, err := scanMealPlanItem(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, item)
	}
	return out, rows.Err()
}

func (r *MealPlanRepository) Update(ctx context.Context, item *mealplandomain.Item) error {
	err := r.pool.QueryRow(ctx, `
		UPDATE meal_plan_items SET
			day = $2, meal = $3, name = $4, amount = $5, calories = $6,
			recipe_id = $7, updated_at = NOW()
		WHERE id = $1
		RETURNING created_at, updated_at`,
		item.ID, item.Day, item.Meal, item.Name, item.Amount, item.Calories, item.RecipeID,
	).Scan(&item.CreatedAt, &item.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return mealplandomain.ErrNotFound
	}
	if err != nil {
		return fmt.Errorf("update meal plan item: %w", err)
	}
	return nil
}

func (r *MealPlanRepository) DeleteAll(ctx context.Context) ([]string, error) {
	rows, err := r.pool.Query(ctx, `DELETE FROM meal_plan_items RETURNING id`)
	if err != nil {
		return nil, fmt.Errorf("delete meal plan items: %w", err)
	}
	defer rows.Close()

	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

func (r *MealPlanRepository) Delete(ctx context.Context, id string) error {
	tag, err := r.pool.Exec(ctx, `DELETE FROM meal_plan_items WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete meal plan item: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return mealplandomain.ErrNotFound
	}
	return nil
}
