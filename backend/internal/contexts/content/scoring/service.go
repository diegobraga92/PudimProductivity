package scoring

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/diegobraga92/pudimproductivity/backend/internal/contexts/audit"
	"github.com/diegobraga92/pudimproductivity/backend/internal/contexts/content/library"
	"github.com/diegobraga92/pudimproductivity/backend/internal/infrastructure/featureflag"
	"github.com/diegobraga92/pudimproductivity/backend/internal/infrastructure/scoringapi"
	"github.com/diegobraga92/pudimproductivity/backend/internal/platform/config"
)

// Reloader rebuilds the runtime score-lookup client from a config.
type Reloader interface {
	Reload(ctx context.Context, cfg config.ScoreProviderConfig) error
}

// Service coordinates the score-provider configuration.
type Service struct {
	repo   Repository
	flags  *featureflag.Service
	reload Reloader
	audit  audit.Logger
	env    config.ScoreProviderConfig
	mu     sync.Mutex // serializes read/update so the effective config stays consistent
}

func NewService(repo Repository, flags *featureflag.Service, reload Reloader, auditLogger audit.Logger, env config.ScoreProviderConfig) *Service {
	if auditLogger == nil {
		auditLogger = audit.NoopLogger{}
	}
	return &Service{repo: repo, flags: flags, reload: reload, audit: auditLogger, env: env}
}

var mediaTypes = []library.MediaType{
	library.MediaTypeMovie,
	library.MediaTypeSeries,
	library.MediaTypeGame,
	library.MediaTypeBook,
}

func trim(v string) string { return strings.TrimSpace(v) }

// TODO: Check about removing env code after full migration to using DB

// envAssignment returns the provider selected for a media type by the
// environment bootstrap config ("" = disabled).
func envAssignment(env config.ScoreProviderConfig, mt library.MediaType) string {
	switch mt {
	case library.MediaTypeMovie:
		return env.Movie
	case library.MediaTypeSeries:
		return env.Series
	case library.MediaTypeGame:
		return env.Game
	case library.MediaTypeBook:
		return env.Book
	default:
		return ""
	}
}

// mergeEnv overlays environment bootstrap defaults onto the persisted config and provider rows.
func mergeEnv(cfg Config, env config.ScoreProviderConfig, dbProviders []Provider) (Config, []Provider) {
	if cfg.SavedAt != nil {
		return cfg, dbProviders
	}
	providers := make([]Provider, len(dbProviders))
	copy(providers, dbProviders)
	byName := make(map[string]int, len(providers))
	for i := range providers {
		byName[providers[i].Name] = i
	}
	for _, mt := range mediaTypes {
		if providerForMediaType(cfg, mt) == "" {
			if envName := envAssignment(env, mt); envName != "" {
				switch mt {
				case library.MediaTypeMovie:
					cfg.MovieProvider = envName
				case library.MediaTypeSeries:
					cfg.SeriesProvider = envName
				case library.MediaTypeGame:
					cfg.GameProvider = envName
				case library.MediaTypeBook:
					cfg.BookProvider = envName
				}
			}
		}
	}
	for name, key := range env.Keys {
		if i, ok := byName[name]; ok && key != "" && providers[i].APIKey == "" {
			providers[i].APIKey = key
		}
	}
	for name, baseURL := range env.BaseURLs {
		if i, ok := byName[name]; ok && baseURL != "" && providers[i].BaseURL == "" {
			providers[i].BaseURL = baseURL
		}
	}
	for name, settings := range env.Settings {
		i, ok := byName[name]
		if !ok {
			continue
		}
		if providers[i].Settings == nil {
			providers[i].Settings = make(map[string]string)
		}
		for k, v := range settings {
			if v != "" && providers[i].Settings[k] == "" {
				providers[i].Settings[k] = v
			}
		}
	}
	return cfg, providers
}

func buildScoreProviderConfig(cfg Config, providers []Provider) (config.ScoreProviderConfig, error) {
	keys := make(map[string]string, len(providers))
	baseURLs := make(map[string]string, len(providers))
	settings := make(map[string]map[string]string, len(providers))
	known := make(map[string]bool, len(providers))
	for _, p := range providers {
		known[p.Name] = true
		keys[p.Name] = p.APIKey
		baseURLs[p.Name] = p.BaseURL
		settings[p.Name] = p.Settings
	}

	out := config.ScoreProviderConfig{
		Movie:    cfg.MovieProvider,
		Series:   cfg.SeriesProvider,
		Game:     cfg.GameProvider,
		Book:     cfg.BookProvider,
		Keys:     keys,
		BaseURLs: baseURLs,
		Settings: settings,
	}

	for _, mt := range mediaTypes {
		name := providerForMediaType(cfg, mt)
		switch name {
		case "", "none":
			continue
		}
		if !known[name] {
			return config.ScoreProviderConfig{}, fmt.Errorf("unknown score provider %q (known: %v)", name, scoringapi.Names())
		}
		if !scoringapi.Supports(name, mt) {
			return config.ScoreProviderConfig{}, fmt.Errorf("score provider %q does not support media type %q", name, mt)
		}
		if keys[name] == "" {
			return config.ScoreProviderConfig{}, fmt.Errorf("score provider %q for %q requires an API key", name, mt)
		}
	}
	return out, nil
}

// ApplyConfig computes the effective config and pushes it into the runtime lookup.
func (s *Service) ApplyConfig(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	cfg, providers, err := s.loadEffective(ctx)
	if err != nil {
		return err
	}
	scoreCfg, err := buildScoreProviderConfig(cfg, providers)
	if err != nil {
		return err
	}
	if s.reload != nil {
		return s.reload.Reload(ctx, scoreCfg)
	}
	return nil
}

// Config returns the effective, client-facing configuration (masked).
func (s *Service) Config(ctx context.Context) (ConfigAPI, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	cfg, providers, err := s.loadEffective(ctx)
	if err != nil {
		return ConfigAPI{}, err
	}
	enabled := false
	if s.flags != nil {
		if v, err := s.flags.IsEnabled(ctx, library.ScoreLookupFeatureFlag); err == nil {
			enabled = v
		}
	}
	return s.toAPI(cfg, providers, enabled), nil
}

// Update applies a new configuration, after validating it.
func (s *Service) Update(ctx context.Context, req UpdateConfigRequest) (ConfigAPI, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	oldCfg, err := s.repo.GetConfig(ctx)
	if err != nil {
		return ConfigAPI{}, err
	}
	oldProviders, err := s.repo.GetProviders(ctx)
	if err != nil {
		return ConfigAPI{}, err
	}

	// nil APIKey/BaseURL mean "keep", "" means "clear". Settings merge per key:
	// present keys are upserted, "" clears, absent keys are kept.
	byName := make(map[string]Provider, len(oldProviders))
	for _, p := range oldProviders {
		byName[p.Name] = p
	}
	for _, up := range req.Providers {
		p, known := byName[up.Name]
		if !known {
			return ConfigAPI{}, fmt.Errorf("unknown score provider %q (known: %v)", up.Name, scoringapi.Names())
		}
		if up.APIKey != nil {
			p.APIKey = trim(*up.APIKey)
		}
		if up.BaseURL != nil {
			p.BaseURL = trim(*up.BaseURL)
		}
		for k, v := range up.Settings {
			if p.Settings == nil {
				p.Settings = make(map[string]string)
			}
			p.Settings[k] = trim(v)
		}
		byName[up.Name] = p
	}
	newProviders := make([]Provider, 0, len(byName))
	for _, p := range byName {
		newProviders = append(newProviders, p)
	}
	sort.Slice(newProviders, func(i, j int) bool { return newProviders[i].Name < newProviders[j].Name })

	now := time.Now().UTC()
	newCfg := Config{
		MovieProvider:  trim(req.MovieProvider),
		SeriesProvider: trim(req.SeriesProvider),
		GameProvider:   trim(req.GameProvider),
		BookProvider:   trim(req.BookProvider),
		SavedAt:        &now,
	}

	scoreCfg, err := buildScoreProviderConfig(newCfg, newProviders)
	if err != nil {
		return ConfigAPI{}, err
	}

	// Activate the new config first, then persist. A reload failure means the
	// config is invalid for the runtime and nothing is written.
	if s.reload != nil {
		if err := s.reload.Reload(ctx, scoreCfg); err != nil {
			return ConfigAPI{}, err
		}
	}
	if err := s.repo.SaveConfig(ctx, newCfg); err != nil {
		return ConfigAPI{}, err
	}
	for _, p := range newProviders {
		if err := s.repo.SaveProvider(ctx, p); err != nil {
			return ConfigAPI{}, err
		}
	}

	if req.LookupEnabled != nil && s.flags != nil {
		flag, err := s.flags.GetByName(ctx, library.ScoreLookupFeatureFlag)
		if err != nil {
			return ConfigAPI{}, err
		}
		if flag != nil {
			if err := s.flags.SetEnabled(ctx, flag.ID, *req.LookupEnabled); err != nil {
				return ConfigAPI{}, err
			}
		}
	}

	s.audit.Log(ctx, audit.ActionScoreProviderUpdated, audit.ResourceScoreProviders, "", masked(oldCfg, oldProviders), masked(newCfg, newProviders))

	enabled := false
	if req.LookupEnabled != nil {
		enabled = *req.LookupEnabled
	} else if s.flags != nil {
		if v, err := s.flags.IsEnabled(ctx, library.ScoreLookupFeatureFlag); err == nil {
			enabled = v
		}
	}
	return s.toAPI(newCfg, newProviders, enabled), nil
}

func (s *Service) loadEffective(ctx context.Context) (Config, []Provider, error) {
	cfg, err := s.repo.GetConfig(ctx)
	if err != nil {
		return Config{}, nil, err
	}
	providers, err := s.repo.GetProviders(ctx)
	if err != nil {
		return Config{}, nil, err
	}
	cfg, providers = mergeEnv(cfg, s.env, providers)
	return cfg, providers, nil
}

func (s *Service) toAPI(cfg Config, providers []Provider, enabled bool) ConfigAPI {
	api := ConfigAPI{
		MovieProvider:  providerForMediaType(cfg, library.MediaTypeMovie),
		SeriesProvider: providerForMediaType(cfg, library.MediaTypeSeries),
		GameProvider:   providerForMediaType(cfg, library.MediaTypeGame),
		BookProvider:   providerForMediaType(cfg, library.MediaTypeBook),
		LookupEnabled:  enabled,
		Providers:      make([]ProviderAPI, 0, len(providers)),
	}
	for _, p := range providers {
		types := make([]string, 0)
		for _, mt := range mediaTypes {
			if scoringapi.Supports(p.Name, mt) {
				types = append(types, string(mt))
			}
		}
		settingsKeys := scoringapi.SettingKeys(p.Name)
		settingsSet := make(map[string]bool, len(settingsKeys))
		for _, k := range settingsKeys {
			settingsSet[k] = p.Settings[k] != ""
		}
		api.Providers = append(api.Providers, ProviderAPI{
			Name:           p.Name,
			BaseURL:        p.BaseURL,
			APIKeySet:      p.APIKey != "",
			SettingsKeys:   settingsKeys,
			SettingsSet:    settingsSet,
			SupportedTypes: types,
		})
	}
	return api
}

// masked builds a secret-free view of a config for the audit log.
func masked(cfg Config, providers []Provider) map[string]any {
	prov := make(map[string]any, len(providers))
	for _, p := range providers {
		settingsSet := make(map[string]bool)
		for _, k := range scoringapi.SettingKeys(p.Name) {
			settingsSet[k] = p.Settings[k] != ""
		}
		prov[p.Name] = map[string]any{
			"api_key_set": p.APIKey != "",
			"base_url":    p.BaseURL,
			"settings":    settingsSet,
		}
	}
	return map[string]any{
		"movie_provider":  providerForMediaType(cfg, library.MediaTypeMovie),
		"series_provider": providerForMediaType(cfg, library.MediaTypeSeries),
		"game_provider":   providerForMediaType(cfg, library.MediaTypeGame),
		"book_provider":   providerForMediaType(cfg, library.MediaTypeBook),
		"providers":       prov,
	}
}
