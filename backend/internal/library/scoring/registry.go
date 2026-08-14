// Package scoring implements configurable rating-provider adapters for the
// library module. It follows ADR 007: thin, vendor-scoped HTTP clients behind
// the library.ScoreLookupClient consumer-side interface. Which provider serves
// which media type is decided at startup from shared.ScoreProviderConfig, so
// swapping providers (or adding new ones) never touches the library module.
package scoring

import (
	"context"
	"fmt"
	"sort"

	"github.com/diegobraga92/pudimproductivity/backend/internal/library"
)

// ProviderConfig is the subset of shared.ScoreProviderConfig that a single
// provider instance needs.
type ProviderConfig struct {
	Name    string
	APIKey  string
	BaseURL string
}

// Constructor builds a provider client from its config. It must return a
// non-nil error when required settings (e.g. the API key) are missing.
type Constructor func(ctx context.Context, cfg ProviderConfig) (library.ScoreLookupClient, error)

// registry holds every built-in provider constructor keyed by provider name.
// Adding a provider = one entry here, nothing else changes.
var registry = map[string]Constructor{
	"omdb": NewOMDB,
	"rawg": NewRAWG,
}

// capabilities declares which media types each provider can serve. Used to
// reject configurations that assign a provider to an unsupported media type.
var capabilities = map[string][]library.MediaType{
	"omdb": {library.MediaTypeMovie, library.MediaTypeSeries},
	"rawg": {library.MediaTypeGame},
}

// Names returns the sorted list of registered provider names.
func Names() []string {
	out := make([]string, 0, len(registry))
	for name := range registry {
		out = append(out, name)
	}
	sort.Strings(out)
	return out
}

// buildClient constructs a provider client, validating the provider name.
func buildClient(ctx context.Context, name string, cfg ProviderConfig) (library.ScoreLookupClient, error) {
	ctor, ok := registry[name]
	if !ok {
		return nil, fmt.Errorf("unknown score provider %q (known: %v)", name, Names())
	}
	return ctor(ctx, cfg)
}

// supports reports whether the named provider can serve the given media type.
func supports(name string, mt library.MediaType) bool {
	for _, supported := range capabilities[name] {
		if supported == mt {
			return true
		}
	}
	return false
}
