package booktrack

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog/log"

	"github.com/diegobraga92/pudimproductivity/backend/internal/booktrack/googlebooks"
	"github.com/diegobraga92/pudimproductivity/backend/internal/shared"
)

type Handler struct {
	service Service
}

func NewHandler(service Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) AddByISBN(w http.ResponseWriter, r *http.Request) {
	var req AddByISBNRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		shared.WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	book, err := h.service.AddByISBN(r.Context(), req.ISBN)
	switch {
	case errors.Is(err, googlebooks.ErrNotFound):
		shared.WriteError(w, http.StatusNotFound, "no book found for this ISBN")
		return
	case errors.Is(err, ErrDuplicateISBN):
		shared.WriteError(w, http.StatusConflict, "book already in your library")
		return
	case err != nil:
		log.Error().Err(err).Str("isbn", req.ISBN).Msg("add book by ISBN failed")
		shared.WriteError(w, http.StatusBadGateway, "could not look up book metadata")
		return
	}
	shared.WriteJSON(w, http.StatusCreated, toResponse(book))
}

func (h *Handler) AddManual(w http.ResponseWriter, r *http.Request) {
	var req AddManualRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		shared.WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	book, err := h.service.AddManual(r.Context(), req.toInput())
	if errors.Is(err, ErrDuplicateISBN) {
		shared.WriteError(w, http.StatusConflict, "book already in your library")
		return
	}
	if err != nil {
		log.Error().Err(err).Msg("add manual book failed")
		shared.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}
	shared.WriteJSON(w, http.StatusCreated, toResponse(book))
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	books, err := h.service.List(r.Context(), r.URL.Query().Get("status"))
	if err != nil {
		log.Error().Err(err).Msg("list books failed")
		shared.WriteError(w, http.StatusInternalServerError, "failed to list books")
		return
	}
	shared.WriteJSON(w, http.StatusOK, toResponses(books))
}

func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "bookId")
	book, err := h.service.Get(r.Context(), id)
	if errors.Is(err, ErrNotFound) {
		shared.WriteError(w, http.StatusNotFound, "book not found")
		return
	}
	if err != nil {
		log.Error().Err(err).Str("book_id", id).Msg("get book failed")
		shared.WriteError(w, http.StatusInternalServerError, "failed to get book")
		return
	}
	shared.WriteJSON(w, http.StatusOK, toResponse(book))
}

func (h *Handler) UpdateStatus(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "bookId")
	var req UpdateStatusRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		shared.WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	err := h.service.UpdateStatus(r.Context(), id, BookStatus(req.Status))
	if errors.Is(err, ErrNotFound) {
		shared.WriteError(w, http.StatusNotFound, "book not found")
		return
	}
	if err != nil {
		shared.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "bookId")
	err := h.service.Delete(r.Context(), id)
	if errors.Is(err, ErrNotFound) {
		shared.WriteError(w, http.StatusNotFound, "book not found")
		return
	}
	if err != nil {
		log.Error().Err(err).Str("book_id", id).Msg("delete book failed")
		shared.WriteError(w, http.StatusInternalServerError, "failed to delete book")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
