package scoring

import (
	"context"
	"testing"

	"github.com/diegobraga92/pudimproductivity/backend/internal/library"
	"github.com/diegobraga92/pudimproductivity/backend/internal/shared"
)

func TestNewComposite_NoProviders_ReturnsNoop(t *testing.T) {
	client, err := NewComposite(context.Background(), shared.ScoreProviderConfig{})
	if err != nil {
		t.Fatalf("NewComposite: %v", err)
	}
	if _, ok := client.(library.NoopScoreLookup); !ok {
		t.Fatalf("expected NoopScoreLookup, got %T", client)
	}
}

func TestNewComposite_UnknownProvider(t *testing.T) {
	cfg := shared.ScoreProviderConfig{Movie: "netflix", Keys: map[string]string{"netflix": "k"}}
	if _, err := NewComposite(context.Background(), cfg); err == nil {
		t.Fatal("expected error for unknown provider")
	}
}

func TestNewComposite_ProviderMediaTypeMismatch(t *testing.T) {
	cfg := shared.ScoreProviderConfig{
		Movie: "rawg", // RAWG only supports games
		Keys:  map[string]string{"rawg": "k"},
	}
	if _, err := NewComposite(context.Background(), cfg); err == nil {
		t.Fatal("expected error for provider/media-type mismatch")
	}
}

func TestNewComposite_MissingKey(t *testing.T) {
	cfg := shared.ScoreProviderConfig{Movie: "omdb"} // provider selected, no key
	if _, err := NewComposite(context.Background(), cfg); err == nil {
		t.Fatal("expected error for missing API key")
	}
}

func TestNewComposite_NoneDisablesType(t *testing.T) {
	client, err := NewComposite(context.Background(), shared.ScoreProviderConfig{
		Movie: "none",
		Game:  "rawg",
		Keys:  map[string]string{"rawg": "k"},
	})
	if err != nil {
		t.Fatalf("NewComposite: %v", err)
	}
	cands, err := client.Search(context.Background(), library.ScoreQuery{Name: "x", MediaType: library.MediaTypeMovie})
	if err != nil || len(cands) != 0 {
		t.Fatalf("Search(disabled movie) = %v (err %v), want empty", cands, err)
	}
}

type stubClient struct {
	name  string
	calls int
}

func (s *stubClient) Search(_ context.Context, _ library.ScoreQuery) ([]library.ScoreCandidate, error) {
	s.calls++
	return nil, nil
}

func TestComposite_RoutesByMediaType(t *testing.T) {
	stubA := &stubClient{name: "A"}
	stubB := &stubClient{name: "B"}
	c := &composite{clients: map[library.MediaType]library.ScoreLookupClient{
		library.MediaTypeMovie: stubA,
		library.MediaTypeGame:  stubB,
	}}
	ctx := context.Background()

	if _, err := c.Search(ctx, library.ScoreQuery{Name: "x", MediaType: library.MediaTypeMovie}); err != nil {
		t.Fatalf("Search(movie): %v", err)
	}
	if stubA.calls != 1 || stubB.calls != 0 {
		t.Fatalf("movie routing wrong: A=%d B=%d", stubA.calls, stubB.calls)
	}

	// Unconfigured media type returns no candidates without calling anything.
	cands, err := c.Search(ctx, library.ScoreQuery{Name: "x", MediaType: library.MediaTypeBook})
	if err != nil || cands != nil {
		t.Fatalf("Search(book) = %v (err %v), want nil", cands, err)
	}
	if stubA.calls != 1 {
		t.Fatalf("unconfigured type must not call any provider, A=%d", stubA.calls)
	}
}
