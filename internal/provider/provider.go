package provider

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"

	"github.com/hashir500/Fuse/internal/config"
	"github.com/hashir500/Fuse/internal/cost"
)

type RequestInfo struct {
	Provider string
	Model    string
	Usage    cost.Usage
}

type ResponseInfo struct {
	Model string
	Usage cost.Usage
}

func Detect(r *http.Request, cfg *config.Config) (string, bool) {
	path := r.URL.Path
	if _, ok := cfg.Providers["anthropic"]; ok && strings.HasPrefix(path, "/v1/messages") {
		return "anthropic", true
	}
	if _, ok := cfg.Providers["openai"]; ok && (strings.HasPrefix(path, "/v1/chat/completions") || strings.HasPrefix(path, "/v1/responses")) {
		return "openai", true
	}
	if _, ok := cfg.Providers["google"]; ok && (strings.HasPrefix(path, "/v1beta/models/") || strings.HasPrefix(path, "/v1/models/")) {
		return "google", true
	}
	return "", false
}

func TargetURL(providerName string, cfg *config.Config, incoming *url.URL) (*url.URL, error) {
	providerCfg, ok := cfg.Providers[providerName]
	if !ok {
		return nil, fmt.Errorf("unknown provider %q", providerName)
	}
	base, err := url.Parse(providerCfg.BaseURL)
	if err != nil {
		return nil, err
	}
	target := *base
	target.Path = singleJoiningSlash(base.Path, incoming.Path)
	target.RawQuery = incoming.RawQuery
	return &target, nil
}

func PrepareRequest(r *http.Request, providerName string) (RequestInfo, []byte, error) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		return RequestInfo{}, nil, err
	}
	_ = r.Body.Close()
	r.Body = io.NopCloser(bytes.NewReader(body))

	info := RequestInfo{Provider: providerName}
	switch providerName {
	case "anthropic":
		info = parseAnthropicRequest(body)
	case "openai":
		info = parseOpenAIRequest(body)
	case "google":
		info = parseGoogleRequest(r.URL.Path, body)
	default:
		return RequestInfo{}, body, fmt.Errorf("unsupported provider %q", providerName)
	}
	info.Provider = providerName

	if info.Model == "" {
		info.Model = "unknown"
	}
	if info.Usage.TotalTokens == 0 {
		info.Usage.TotalTokens = info.Usage.PromptTokens + info.Usage.CompletionTokens
	}
	return info, body, nil
}

func ParseResponse(providerName string, body []byte) ResponseInfo {
	switch providerName {
	case "anthropic":
		return parseAnthropicResponse(body)
	case "openai":
		return parseOpenAIResponse(body)
	case "google":
		return parseGoogleResponse(body)
	default:
		return ResponseInfo{}
	}
}

func AddAuth(r *http.Request, providerName string, cfg *config.Config) {
	key := cfg.APIKey(providerName)
	if key == "" {
		return
	}
	switch providerName {
	case "anthropic":
		r.Header.Set("x-api-key", key)
	case "openai":
		r.Header.Set("Authorization", "Bearer "+key)
	case "google":
		q := r.URL.Query()
		if q.Get("key") == "" {
			q.Set("key", key)
			r.URL.RawQuery = q.Encode()
		}
	}
}

func parseAnthropicRequest(body []byte) RequestInfo {
	var req struct {
		Model     string `json:"model"`
		MaxTokens int    `json:"max_tokens"`
		Messages  any    `json:"messages"`
		System    any    `json:"system"`
	}
	_ = json.Unmarshal(body, &req)
	return RequestInfo{
		Model: req.Model,
		Usage: cost.Usage{
			PromptTokens:     approximateTokens(req.Messages) + approximateTokens(req.System),
			CompletionTokens: req.MaxTokens,
		},
	}
}

func parseOpenAIRequest(body []byte) RequestInfo {
	var req struct {
		Model            string `json:"model"`
		MaxTokens        int    `json:"max_tokens"`
		MaxCompletionTok int    `json:"max_completion_tokens"`
		Input            any    `json:"input"`
		Messages         any    `json:"messages"`
	}
	_ = json.Unmarshal(body, &req)
	output := req.MaxCompletionTok
	if output == 0 {
		output = req.MaxTokens
	}
	return RequestInfo{
		Model: req.Model,
		Usage: cost.Usage{
			PromptTokens:     approximateTokens(req.Messages) + approximateTokens(req.Input),
			CompletionTokens: output,
		},
	}
}

func parseGoogleRequest(path string, body []byte) RequestInfo {
	var req struct {
		Contents         any `json:"contents"`
		GenerationConfig struct {
			MaxOutputTokens int `json:"maxOutputTokens"`
		} `json:"generationConfig"`
	}
	_ = json.Unmarshal(body, &req)
	return RequestInfo{
		Model: googleModelFromPath(path),
		Usage: cost.Usage{
			PromptTokens:     approximateTokens(req.Contents),
			CompletionTokens: req.GenerationConfig.MaxOutputTokens,
		},
	}
}

func parseAnthropicResponse(body []byte) ResponseInfo {
	var resp struct {
		Model string `json:"model"`
		Usage struct {
			InputTokens  int `json:"input_tokens"`
			OutputTokens int `json:"output_tokens"`
		} `json:"usage"`
	}
	_ = json.Unmarshal(body, &resp)
	return ResponseInfo{
		Model: resp.Model,
		Usage: cost.Usage{
			PromptTokens:     resp.Usage.InputTokens,
			CompletionTokens: resp.Usage.OutputTokens,
			TotalTokens:      resp.Usage.InputTokens + resp.Usage.OutputTokens,
		},
	}
}

func parseOpenAIResponse(body []byte) ResponseInfo {
	var resp struct {
		Model string `json:"model"`
		Usage struct {
			PromptTokens     int `json:"prompt_tokens"`
			CompletionTokens int `json:"completion_tokens"`
			TotalTokens      int `json:"total_tokens"`
			InputTokens      int `json:"input_tokens"`
			OutputTokens     int `json:"output_tokens"`
		} `json:"usage"`
	}
	_ = json.Unmarshal(body, &resp)
	prompt := resp.Usage.PromptTokens
	if prompt == 0 {
		prompt = resp.Usage.InputTokens
	}
	completion := resp.Usage.CompletionTokens
	if completion == 0 {
		completion = resp.Usage.OutputTokens
	}
	total := resp.Usage.TotalTokens
	if total == 0 {
		total = prompt + completion
	}
	return ResponseInfo{
		Model: resp.Model,
		Usage: cost.Usage{
			PromptTokens:     prompt,
			CompletionTokens: completion,
			TotalTokens:      total,
		},
	}
}

func parseGoogleResponse(body []byte) ResponseInfo {
	var resp struct {
		Model string `json:"modelVersion"`
		Usage struct {
			PromptTokenCount     int `json:"promptTokenCount"`
			CandidatesTokenCount int `json:"candidatesTokenCount"`
			TotalTokenCount      int `json:"totalTokenCount"`
		} `json:"usageMetadata"`
	}
	_ = json.Unmarshal(body, &resp)
	return ResponseInfo{
		Model: resp.Model,
		Usage: cost.Usage{
			PromptTokens:     resp.Usage.PromptTokenCount,
			CompletionTokens: resp.Usage.CandidatesTokenCount,
			TotalTokens:      resp.Usage.TotalTokenCount,
		},
	}
}

func approximateTokens(value any) int {
	if value == nil {
		return 0
	}
	data, err := json.Marshal(value)
	if err != nil {
		return 0
	}
	tokens := len(data) / 4
	if tokens == 0 && len(data) > 0 {
		return 1
	}
	return tokens
}

var googleModelPattern = regexp.MustCompile(`/models/([^/:]+)`)

func googleModelFromPath(path string) string {
	matches := googleModelPattern.FindStringSubmatch(path)
	if len(matches) == 2 {
		return matches[1]
	}
	return ""
}

func singleJoiningSlash(a, b string) string {
	aslash := strings.HasSuffix(a, "/")
	bslash := strings.HasPrefix(b, "/")
	switch {
	case aslash && bslash:
		return a + b[1:]
	case !aslash && !bslash:
		return a + "/" + b
	default:
		return a + b
	}
}
