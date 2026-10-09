package mealplan

import "testing"

func TestNewItemValid(t *testing.T) {
	recipeID := "recipe-1"
	item, err := NewItem("id-1", Monday, MealLunch, "Rice", "100g", 120, &recipeID)
	if err != nil {
		t.Fatalf("NewItem: %v", err)
	}
	if item.ID != "id-1" || item.Day != Monday || item.Meal != MealLunch {
		t.Fatalf("unexpected item identity: %+v", item)
	}
	if item.Name != "Rice" || item.Amount != "100g" || item.Calories != 120 {
		t.Fatalf("unexpected item fields: %+v", item)
	}
	if item.RecipeID == nil || *item.RecipeID != recipeID {
		t.Fatal("recipe link not preserved")
	}
}

func TestNewItemRejectsInvalidInput(t *testing.T) {
	tests := []struct {
		name     string
		id       string
		day      Weekday
		meal     Meal
		itemName string
		calories int
	}{
		{"empty id", "", Monday, MealLunch, "Rice", 100},
		{"unknown day", "id-1", "monday", MealLunch, "Rice", 100},
		{"unknown meal", "id-1", Monday, "brunch", "Rice", 100},
		{"blank name", "id-1", Monday, MealLunch, "   ", 100},
		{"negative calories", "id-1", Monday, MealLunch, "Rice", -1},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := NewItem(tc.id, tc.day, tc.meal, tc.itemName, "", tc.calories, nil); err == nil {
				t.Fatal("expected validation error")
			}
		})
	}
}

func TestMealValid(t *testing.T) {
	for _, meal := range []Meal{MealBreakfast, MealLunch, MealDinner, MealSnack} {
		if !meal.Valid() {
			t.Fatalf("%q should be valid", meal)
		}
	}
	if Meal("brunch").Valid() {
		t.Fatal("unknown meal should be invalid")
	}
}

func TestWeekdayValidAndOrder(t *testing.T) {
	days := []Weekday{Monday, Tuesday, Wednesday, Thursday, Friday, Saturday, Sunday}
	for i, day := range days {
		if !day.Valid() {
			t.Fatalf("%q should be valid", day)
		}
		if day.Order() != i {
			t.Fatalf("%q order = %d, want %d", day, day.Order(), i)
		}
	}
	if Weekday("monday").Valid() {
		t.Fatal("unknown weekday should be invalid")
	}
}
