package command

import (
	"context"
	"fmt"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"

	provideroauth "github.com/vitaliiPsl/crappy-ai/internal/providers/oauth"
)

type ProviderLimitsSource interface {
	ProviderLimits(ctx context.Context, providerID string) (provideroauth.Limits, error)
}

type usageProvider struct {
	source     ProviderLimitsSource
	providerID string
}

func NewUsageProvider(source ProviderLimitsSource, providerID string) Provider {
	if source == nil || providerID == "" {
		return nil
	}

	return usageProvider{source: source, providerID: providerID}
}

func (p usageProvider) Commands(_ context.Context) []Command {
	return []Command{&UsageCommand{source: p.source, providerID: p.providerID}}
}

type UsageCommand struct {
	source     ProviderLimitsSource
	providerID string
}

func (c *UsageCommand) Definition() Definition {
	return Definition{Name: "usage", Description: "Show subscription usage limits"}
}

func (c *UsageCommand) Execute(ctx context.Context, _ Request) tea.Cmd {
	return func() tea.Msg {
		limits, err := c.source.ProviderLimits(ctx, c.providerID)
		if err != nil {
			return SystemMsg{Text: fmt.Sprintf("Usage unavailable: %v", err)}
		}

		return SystemMsg{Text: formatLimits(limits)}
	}
}

func formatLimits(limits provideroauth.Limits) string {
	var b strings.Builder
	if limits.Plan != "" {
		fmt.Fprintf(&b, "%s plan\n", strings.ToUpper(limits.Plan[:1])+limits.Plan[1:])
	}

	for _, snapshot := range limits.Snapshots {
		if snapshot.Name != "" {
			fmt.Fprintf(&b, "%s\n", snapshot.Name)
		}

		for _, window := range snapshot.Windows {
			fmt.Fprintf(&b, "  %s: %.0f%% used", formatWindow(window.Duration), window.UsedPercent)

			if !window.ResetsAt.IsZero() {
				fmt.Fprintf(&b, " · resets in %s", formatReset(window.ResetsAt))
			}

			b.WriteByte('\n')
		}
	}

	result := strings.TrimSpace(b.String())
	if result == "" {
		return "No subscription usage information available"
	}

	return result
}

func formatWindow(window time.Duration) string {
	if window <= 0 {
		return "Usage"
	}

	hours := int(window.Hours())
	if hours%24 == 0 {
		days := hours / 24

		return fmt.Sprintf("%d day%s", days, plural(days))
	}

	return fmt.Sprintf("%d hour%s", hours, plural(hours))
}

func formatReset(reset time.Time) string {
	remaining := time.Until(reset).Round(time.Minute)
	if remaining <= 0 {
		return "now"
	}

	days := int(remaining / (24 * time.Hour))
	remaining -= time.Duration(days) * 24 * time.Hour
	hours := int(remaining / time.Hour)
	remaining -= time.Duration(hours) * time.Hour
	minutes := int(remaining / time.Minute)

	parts := make([]string, 0, 2)
	if days > 0 {
		parts = append(parts, fmt.Sprintf("%dd", days))
	}

	if hours > 0 {
		parts = append(parts, fmt.Sprintf("%dh", hours))
	}

	if days == 0 && minutes > 0 {
		parts = append(parts, fmt.Sprintf("%dm", minutes))
	}

	if len(parts) == 0 {
		return "less than a minute"
	}

	return strings.Join(parts, " ")
}

func plural(value int) string {
	if value == 1 {
		return ""
	}

	return "s"
}
