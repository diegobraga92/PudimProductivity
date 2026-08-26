// Package scoring implements configurable rating-provider adapters for the
// library module.
package scoringapi

import (
	"context"
	"fmt"
	"sort"

	"github.com/diegobraga92/pudimproductivity/backend/internal/contexts/content/library"
)

// ProviderConfig is the subset of config.ScoreProviderConfig that a single
// provider instance needs.
type ProviderConfig struct {
	Name     string
	APIKey   string
	BaseURL  string
	Settings map[string]string
}

// Constructor builds a provider client from its config.
type Constructor func(ctx context.Context, cfg ProviderConfig) (library.ScoreLookupClient, error)

// registry holds every built-in provider constructor keyed by provider name.
// Adding a provider = one entry here, nothing else changes.
var registry = map[string]Constructor{
	"omdb": NewOMDB,
	"rawg": NewRAWG,
	"igdb": NewIGDB,
}

// capabilities declares which media types each provider can serve. Used to
// reject configurations that assign a provider to an unsupported media type.
var capabilities = map[string][]library.MediaType{
	"omdb": {library.MediaTypeMovie, library.MediaTypeSeries},
	"rawg": {library.MediaTypeGame},
	"igdb": {library.MediaTypeGame},
}

// settingKeys declares the extra per-provider settings fields exposed by the
// admin API, beyond the generic api_key + base_url columns.
var settingKeys = map[string][]string{
	"igdb": {igdbClientSecretSetting},
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

// Supports is the exported form of supports, used by the score-provider
// settings module to validate media-type → provider assignments at save time.
func Supports(name string, mt library.MediaType) bool {
	return supports(name, mt)
}

// SettingKeys returns the extra settings fields a provider exposes in the admin
// UI (beyond api_key and base_url). Empty for providers that only need a single
// key.
func SettingKeys(name string) []string {
	return settingKeys[name]
}
