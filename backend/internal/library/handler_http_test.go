package library

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/diegobraga92/pudimproductivity/backend/internal/featureflag"
)

// batchStub is a configured ScoreLookupProvider that echoes the query back as
// a single candidate.
type batchStub struct {
	configured bool
	fail       bool
}

func (s *batchStub) Search(_ context.Context, query ScoreQuery) ([]ScoreCandidate, error) {
	if s.fail {
		return nil, errStub
	}
	return []ScoreCandidate{{Title: query.Name, Year: 2000, Score: 88, Source: "metacritic"}}, nil
}

func (s *batchStub) Configured() bool { return s.configured }

var errStub = &batchStubError{}

type batchStubError struct{}

func (e *batchStubError) Error() string { return "stub provider failure" }

func postBatch(t *testing.T, h *Handler, items []ScoreBatchItemRequest) *httptest.ResponseRecorder {
	t.Helper()
	body, _ := json.Marshal(ScoreBatchRequest{Items: items})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/library/score/batch", bytes.NewReader(body))
	rec := httptest.NewRecorder()
	h.SearchScoresBatch(rec, req)
	return rec
}

func TestSearchScoresBatch_NotConfigured_Returns503(t *testing.T) {
	h := NewHandler(nil, &batchStub{configured: false}, (*featureflag.Service)(nil))
	rec := postBatch(t, h, []ScoreBatchItemRequest{{Name: "Zelda", MediaType: "game"}})
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want 503", rec.Code)
	}
}

func TestSearchScoresBatch_ReturnsCandidatesInOrder(t *testing.T) {
	h := NewHandler(nil, &batchStub{configured: true}, (*featureflag.Service)(nil))
	items := []ScoreBatchItemRequest{
		{Name: "Zelda", MediaType: "game", Year: intPtr(2017)},
		{Name: "Odyssey", MediaType: "game", Year: intPtr(2017)},
	}
	rec := postBatch(t, h, items)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200: %s", rec.Code, rec.Body.String())
	}
	var resp ScoreBatchResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(resp.Results) != 2 {
		t.Fatalf("results = %d, want 2", len(resp.Results))
	}
	if resp.Results[0].Index != 0 || len(resp.Results[0].Candidates) != 1 {
		t.Fatalf("result[0] = %+v", resp.Results[0])
	}
	if resp.Results[0].Candidates[0].Title != "Zelda" {
		t.Fatalf("candidate title = %q, want Zelda", resp.Results[0].Candidates[0].Title)
	}
	if resp.Results[1].Index != 1 || resp.Results[1].Candidates[0].Title != "Odyssey" {
		t.Fatalf("result[1] = %+v", resp.Results[1])
	}
}

func TestSearchScoresBatch_InvalidType_InlineError(t *testing.T) {
	h := NewHandler(nil, &batchStub{configured: true}, (*featureflag.Service)(nil))
	rec := postBatch(t, h, []ScoreBatchItemRequest{{Name: "x", MediaType: "cartridge"}})
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (per-item error)", rec.Code)
	}
	var resp ScoreBatchResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.Results[0].Error == "" {
		t.Fatal("expected per-item error for invalid type")
	}
}

func TestSearchScoresBatch_ProviderError_InlineError(t *testing.T) {
	h := NewHandler(nil, &batchStub{configured: true, fail: true}, (*featureflag.Service)(nil))
	rec := postBatch(t, h, []ScoreBatchItemRequest{{Name: "x", MediaType: "game"}})
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (per-item error)", rec.Code)
	}
	var resp ScoreBatchResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.Results[0].Error == "" {
		t.Fatal("expected per-item error for provider failure")
	}
}

func TestSearchScoresBatch_EmptyItems_BadRequest(t *testing.T) {
	h := NewHandler(nil, &batchStub{configured: true}, (*featureflag.Service)(nil))
	rec := postBatch(t, h, nil)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
}

func TestSearchScoresBatch_TooManyItems_BadRequest(t *testing.T) {
	h := NewHandler(nil, &batchStub{configured: true}, (*featureflag.Service)(nil))
	items := make([]ScoreBatchItemRequest, maxScoreBatchItems+1)
	rec := postBatch(t, h, items)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
}

func intPtr(i int) *int { return &i }
