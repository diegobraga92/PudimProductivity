package mealplan

import (
	"bytes"
	"fmt"
	"strings"
	"time"

	"github.com/go-pdf/fpdf"
)

// PDF render constants (A4 portrait, mm units).
const (
	pdfMargin      = 12.0
	pdfPageWidth   = 210.0
	pdfContentW    = pdfPageWidth - 2*pdfMargin
	pdfHeaderColor = 52
)

// RenderPlanPDF renders a printable weekly meal-plan PDF: a day × meal-type
// grid (recipe titles or notes per slot) followed by the shopping list.
// recipeTitles maps recipe IDs to titles (resolved by the caller).
func RenderPlanPDF(plan *MealPlan, shopping []ShoppingItem, recipeTitles map[string]string) ([]byte, error) {
	pdf := fpdf.New("P", "mm", "A4", "")
	pdf.SetMargins(pdfMargin, pdfMargin, pdfMargin)
	pdf.SetAutoPageBreak(true, 20)
	// Keep content uncompressed: meal-plan PDFs are small, and it lets tests
	// (and debugging) inspect the embedded text.
	pdf.SetCompression(false)
	pdf.AddPage()

	// Header
	pdf.SetFont("Helvetica", "B", 18)
	pdf.SetTextColor(pdfHeaderColor, pdfHeaderColor, pdfHeaderColor)
	pdf.CellFormat(pdfContentW, 12, "Meal Plan", "", 1, "L", false, 0, "")
	pdf.SetFont("Helvetica", "B", 14)
	pdf.CellFormat(pdfContentW, 10, plan.Name, "", 1, "L", false, 0, "")
	pdf.SetFont("Helvetica", "", 10)
	pdf.SetTextColor(90, 90, 90)
	pdf.CellFormat(pdfContentW, 7, fmt.Sprintf("%s  →  %s",
		plan.StartDate.Format("Mon 02 Jan 2006"),
		plan.EndDate.Format("Mon 02 Jan 2006")), "", 1, "L", false, 0, "")
	pdf.Ln(4)

	// Build the day columns.
	var days []time.Time
	for d := plan.StartDate; !d.After(plan.EndDate); d = d.AddDate(0, 0, 1) {
		days = append(days, d)
	}
	if len(days) == 0 {
		days = append(days, plan.StartDate)
	}
	daysByDate := make(map[time.Time]map[MealType]MealSlot, len(days))
	for _, slot := range plan.Slots {
		if daysByDate[slot.Date.Truncate(24*time.Hour)] == nil {
			daysByDate[slot.Date.Truncate(24*time.Hour)] = make(map[MealType]MealSlot)
		}
		daysByDate[slot.Date.Truncate(24*time.Hour)][slot.MealType] = slot
	}

	mealTypes := []MealType{MealBreakfast, MealLunch, MealDinner, MealSnack}
	colW := pdfContentW / float64(len(days))
	rowH := 22.0

	// Column headers (weekday names).
	pdf.SetFont("Helvetica", "B", 9)
	pdf.SetFillColor(235, 235, 235)
	pdf.SetTextColor(40, 40, 40)
	_, lineH := pdf.GetFontSize()
	headerH := lineH*2 + 4
	for i, d := range days {
		pdf.SetX(pdfMargin + float64(i)*colW)
		pdf.CellFormat(colW, headerH, d.Format("Mon\n02/01"), "", 0, "C", true, 0, "")
	}
	pdf.Ln(headerH)

	// Meal-type rows.
	for _, meal := range mealTypes {
		pdf.SetFont("Helvetica", "B", 9)
		pdf.SetTextColor(60, 60, 60)
		pdf.CellFormat(colW, rowH, " "+string(meal), "", 0, "L", false, 0, "")
		for i, d := range days {
			slot, ok := daysByDate[d.Truncate(24*time.Hour)][meal]
			cell := "—"
			if ok && slot.RecipeID != nil {
				if title, found := recipeTitles[*slot.RecipeID]; found && title != "" {
					cell = title
				} else {
					cell = "Recipe"
				}
			} else if ok && strings.TrimSpace(slot.Notes) != "" {
				cell = slot.Notes
			}
			// Day columns start after the meal-label column.
			pdf.SetXY(pdfMargin+colW+float64(i)*colW, pdf.GetY())
			pdf.SetFont("Helvetica", "", 8)
			pdf.SetTextColor(30, 30, 30)
			pdf.MultiCell(colW, 4, cell, "1", "C", false)
		}
		pdf.Ln(rowH)
	}

	// Shopping list.
	if len(shopping) > 0 {
		pdf.AddPage()
		pdf.SetFont("Helvetica", "B", 14)
		pdf.SetTextColor(pdfHeaderColor, pdfHeaderColor, pdfHeaderColor)
		pdf.CellFormat(pdfContentW, 10, "Shopping List", "", 1, "L", false, 0, "")
		pdf.Ln(3)
		pdf.SetFont("Helvetica", "", 10)
		pdf.SetTextColor(40, 40, 40)
		for _, item := range shopping {
			marker := "☐"
			if item.IsChecked {
				marker = "☑"
			}
			qty := strings.TrimSpace(item.QuantityAgg)
			name := item.IngredientName
			if qty != "" {
				name = fmt.Sprintf("%s  —  %s", item.IngredientName, qty)
			}
			pdf.CellFormat(8, 6, marker, "", 0, "L", false, 0, "")
			pdf.MultiCell(pdfContentW-8, 6, name, "", "L", false)
		}
	}

	var buf bytes.Buffer
	if err := pdf.Output(&buf); err != nil {
		return nil, fmt.Errorf("render meal plan pdf: %w", err)
	}
	return buf.Bytes(), nil
}
