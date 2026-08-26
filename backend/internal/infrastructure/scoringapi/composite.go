package scoringapi

import (
	"context"
	"fmt"

	"github.com/diegobraga92/pudimproductivity/backend/internal/contexts/content/library"
	"github.com/diegobraga92/pudimproductivity/backend/internal/platform/config"
)

// composite routes score searches to the provider configured for the query's
// media type. Media types without a configured provider return no candidates.
type composite struct {
	clients map[library.MediaType]library.ScoreLookupClient
}

func (c *composite) Search(ctx context.Context, query library.ScoreQuery) ([]library.ScoreCandidate, error) {
	client, ok := c.clients[query.MediaType]
	if !ok {
		return nil, nil
	}
	return client.Search(ctx, query)
}

// Configured reports whether at least one media type has a live provider
// client. Satisfies library.ScoreLookupProvider.
func (c *composite) Configured() bool { return len(c.clients) > 0 }

// NewComposite builds the per-media-type client set from config. Media types
// mapped to "" or "none" are left disabled.
func NewComposite(ctx context.Context, cfg config.ScoreProviderConfig) (library.ScoreLookupClient, error) {
	clients := make(map[library.MediaType]library.ScoreLookupClient)
	byType := map[library.MediaType]string{
		library.MediaTypeMovie:  cfg.Movie,
		library.MediaTypeSeries: cfg.Series,
		library.MediaTypeGame:   cfg.Game,
		library.MediaTypeBook:   cfg.Book,
	}
	for mt, name := range byType {
		switch name {
		case "", "none":
			continue
		}
		if !supports(name, mt) {
			return nil, fmt.Errorf("score provider %q does not support media type %q", name, mt)
		}
		client, err := buildClient(ctx, name, ProviderConfig{
			Name:     name,
			APIKey:   cfg.Keys[name],
			BaseURL:  cfg.BaseURLs[name],
			Settings: cfg.Settings[name],
		})
		if err != nil {
			return nil, fmt.Errorf("configure score provider %q for %q: %w", name, mt, err)
		}
		clients[mt] = client
	}
	if len(clients) == 0 {
		return library.NoopScoreLookup{}, nil
	}
	return &composite{clients: clients}, nil
}
