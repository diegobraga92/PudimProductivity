package recipe

import (
	"testing"
)

func mustIngredient(name string) Ingredient { return Ingredient{Name: name} }
func mustStep(text string) Step              { return Step{Instruction: text} }

func TestNewRecipe_Valid(t *testing.T) {
	r, err := NewRecipe(
		"id-1", "Pancakes", "Fluffy breakfast", DifficultyEasy,
		10, 15, 4,
		ptr("https://img/pancakes.jpg"), nil,
		[]string{"Breakfast", "sweet", "breakfast"},
		[]Ingredient{mustIngredient("Flour")}, []Step{mustStep("Mix")},
	)
	if err != nil {
		t.Fatalf("NewRecipe: %v", err)
	}
	if r.Title != "Pancakes" {
		t.Fatalf("title = %q", r.Title)
	}
	// Tags must be normalized (lowercased, deduped, sorted).
	if len(r.Tags) != 2 || r.Tags[0] != "breakfast" || r.Tags[1] != "sweet" {
		t.Fatalf("tags = %v", r.Tags)
	}
	if r.TotalMinutes() != 25 {
		t.Fatalf("TotalMinutes = %d", r.TotalMinutes())
	}
}

func TestNewRecipe_Validation(t *testing.T) {
	cases := []struct {
		name    string
		id      string
		title   string
		diff    Difficulty
		prep    int
		cook    int
		serv    int
		ing     []Ingredient
		step    []Step
	}{
		{"empty id", "", "X", DifficultyEasy, 0, 0, 1, nil, nil},
		{"empty title", "id", "", DifficultyEasy, 0, 0, 1, nil, nil},
		{"whitespace title", "id", "   ", DifficultyEasy, 0, 0, 1, nil, nil},
		{"bad difficulty", "id", "X", "extreme", 0, 0, 1, nil, nil},
		{"negative time", "id", "X", DifficultyEasy, -1, 0, 1, nil, nil},
		{"zero servings", "id", "X", DifficultyEasy, 0, 0, 0, nil, nil},
		{"empty ingredient name", "id", "X", DifficultyEasy, 0, 0, 1, []Ingredient{{Name: " "}}, nil},
		{"empty step text", "id", "X", DifficultyEasy, 0, 0, 1, nil, []Step{{Instruction: ""}}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := NewRecipe(
				tc.id, tc.title, "", tc.diff, tc.prep, tc.cook, tc.serv,
				nil, nil, nil, tc.ing, tc.step,
			)
			if err == nil {
				t.Fatal("expected validation error")
			}
		})
	}
}

func TestNormalizedTags(t *testing.T) {
	got := normalizedTags([]string{"  Veggie ", "veggie", "vegan", "", "Quick"})
	want := []string{"quick", "vegan", "veggie"}
	if len(got) != len(want) {
		t.Fatalf("got %v want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("got %v want %v", got, want)
		}
	}
}

func TestDifficultyValid(t *testing.T) {
	for _, d := range []Difficulty{DifficultyEasy, DifficultyMedium, DifficultyHard} {
		if !d.Valid() {
			t.Fatalf("%q should be valid", d)
		}
	}
	if Difficulty("nope").Valid() {
		t.Fatal("invalid difficulty should not be valid")
	}
}

func ptr(s string) *string { return &s }
