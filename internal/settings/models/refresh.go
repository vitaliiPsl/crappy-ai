package models

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"slices"
	"sort"
	"time"

	"github.com/vitaliiPsl/crappy-adk/kit"
)

const (
	modelsDevURL = "https://models.dev/api.json"
	fetchTimeout = 10 * time.Second
)

type upstreamProvider struct {
	Models map[string]upstreamModel `json:"models"`
}

type upstreamModel struct {
	ID         string         `json:"id"`
	Reasoning  bool           `json:"reasoning"`
	ToolCall   bool           `json:"tool_call"`
	Modalities upstreamModes  `json:"modalities"`
	Cost       *upstreamCost  `json:"cost"`
	Limit      upstreamLimits `json:"limit"`
	Knowledge  string         `json:"knowledge"`
	Release    string         `json:"release_date"`
}

type upstreamModes struct {
	Input  []string `json:"input"`
	Output []string `json:"output"`
}

type upstreamCost struct {
	Input      float64 `json:"input"`
	Output     float64 `json:"output"`
	CacheRead  float64 `json:"cache_read"`
	CacheWrite float64 `json:"cache_write"`
}

type upstreamLimits struct {
	Context int `json:"context"`
	Input   int `json:"input"`
	Output  int `json:"output"`
}

func Refresh(ctx context.Context, path string, providers []ProviderSettings) error {
	if path == "" {
		return nil
	}

	upstream, err := fetchModelsDev(ctx)
	if err != nil {
		return err
	}

	out := filterCatalog(providers, upstream)
	if len(out) == 0 {
		return nil
	}

	return writeModels(path, out)
}

func filterCatalog(providers []ProviderSettings, upstream map[string]upstreamProvider) map[string][]kit.ModelConfig {
	out := make(map[string][]kit.ModelConfig, len(providers))

	for _, p := range providers {
		src, ok := upstream[p.API]
		if !ok {
			continue
		}

		curatedOrder := make(map[string]int, len(p.Models))
		for i, m := range p.Models {
			curatedOrder[m.ID] = i
		}

		models := make([]kit.ModelConfig, 0, len(src.Models))
		for id, m := range src.Models {
			models = append(models, normalizeUpstreamModel(p.Name, id, m))
		}

		sort.Slice(models, func(i, j int) bool {
			oi, iok := curatedOrder[models[i].ID]
			oj, jok := curatedOrder[models[j].ID]

			switch {
			case iok && jok:
				return oi < oj
			case iok:
				return true
			case jok:
				return false
			}

			if models[i].ReleaseDate != models[j].ReleaseDate {
				return models[i].ReleaseDate > models[j].ReleaseDate
			}

			return models[i].ID < models[j].ID
		})

		if len(models) > 0 {
			out[p.Name] = models
		}
	}

	return out
}

func fetchModelsDev(ctx context.Context) (map[string]upstreamProvider, error) {
	reqCtx, cancel := context.WithTimeout(ctx, fetchTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(reqCtx, http.MethodGet, modelsDevURL, nil)
	if err != nil {
		return nil, fmt.Errorf("build models.dev request: %w", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch models.dev catalog: %w", err)
	}

	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("fetch models.dev catalog: unexpected status %s", resp.Status)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read models.dev catalog: %w", err)
	}

	var catalog map[string]upstreamProvider
	if err := json.Unmarshal(body, &catalog); err != nil {
		return nil, fmt.Errorf("decode models.dev catalog: %w", err)
	}

	return catalog, nil
}

func normalizeUpstreamModel(provider, id string, u upstreamModel) kit.ModelConfig {
	cfg := kit.ModelConfig{
		ID:              id,
		Provider:        provider,
		ContextWindow:   u.Limit.Context,
		InputLimit:      u.Limit.Input,
		OutputLimit:     u.Limit.Output,
		KnowledgeCutoff: normalizeUpstreamDate(u.Knowledge),
		ReleaseDate:     normalizeUpstreamDate(u.Release),
		Capabilities: kit.ModelCapabilities{
			Text:      hasModality(u.Modalities, "text"),
			Image:     hasModality(u.Modalities, "image"),
			Audio:     hasModality(u.Modalities, "audio"),
			Video:     hasModality(u.Modalities, "video"),
			PDF:       hasModality(u.Modalities, "pdf"),
			Tools:     u.ToolCall,
			Streaming: true,
			Reasoning: u.Reasoning,
			Batch:     true,
		},
	}

	if cfg.InputLimit == 0 {
		cfg.InputLimit = cfg.ContextWindow
	}

	if u.Cost != nil {
		cfg.Cost = kit.ModelCost{
			Input:      u.Cost.Input,
			Output:     u.Cost.Output,
			CacheRead:  u.Cost.CacheRead,
			CacheWrite: u.Cost.CacheWrite,
		}
		cfg.Capabilities.Caching = u.Cost.CacheRead > 0 || u.Cost.CacheWrite > 0
	}

	return cfg
}

func hasModality(m upstreamModes, needle string) bool {
	return slices.Contains(m.Input, needle) || slices.Contains(m.Output, needle)
}

func normalizeUpstreamDate(raw string) string {
	if raw == "" {
		return ""
	}

	for _, layout := range []string{"2006-01-02", "2006-01"} {
		if t, err := time.Parse(layout, raw); err == nil {
			return t.Format("2006-01-02")
		}
	}

	if t, err := time.Parse(time.RFC3339, raw); err == nil {
		return t.UTC().Format("2006-01-02")
	}

	return raw
}
