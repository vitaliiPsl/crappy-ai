package openai

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	provideroauth "github.com/vitaliiPsl/crappy-ai/internal/providers/oauth"
)

type limitsResponse struct {
	PlanType             string                `json:"plan_type"`
	RateLimit            *rateLimitDetails     `json:"rate_limit"`
	AdditionalRateLimits []additionalRateLimit `json:"additional_rate_limits"`
}

type additionalRateLimit struct {
	LimitName string            `json:"limit_name"`
	RateLimit *rateLimitDetails `json:"rate_limit"`
}

type rateLimitDetails struct {
	PrimaryWindow   *rateLimitWindow `json:"primary_window"`
	SecondaryWindow *rateLimitWindow `json:"secondary_window"`
}

type rateLimitWindow struct {
	UsedPercent   float64 `json:"used_percent"`
	WindowSeconds int64   `json:"limit_window_seconds"`
	ResetAt       int64   `json:"reset_at"`
}

func (p *Provider) Limits(
	ctx context.Context,
	auth provideroauth.Authorization,
	config provideroauth.Config,
) (provideroauth.Limits, error) {
	if config.LimitsURL == "" {
		return provideroauth.Limits{}, fmt.Errorf("openai codex limits: limits_url is required")
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, config.LimitsURL, nil)
	if err != nil {
		return provideroauth.Limits{}, fmt.Errorf("openai codex limits: create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+auth.BearerToken)

	for name, value := range auth.Headers {
		req.Header.Set(name, value)
	}

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return provideroauth.Limits{}, fmt.Errorf("openai codex limits: request: %w", err)
	}

	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))

		return provideroauth.Limits{}, fmt.Errorf(
			"openai codex limits: request failed: %s: %s",
			resp.Status,
			strings.TrimSpace(string(body)),
		)
	}

	var payload limitsResponse
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return provideroauth.Limits{}, fmt.Errorf("openai codex limits: decode response: %w", err)
	}

	return convertLimits(payload), nil
}

func convertLimits(payload limitsResponse) provideroauth.Limits {
	var snapshots []provideroauth.LimitSnapshot
	if payload.RateLimit != nil {
		snapshots = append(snapshots, convertLimitSnapshot("", *payload.RateLimit))
	}

	for _, additional := range payload.AdditionalRateLimits {
		if additional.RateLimit == nil {
			continue
		}

		snapshots = append(snapshots, convertLimitSnapshot(
			additional.LimitName,
			*additional.RateLimit,
		))
	}

	return provideroauth.Limits{
		Plan:      payload.PlanType,
		Snapshots: snapshots,
	}
}

func convertLimitSnapshot(
	name string,
	details rateLimitDetails,
) provideroauth.LimitSnapshot {
	snapshot := provideroauth.LimitSnapshot{Name: name}
	for _, window := range []*rateLimitWindow{details.PrimaryWindow, details.SecondaryWindow} {
		if window != nil {
			snapshot.Windows = append(snapshot.Windows, convertLimitWindow(*window))
		}
	}

	return snapshot
}

func convertLimitWindow(window rateLimitWindow) provideroauth.LimitWindow {
	var resetsAt time.Time
	if window.ResetAt > 0 {
		resetsAt = time.Unix(window.ResetAt, 0)
	}

	return provideroauth.LimitWindow{
		UsedPercent: window.UsedPercent,
		Duration:    time.Duration(window.WindowSeconds) * time.Second,
		ResetsAt:    resetsAt,
	}
}
