package mealplan

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresRepository struct {
	pool *pgxpool.Pool
}

func NewPostgresRepository(pool *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{pool: pool}
}

const planColumns = `id, name, start_date, end_date, is_published, created_at, updated_at`

func scanPlan(scanner interface{ Scan(dest ...any) error }) (*MealPlan, error) {
	p := &MealPlan{}
	err := scanner.Scan(&p.ID, &p.Name, &p.StartDate, &p.EndDate, &p.IsPublished, &p.CreatedAt, &p.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return p, nil
}

func (r *PostgresRepository) Create(ctx context.Context, plan *MealPlan) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if _, err := tx.Exec(ctx, `
		INSERT INTO meal_plans (id, name, start_date, end_date, is_published)
		VALUES ($1, $2, $3, $4, $5)`,
		plan.ID, plan.Name, plan.StartDate, plan.EndDate, plan.IsPublished,
	); err != nil {
		return fmt.Errorf("insert meal plan: %w", err)
	}
	if err := insertSlots(ctx, tx, plan.ID, plan.Slots); err != nil {
		return err
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit: %w", err)
	}
	return nil
}

func insertSlots(ctx context.Context, tx pgx.Tx, planID string, slots []MealSlot) error {
	for _, s := range slots {
		if _, err := tx.Exec(ctx, `
			INSERT INTO meal_plan_slots (id, meal_plan_id, date, meal_type, recipe_id, notes)
			VALUES ($1, $2, $3, $4, $5, $6)`,
			s.ID, planID, s.Date, s.MealType, s.RecipeID, s.Notes,
		); err != nil {
			return fmt.Errorf("insert slot: %w", err)
		}
	}
	return nil
}

func (r *PostgresRepository) GetByID(ctx context.Context, id string) (*MealPlan, error) {
	row := r.pool.QueryRow(ctx, `SELECT `+planColumns+` FROM meal_plans WHERE id = $1`, id)
	plan, err := scanPlan(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get meal plan %s: %w", id, err)
	}

	slots, err := r.loadSlots(ctx, id)
	if err != nil {
		return nil, err
	}
	plan.Slots = slots
	return plan, nil
}

func (r *PostgresRepository) loadSlots(ctx context.Context, planID string) ([]MealSlot, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, date, meal_type, recipe_id, notes
		FROM meal_plan_slots WHERE meal_plan_id = $1 ORDER BY date, meal_type`, planID)
	if err != nil {
		return nil, fmt.Errorf("load slots: %w", err)
	}
	defer rows.Close()

	var out []MealSlot
	for rows.Next() {
		var s MealSlot
		if err := rows.Scan(&s.ID, &s.Date, (*string)(&s.MealType), &s.RecipeID, &s.Notes); err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

func (r *PostgresRepository) List(ctx context.Context) ([]*MealPlan, error) {
	rows, err := r.pool.Query(ctx, `SELECT `+planColumns+` FROM meal_plans ORDER BY start_date DESC`)
	if err != nil {
		return nil, fmt.Errorf("list meal plans: %w", err)
	}
	defer rows.Close()

	var out []*MealPlan
	for rows.Next() {
		plan, err := scanPlan(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, plan)
	}
	return out, rows.Err()
}

func (r *PostgresRepository) Update(ctx context.Context, plan *MealPlan) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	tag, err := tx.Exec(ctx, `
		UPDATE meal_plans SET name = $2, start_date = $3, end_date = $4, updated_at = NOW()
		WHERE id = $1`, plan.ID, plan.Name, plan.StartDate, plan.EndDate)
	if err != nil {
		return fmt.Errorf("update meal plan: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}

	// Replace slots wholesale.
	if _, err := tx.Exec(ctx, `DELETE FROM meal_plan_slots WHERE meal_plan_id = $1`, plan.ID); err != nil {
		return fmt.Errorf("clear slots: %w", err)
	}
	if err := insertSlots(ctx, tx, plan.ID, plan.Slots); err != nil {
		return err
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit: %w", err)
	}
	return nil
}

func (r *PostgresRepository) Delete(ctx context.Context, id string) error {
	tag, err := r.pool.Exec(ctx, `DELETE FROM meal_plans WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete meal plan: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *PostgresRepository) UpdateSlot(ctx context.Context, planID, slotID string, recipeID *string, notes *string) error {
	var sets []string
	var args []any
	if recipeID != nil {
		args = append(args, *recipeID)
		sets = append(sets, fmt.Sprintf("recipe_id = $%d", len(args)))
	}
	if notes != nil {
		args = append(args, *notes)
		sets = append(sets, fmt.Sprintf("notes = $%d", len(args)))
	}
	if len(sets) == 0 {
		return nil // no-op
	}
	args = append(args, planID, slotID)
	query := fmt.Sprintf(
		"UPDATE meal_plan_slots SET %s WHERE meal_plan_id = $%d AND id = $%d",
		strings.Join(sets, ", "), len(args)-1, len(args),
	)

	tag, err := r.pool.Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("update slot: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *PostgresRepository) ReplaceShoppingList(ctx context.Context, planID string, items []ShoppingItem) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if _, err := tx.Exec(ctx, `DELETE FROM meal_plan_shopping_list WHERE meal_plan_id = $1`, planID); err != nil {
		return fmt.Errorf("clear shopping list: %w", err)
	}
	for _, item := range items {
		if _, err := tx.Exec(ctx, `
			INSERT INTO meal_plan_shopping_list (id, meal_plan_id, ingredient_name, quantity_agg, unit)
			VALUES ($1, $2, $3, $4, $5)`,
			item.ID, planID, item.IngredientName, item.QuantityAgg, item.Unit,
		); err != nil {
			return fmt.Errorf("insert shopping item: %w", err)
		}
	}
	return tx.Commit(ctx)
}

func (r *PostgresRepository) GetShoppingList(ctx context.Context, planID string) ([]ShoppingItem, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, ingredient_name, quantity_agg, unit, is_checked
		FROM meal_plan_shopping_list WHERE meal_plan_id = $1 ORDER BY ingredient_name`, planID)
	if err != nil {
		return nil, fmt.Errorf("get shopping list: %w", err)
	}
	defer rows.Close()

	var out []ShoppingItem
	for rows.Next() {
		var item ShoppingItem
		if err := rows.Scan(&item.ID, &item.IngredientName, &item.QuantityAgg, &item.Unit, &item.IsChecked); err != nil {
			return nil, err
		}
		out = append(out, item)
	}
	return out, rows.Err()
}

func (r *PostgresRepository) ToggleShoppingItem(ctx context.Context, planID, itemID string) error {
	tag, err := r.pool.Exec(ctx, `
		UPDATE meal_plan_shopping_list SET is_checked = NOT is_checked
		WHERE meal_plan_id = $1 AND id = $2`, planID, itemID)
	if err != nil {
		return fmt.Errorf("toggle shopping item: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *PostgresRepository) SetPublished(ctx context.Context, id string) error {
	tag, err := r.pool.Exec(ctx, `
		UPDATE meal_plans SET is_published = true, updated_at = NOW() WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("publish meal plan: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

