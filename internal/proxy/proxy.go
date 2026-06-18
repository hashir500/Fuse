package proxy

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/hashir500/Fuse/internal/budget"
	"github.com/hashir500/Fuse/internal/config"
	"github.com/hashir500/Fuse/internal/cost"
	"github.com/hashir500/Fuse/internal/money"
	"github.com/hashir500/Fuse/internal/provider"
	"github.com/hashir500/Fuse/internal/store"
)

type Server struct {
	Config *config.Config
	Store  *store.Store
	Stderr io.Writer
	Client *http.Client
}

func (s *Server) ListenAndServe(addr string) error {
	if s.Client == nil {
		s.Client = &http.Client{Timeout: 10 * time.Minute}
	}
	return http.ListenAndServe(addr, s)
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if s.Stderr == nil {
		s.Stderr = io.Discard
	}
	providerName, ok := provider.Detect(r, s.Config)
	if !ok {
		http.Error(w, "fuse: unsupported provider route", http.StatusBadGateway)
		return
	}

	reqInfo, body, err := provider.PrepareRequest(r, providerName)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	preflightUsage := cost.PreflightUsage(reqInfo.Usage, s.Config.Estimation)
	estimatedCost, ok := s.estimateCost(reqInfo.Provider, reqInfo.Model, preflightUsage)
	if !ok {
		http.Error(w, fmt.Sprintf("fuse: model %q is not configured for provider %q", reqInfo.Model, reqInfo.Provider), http.StatusBadRequest)
		return
	}
	spend, err := s.Store.PeriodSpend(r.Context(), time.Now())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if os.Getenv("FUSE_DEBUG") == "1" {
		fmt.Fprintf(s.Stderr, "DEBUG: provider=%s model=%s estimate_mode=%s input=%d requested_output=%d estimated_output=%d estimated_cost=%.6f daily_hard=%.6f\n",
			reqInfo.Provider,
			reqInfo.Model,
			s.Config.Estimation.Mode,
			reqInfo.Usage.PromptTokens,
			reqInfo.Usage.CompletionTokens,
			preflightUsage.CompletionTokens,
			estimatedCost,
			s.Config.Budgets.Daily.Hard,
		)
	}
	decision := budget.Check(s.Config.Budgets, spend, estimatedCost, s.Config.OnHardCap)
	if !decision.Allowed {
		reqInfo.Usage = preflightUsage
		s.writeBlocked(w, r.Context(), reqInfo, estimatedCost, *decision.HardHit)
		return
	}
	for _, warning := range decision.SoftWarnings {
		fmt.Fprintf(s.Stderr, "SOFT CAP: %s spend (%s) exceeded. Current: %s\n", title(warning.Period), money.Dollars(warning.Soft), money.Dollars(warning.Spend))
	}

	target, err := provider.TargetURL(providerName, s.Config, r.URL)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	outReq := r.Clone(r.Context())
	outReq.URL = target
	outReq.Host = target.Host
	outReq.RequestURI = ""
	outReq.Header = r.Header.Clone()
	outReq.Body = io.NopCloser(bytes.NewReader(body))
	outReq.ContentLength = int64(len(body))
	provider.AddAuth(outReq, providerName, s.Config)

	resp, err := s.Client.Do(outReq)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}
	copyHeaders(w.Header(), resp.Header)
	w.WriteHeader(resp.StatusCode)
	_, _ = w.Write(respBody)

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		respInfo := provider.ParseResponse(providerName, respBody)
		if _, ok := s.Config.ModelCost(reqInfo.Provider, respInfo.Model); respInfo.Model != "" && ok {
			reqInfo.Model = respInfo.Model
		}
		if respInfo.Usage.TotalTokens > 0 || respInfo.Usage.PromptTokens > 0 || respInfo.Usage.CompletionTokens > 0 {
			reqInfo.Usage = respInfo.Usage
		}
		actualCost, _ := s.estimateCost(reqInfo.Provider, reqInfo.Model, reqInfo.Usage)
		_ = s.Store.LogRequest(context.Background(), store.RequestLog{
			Provider:         reqInfo.Provider,
			Model:            reqInfo.Model,
			PromptTokens:     reqInfo.Usage.PromptTokens,
			CompletionTokens: reqInfo.Usage.CompletionTokens,
			TotalTokens:      reqInfo.Usage.TotalTokens,
			EstimatedCost:    actualCost,
		})
	}
}

func (s *Server) estimateCost(providerName, model string, usage cost.Usage) (float64, bool) {
	modelCosts, ok := s.Config.ModelCost(providerName, model)
	if !ok {
		return 0, false
	}
	return cost.Estimate(usage, modelCosts), true
}

func (s *Server) writeBlocked(w http.ResponseWriter, ctx context.Context, reqInfo provider.RequestInfo, requestCost float64, hit budget.HardHit) {
	reason := hit.Message()
	_ = s.Store.LogRequest(ctx, store.RequestLog{
		Provider:         reqInfo.Provider,
		Model:            reqInfo.Model,
		PromptTokens:     reqInfo.Usage.PromptTokens,
		CompletionTokens: reqInfo.Usage.CompletionTokens,
		TotalTokens:      reqInfo.Usage.TotalTokens,
		EstimatedCost:    requestCost,
		WasBlocked:       true,
		BlockReason:      reason,
	})
	fmt.Fprintf(s.Stderr, "BLOCKED: %s hard cap (%s) would be exceeded\n   Estimated max request cost: %s | Current spend: %s\n",
		title(hit.Period), money.Dollars(hit.CapAmount), money.Dollars(hit.RequestCost), money.Dollars(hit.CurrentSpend))

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusTooManyRequests)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"error":         "fuse_hard_cap_exceeded",
		"message":       reason,
		"cap_type":      hit.Period,
		"cap_amount":    hit.CapAmount,
		"current_spend": hit.CurrentSpend,
	})
}

func copyHeaders(dst, src http.Header) {
	for key, values := range src {
		for _, value := range values {
			dst.Add(key, value)
		}
	}
}

func title(value string) string {
	if len(value) == 0 {
		return value
	}
	return strings.ToUpper(value[:1]) + value[1:]
}
