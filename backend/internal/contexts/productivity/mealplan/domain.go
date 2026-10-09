// Package mealplan implements the weekly meal planner.
package mealplan

import (
	"fmt"
	"strings"
	"time"
)

// Meal is a fixed slot in the weekly plan.
type Meal string

const (
	MealBreakfast Meal = "breakfast"
	MealLunch     Meal = "lunch"
	MealDinner    Meal = "dinner"
	MealSnack     Meal = "snack"
)

// Valid reports whether the meal is one of the known slots.
func (m Meal) Valid() bool {
	switch m {
	case MealBreakfast, MealLunch, MealDinner, MealSnack:
		return true
	default:
		return false
	}
}

// Weekday is a day column in the weekly template.
type Weekday string

const (
	Monday    Weekday = "mon"
	Tuesday   Weekday = "tue"
	Wednesday Weekday = "wed"
	Thursday  Weekday = "thu"
	Friday    Weekday = "fri"
	Saturday  Weekday = "sat"
	Sunday    Weekday = "sun"
)

// Valid reports whether the weekday is one of the seven known days.
func (d Weekday) Valid() bool {
	switch d {
	case Monday, Tuesday, Wednesday, Thursday, Friday, Saturday, Sunday:
		return true
	default:
		return false
	}
}

var weekdayOrder = map[Weekday]int{
	Monday: 0, Tuesday: 1, Wednesday: 2, Thursday: 3, Friday: 4, Saturday: 5, Sunday: 6,
}

// Order returns the weekday position, Monday first.
func (d Weekday) Order() int { return weekdayOrder[d] }

// Item is a single food entry on one day and one meal slot.
type Item struct {
	ID        string
	Day       Weekday
	Meal      Meal
	Name      string
	Amount    string
	Calories  int
	RecipeID  *string
	CreatedAt time.Time
	UpdatedAt time.Time
}

// NewItem validates and builds a meal plan item.
func NewItem(id string, day Weekday, meal Meal, name, amount string, calories int, recipeID *string) (*Item, error) {
	if id == "" {
		return nil, fmt.Errorf("item id cannot be empty")
	}
	if !day.Valid() {
		return nil, fmt.Errorf("invalid day %q", day)
	}
	if !meal.Valid() {
		return nil, fmt.Errorf("invalid meal %q", meal)
	}
	if strings.TrimSpace(name) == "" {
		return nil, fmt.Errorf("item name cannot be empty")
	}
	if calories < 0 {
		return nil, fmt.Errorf("calories cannot be negative")
	}
	return &Item{
		ID:       id,
		Day:      day,
		Meal:     meal,
		Name:     name,
		Amount:   amount,
		Calories: calories,
		RecipeID: recipeID,
	}, nil
}
