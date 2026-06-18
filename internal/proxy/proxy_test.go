package proxy

import (
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hashir500/Fuse/internal/config"
	"github.com/hashir500/Fuse/internal/store"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) {
	return f(r)
}

func TestProxyForwardsAuthHeaders(t *testing.T) {
	tests := []struct {
		name        string
		path        string
		body        string
		headerName  string
		headerValue string
		response    string
	}{
		{
			name:        "anthropic x-api-key",
			path:        "/v1/messages",
			body:        `{"model":"claude-sonnet-4-20250514","max_tokens":20,"messages":[{"role":"user","content":"hello"}]}`,
			headerName:  "x-api-key",
			headerValue: "sk-ant-test",
			response:    `{"model":"claude-sonnet-4-20250514","usage":{"input_tokens":1,"output_tokens":3}}`,
		},
		{
			name:        "openai authorization",
			path:        "/v1/chat/completions",
			body:        `{"model":"gpt-4.1","max_tokens":20,"messages":[{"role":"user","content":"hello"}]}`,
			headerName:  "authorization",
			headerValue: "Bearer sk-test",
			response:    `{"model":"gpt-4.1","usage":{"prompt_tokens":1,"completion_tokens":3,"total_tokens":4}}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, err := store.Open(filepath.Join(t.TempDir(), "spend.db"))
			if err != nil {
				t.Fatal(err)
			}
			defer db.Close()

			var upstreamHeader string
			server := &Server{
				Config: testConfig(),
				Store:  db,
				Stderr: io.Discard,
				Client: &http.Client{
					Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
						upstreamHeader = r.Header.Get(tt.headerName)
						return &http.Response{
							StatusCode: http.StatusOK,
							Header:     http.Header{"Content-Type": []string{"application/json"}},
							Body:       io.NopCloser(strings.NewReader(tt.response)),
							Request:    r,
						}, nil
					}),
				},
			}

			req := httptest.NewRequest(http.MethodPost, "http://localhost:8787"+tt.path, strings.NewReader(tt.body))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set(tt.headerName, tt.headerValue)

			rec := httptest.NewRecorder()
			server.ServeHTTP(rec, req)

			if rec.Code != http.StatusOK {
				t.Fatalf("status = %d, want %d; body=%s", rec.Code, http.StatusOK, rec.Body.String())
			}
			if upstreamHeader != tt.headerValue {
				t.Fatalf("upstream %s = %q, want %q", tt.headerName, upstreamHeader, tt.headerValue)
			}
		})
	}
}

func testConfig() *config.Config {
	return &config.Config{
		Providers: map[string]config.ProviderConfig{
			"anthropic": {
				BaseURL: "https://api.anthropic.com",
				Models: map[string]config.ModelCosts{
					"claude-sonnet-4-20250514": {
						InputCostPer1K:  0.003,
						OutputCostPer1K: 0.015,
					},
				},
			},
			"openai": {
				BaseURL: "https://api.openai.com",
				Models: map[string]config.ModelCosts{
					"gpt-4.1": {
						InputCostPer1K:  0.002,
						OutputCostPer1K: 0.008,
					},
				},
			},
		},
		Budgets: config.BudgetConfig{
			Daily:   config.Budget{Hard: 100},
			Weekly:  config.Budget{Hard: 500},
			Monthly: config.Budget{Hard: 1000},
		},
		Estimation: config.EstimationConfig{Mode: "max", OutputRatio: 0.3},
		OnHardCap:  "block",
	}
}
