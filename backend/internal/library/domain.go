// Package library implements media tracking: movies, series, books and games
// are stored as simple items with a name, media type, release year, a done
// flag (consumed/read/watched/played) and optional notes. It replaces the
// book-specific booktrack module.
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
	// Score is an optional rating (0-100 scale: IMDb 8.7, Metacritic 95, ...).
	Score *float64
	// ScoreSource names where Score came from (e.g. "imdb", "metacritic").
	// Empty means no score is recorded.
	ScoreSource string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// ScoreSource is a canonical token for where an item's score came from. Any
// non-empty string is accepted so configurable providers can introduce new
// sources without a domain change; these constants document the known ones.
type ScoreSource string

const (
	ScoreSourceIMDb       ScoreSource = "imdb"
	ScoreSourceMetacritic ScoreSource = "metacritic"
	ScoreSourceTMDB       ScoreSource = "tmdb"
	ScoreSourceRAWG       ScoreSource = "rawg"
	ScoreSourceCustom     ScoreSource = "custom"
)

// NewItem validates and builds an item.
func NewItem(id, name string, mediaType MediaType, releaseYear *int, done bool, notes string, score *float64, scoreSource string) (*Item, error) {
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
		Score:       score,
		ScoreSource: scoreSource,
	}, nil
}
