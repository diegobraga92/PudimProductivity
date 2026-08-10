package mealplan

import (
	"testing"
	"time"
)

func TestMergeQuantities(t *testing.T) {
	cases := []struct{ a, b, want string }{
		{"2", "1", "3"},
		{"2", "", "2"},
		{"", "3", "3"},
		{"1.5", "0.5", "2"},
		{"2 cups worth", "1", "2 cups worth + 1"}, // non-numeric → joined
		{"some", "more", "some + more"},
	}
	for _, tc := range cases {
		if got := mergeQuantities(tc.a, tc.b); got != tc.want {
			t.Errorf("mergeQuantities(%q, %q) = %q, want %q", tc.a, tc.b, got, tc.want)
		}
	}
}

func TestNewMealPlan_Validation(t *testing.T) {
	start := time.Date(2026, 8, 10, 0, 0, 0, 0, time.UTC)
	end := time.Date(2026, 8, 16, 0, 0, 0, 0, time.UTC)

	// End before start.
	if _, err := NewMealPlan("id", "plan", end, start, false, nil); err == nil {
		t.Fatal("expected error when end < start")
	}
	// Invalid meal type.
	if _, err := NewMealPlan("id", "plan", start, end, false, []MealSlot{
		{Date: start, MealType: "brunch"},
	}); err == nil {
		t.Fatal("expected error for invalid meal type")
	}
	// Slot out of range.
	if _, err := NewMealPlan("id", "plan", start, end, false, []MealSlot{
		{Date: end.AddDate(0, 0, 1), MealType: MealLunch},
	}); err == nil {
		t.Fatal("expected error for out-of-range slot date")
	}
	// Valid plan.
	p, err := NewMealPlan("id", "plan", start, end, false, nil)
	if err != nil {
		t.Fatalf("valid plan rejected: %v", err)
	}
	if p.Slots == nil {
		t.Fatal("nil slots should be normalized to empty slice")
	}
}
