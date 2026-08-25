package scoring

import (
	"context"
	"testing"
	"time"

	"github.com/diegobraga92/pudimproductivity/backend/internal/platform/config"
)

type fakeRepo struct {
	cfg            Config
	providers      []Provider
	savedCfg       *Config
	savedProviders []Provider
	getErr         error
}

func (f *fakeRepo) GetConfig(ctx context.Context) (Config, error) {
	return f.cfg, f.getErr
}

func (f *fakeRepo) SaveConfig(ctx context.Context, cfg Config) error {
	if f.getErr != nil {
		return f.getErr
	}
	f.cfg = cfg
	f.savedCfg = &cfg
	return nil
}

func (f *fakeRepo) GetProviders(ctx context.Context) ([]Provider, error) {
	return f.providers, f.getErr
}

func (f *fakeRepo) SaveProvider(ctx context.Context, p Provider) error {
	if f.getErr != nil {
		return f.getErr
	}
	for i := range f.providers {
		if f.providers[i].Name == p.Name {
			f.providers[i] = p
			return nil
		}
	}
	f.providers = append(f.providers, p)
	f.savedProviders = append(f.savedProviders, p)
	return nil
}

type fakeReloader struct {
	last config.ScoreProviderConfig
	err  error
}

func (f *fakeReloader) Reload(_ context.Context, cfg config.ScoreProviderConfig) error {
	if f.err != nil {
		return f.err
	}
	f.last = cfg
	return nil
}

func strPtr(s string) *string { return &s }
func boolPtr(b bool) *bool    { return &b }

func seededRepo() *fakeRepo {
	return &fakeRepo{
		providers: []Provider{{Name: "omdb"}, {Name: "rawg"}},
	}
}

func TestMergeEnv_OverlaysWhenNeverSaved(t *testing.T) {
	env := config.ScoreProviderConfig{
		Game:     "rawg",
		Keys:     map[string]string{"rawg": "env-key"},
		BaseURLs: map[string]string{"rawg": "https://rawg.example"},
	}
	cfg, providers := mergeEnv(Config{}, env, []Provider{{Name: "omdb"}, {Name: "rawg"}})

	if cfg.GameProvider != "rawg" || cfg.MovieProvider != "" {
		t.Fatalf("mergeEnv(unsaved) assignments = movie %q game %q", cfg.MovieProvider, cfg.GameProvider)
	}
	got := map[string]Provider{}
	for _, p := range providers {
		got[p.Name] = p
	}
	if got["rawg"].APIKey != "env-key" || got["rawg"].BaseURL != "https://rawg.example" {
		t.Fatalf("mergeEnv(unsaved) rawg = %+v, want env key/base", got["rawg"])
	}
	if got["omdb"].APIKey != "" {
		t.Fatalf("mergeEnv(unsaved) omdb key = %q, want empty", got["omdb"].APIKey)
	}
}

func TestMergeEnv_IgnoresEnvAfterSaved(t *testing.T) {
	now := time.Now()
	cfg := Config{SavedAt: &now} // explicitly saved
	env := config.ScoreProviderConfig{
		Game: "rawg",
		Keys: map[string]string{"rawg": "env-key"},
	}
	merged, providers := mergeEnv(cfg, env, []Provider{{Name: "rawg"}})
	if merged.GameProvider != "" {
		t.Fatalf("mergeEnv(saved) game provider = %q, want empty", merged.GameProvider)
	}
	if providers[0].APIKey != "" {
		t.Fatalf("mergeEnv(saved) rawg key = %q, want empty", providers[0].APIKey)
	}
}

func TestBuildScoreProviderConfig_Valid(t *testing.T) {
	cfg := Config{GameProvider: "rawg", MovieProvider: "none"}
	providers := []Provider{{Name: "omdb"}, {Name: "rawg", APIKey: "k"}}
	out, err := buildScoreProviderConfig(cfg, providers)
	if err != nil {
		t.Fatalf("buildScoreProviderConfig: %v", err)
	}
	if out.Game != "rawg" || out.Movie != "none" || out.Keys["rawg"] != "k" {
		t.Fatalf("unexpected config: %+v", out)
	}
}

func TestBuildScoreProviderConfig_Rejects(t *testing.T) {
	t.Run("unknown provider", func(t *testing.T) {
		_, err := buildScoreProviderConfig(
			Config{MovieProvider: "netflix"},
			[]Provider{{Name: "omdb", APIKey: "k"}, {Name: "rawg", APIKey: "k"}},
		)
		if err == nil {
			t.Fatal("expected error for unknown provider")
		}
	})

	t.Run("capability mismatch", func(t *testing.T) {
		_, err := buildScoreProviderConfig(
			Config{MovieProvider: "rawg"},
			[]Provider{{Name: "omdb", APIKey: "k"}, {Name: "rawg", APIKey: "k"}},
		)
		if err == nil {
			t.Fatal("expected error for provider/media-type mismatch")
		}
	})

	t.Run("missing api key", func(t *testing.T) {
		_, err := buildScoreProviderConfig(
			Config{GameProvider: "rawg"},
			[]Provider{{Name: "omdb", APIKey: "k"}, {Name: "rawg"}},
		)
		if err == nil {
			t.Fatal("expected error for missing API key")
		}
	})
}

func TestService_ApplyConfig_UsesEnvBootstrap(t *testing.T) {
	repo := seededRepo()
	env := config.ScoreProviderConfig{
		Game: "rawg",
		Keys: map[string]string{"rawg": "env-key"},
	}
	reloader := &fakeReloader{}
	svc := NewService(repo, nil, reloader, nil, env)

	if err := svc.ApplyConfig(context.Background()); err != nil {
		t.Fatalf("ApplyConfig: %v", err)
	}
	if reloader.last.Game != "rawg" || reloader.last.Keys["rawg"] != "env-key" {
		t.Fatalf("ApplyConfig did not apply env bootstrap: %+v", reloader.last)
	}
}

func TestService_ApplyConfig_InvalidEnvKeepsDegraded(t *testing.T) {
	repo := seededRepo()
	env := config.ScoreProviderConfig{Movie: "netflix", Keys: map[string]string{"netflix": "k"}}
	reloader := &fakeReloader{}
	svc := NewService(repo, nil, reloader, nil, env)

	if err := svc.ApplyConfig(context.Background()); err == nil {
		t.Fatal("expected error for unknown provider in env")
	}
	if reloader.last.Game != "" {
		t.Fatalf("unexpected reload on invalid env: %+v", reloader.last)
	}
}

func TestService_Config_MasksKeys(t *testing.T) {
	repo := &fakeRepo{
		cfg:       Config{GameProvider: "rawg"},
		providers: []Provider{{Name: "rawg", APIKey: "super-secret", BaseURL: "https://rawg.io/api"}},
	}
	svc := NewService(repo, nil, &fakeReloader{}, nil, config.ScoreProviderConfig{})

	api, err := svc.Config(context.Background())
	if err != nil {
		t.Fatalf("Config: %v", err)
	}
	if api.GameProvider != "rawg" {
		t.Fatalf("GameProvider = %q, want rawg", api.GameProvider)
	}
	if len(api.Providers) != 1 || api.Providers[0].Name != "rawg" || !api.Providers[0].APIKeySet {
		t.Fatalf("Providers = %+v, want masked rawg with api_key_set=true", api.Providers)
	}
	if api.Providers[0].BaseURL != "https://rawg.io/api" {
		t.Fatalf("BaseURL = %q", api.Providers[0].BaseURL)
	}
}
