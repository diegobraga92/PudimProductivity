// Package scoring manages the library's rating-provider configuration
// (which provider serves each media type, plus per-provider API keys) from the
// database so it can be edited at runtime through the admin UI instead of
// environment variables.
package scoring

import (
	"time"

	"github.com/diegobraga92/pudimproductivity/backend/internal/contexts/content/library"
)

// Provider is a single registered score provider with its stored configuration.
// api_key is a secret: it is never returned by the API and is excluded from
// backups.
type Provider struct {
	Name    string
	APIKey  string
	BaseURL string
}

// Config is the persisted media-type → provider mapping. saved_at is nil until
// the user explicitly saves via the admin UI; while nil the service overlays
// environment defaults so existing .env-based deployments keep working.
type Config struct {
	MovieProvider  string
	SeriesProvider string
	GameProvider   string
	BookProvider   string
	SavedAt        *time.Time
}

// ProviderAPI is the masked, client-facing view of a provider.
type ProviderAPI struct {
	Name           string   `json:"name"`
	BaseURL        string   `json:"base_url"`
	APIKeySet      bool     `json:"api_key_set"`
	SupportedTypes []string `json:"supported_types"`
}

// ConfigAPI is the full response for GET /api/v1/admin/score-providers.
type ConfigAPI struct {
	MovieProvider  string        `json:"movie_provider"`
	SeriesProvider string        `json:"series_provider"`
	GameProvider   string        `json:"game_provider"`
	BookProvider   string        `json:"book_provider"`
	LookupEnabled  bool          `json:"lookup_enabled"`
	Providers      []ProviderAPI `json:"providers"`
}

// mediaTypeProviderField maps each library media type to its config field.
func providerForMediaType(cfg Config, mt library.MediaType) string {
	switch mt {
	case library.MediaTypeMovie:
		return cfg.MovieProvider
	case library.MediaTypeSeries:
		return cfg.SeriesProvider
	case library.MediaTypeGame:
		return cfg.GameProvider
	case library.MediaTypeBook:
		return cfg.BookProvider
	default:
		return ""
	}
}
