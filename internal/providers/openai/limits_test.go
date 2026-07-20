package openai

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	provideroauth "github.com/vitaliiPsl/crappy-ai/internal/providers/oauth"
)

func TestLimitsFetchesAndConvertsCodexUsage(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if req.URL.Path != "/backend-api/wham/usage" {
			t.Errorf("path = %q, want /backend-api/wham/usage", req.URL.Path)
		}

		if req.Header.Get("Authorization") != "Bearer access" {
			t.Errorf("Authorization = %q", req.Header.Get("Authorization"))
		}

		if req.Header.Get("ChatGPT-Account-Id") != "account" {
			t.Errorf("ChatGPT-Account-Id = %q", req.Header.Get("ChatGPT-Account-Id"))
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprint(w, `{
			"plan_type":"plus",
			"rate_limit":{
				"primary_window":{"used_percent":42,"limit_window_seconds":18000,"reset_at":2000000000},
				"secondary_window":{"used_percent":15,"limit_window_seconds":604800,"reset_at":2000000100}
			},
			"additional_rate_limits":[{
				"limit_name":"Other model",
				"rate_limit":{"primary_window":{"used_percent":70,"limit_window_seconds":900,"reset_at":2000000200}}
			}]
		}`)
	}))
	defer server.Close()

	provider := New()
	provider.httpClient = server.Client()

	limits, err := provider.Limits(
		context.Background(),
		provideroauth.Authorization{
			BearerToken: "access",
			Headers:     map[string]string{"ChatGPT-Account-Id": "account"},
		},
		provideroauth.Config{LimitsURL: server.URL + "/backend-api/wham/usage"},
	)
	if err != nil {
		t.Fatalf("Limits() error = %v", err)
	}

	if limits.Plan != "plus" || len(limits.Snapshots) != 2 {
		t.Fatalf("Limits() = %+v", limits)
	}

	primary := limits.Snapshots[0]
	if len(primary.Windows) != 2 || primary.Windows[0].Duration != 5*time.Hour {
		t.Fatalf("primary snapshot = %+v", primary)
	}

	additional := limits.Snapshots[1]
	if additional.Name != "Other model" || len(additional.Windows) != 1 {
		t.Fatalf("additional snapshot = %+v", additional)
	}
}
