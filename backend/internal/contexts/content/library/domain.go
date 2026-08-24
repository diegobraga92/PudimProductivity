// Package library implements media tracking.
package library

import (
	"fmt"
	"strings"
	"time"
)

// MediaType is the kind of media an item represents.
type MediaType string

const (
	MediaTypeMovie  MediaType = "movie"
	MediaTypeSeries MediaType = "series"
	MediaTypeBook   MediaType = "book"
	MediaTypeGame   MediaType = "game"
)

// Valid reports whether t is a supported media type.
func (t MediaType) Valid() bool {
	switch t {
	case MediaTypeMovie, MediaTypeSeries, MediaTypeBook, MediaTypeGame:
		return true
	default:
		return false
	}
}

// Item is a single tracked piece of media.
type Item struct {
	ID          string
	Name        string
	MediaType   MediaType
	ReleaseYear *int
	Done        bool
	Notes       string
	Subtype     string
	// Score is an optional rating from a provider, like IMDB.
	Score       *float64
	ScoreSource string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// TODO: Check for other sources.

// ScoreSource is a canonical token for where an item's score came from.
type ScoreSource string

const (
	ScoreSourceIMDb       ScoreSource = "imdb"
	ScoreSourceMetacritic ScoreSource = "metacritic"
	ScoreSourceTMDB       ScoreSource = "tmdb"
	ScoreSourceRAWG       ScoreSource = "rawg"
	ScoreSourceCustom     ScoreSource = "custom"
)

// NewItem validates and builds an item.
func NewItem(id, name string, mediaType MediaType, releaseYear *int, done bool, notes string, score *float64, scoreSource, subtype string) (*Item, error) {
	if id == "" {
		return nil, fmt.Errorf("item id cannot be empty")
	}
	if strings.TrimSpace(name) == "" {
		return nil, fmt.Errorf("item name cannot be empty")
	}
	if !mediaType.Valid() {
		return nil, fmt.Errorf("invalid media type %q", mediaType)
	}
	if releaseYear != nil && (*releaseYear < 1800 || *releaseYear > 2100) {
		return nil, fmt.Errorf("release year %d out of range (1800-2100)", *releaseYear)
	}
	if score != nil && (*score < 0 || *score > 100) {
		return nil, fmt.Errorf("score %v out of range (0-100)", *score)
	}
	scoreSource = strings.TrimSpace(scoreSource)
	if score == nil && scoreSource != "" {
		return nil, fmt.Errorf("score source %q requires a score", scoreSource)
	}
	return &Item{
		ID:          id,
		Name:        strings.TrimSpace(name),
		MediaType:   mediaType,
		ReleaseYear: releaseYear,
		Done:        done,
		Notes:       notes,
		Subtype:     strings.TrimSpace(subtype),
		Score:       score,
		ScoreSource: scoreSource,
		CreatedAt:   time.Now().UTC(),
		UpdatedAt:   time.Now().UTC(),
	}, nil
}
