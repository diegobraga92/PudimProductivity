package mealplan

import (
	"bytes"
	"strings"
	"testing"
	"time"
)

// TestRenderPlanPDF_ProducesValidPdf checks the renderer emits a PDF header and
// includes the plan name.
func TestRenderPlanPDF_ProducesValidPdf(t *testing.T) {
	start := time.Date(2026, 8, 10, 0, 0, 0, 0, time.UTC) // Monday
	plan := &MealPlan{
		ID:        "p1",
		Name:      "Week of Fuel",
		StartDate: start,
		EndDate:   start.AddDate(0, 0, 6),
		Slots: []MealSlot{
			{ID: "s1", Date: start, MealType: MealBreakfast, Notes: "Oats"},
			{ID: "s2", Date: start, MealType: MealLunch, RecipeID: ptr("r1")},
		},
	}
	shopping := []ShoppingItem{
		{ID: "i1", IngredientName: "Oats", QuantityAgg: "1 cup", IsChecked: false},
		{ID: "i2", IngredientName: "Eggs", QuantityAgg: "6", IsChecked: true},
	}
	titles := map[string]string{"r1": "Pancakes"}

	data, err := RenderPlanPDF(plan, shopping, titles)
	if err != nil {
		t.Fatalf("RenderPlanPDF: %v", err)
	}
	if !bytes.HasPrefix(data, []byte("%PDF")) {
		t.Fatalf("output does not start with %%PDF: %q", data[:8])
	}
	if !bytes.Contains(data, []byte("Pancakes")) {
		t.Error("expected recipe title in PDF content")
	}
}

// TestRenderPlanPDF_EmptyPlanStillRenders — a plan with no slots must render.
func TestRenderPlanPDF_EmptyPlanStillRenders(t *testing.T) {
	start := time.Date(2026, 8, 10, 0, 0, 0, 0, time.UTC)
	plan := &MealPlan{ID: "p2", Name: "Empty", StartDate: start, EndDate: start.AddDate(0, 0, 6)}

	data, err := RenderPlanPDF(plan, nil, nil)
	if err != nil {
		t.Fatalf("RenderPlanPDF: %v", err)
	}
	if !strings.HasPrefix(string(data), "%PDF") {
		t.Fatal("expected a PDF")
	}
}

func ptr(s string) *string { return &s }
