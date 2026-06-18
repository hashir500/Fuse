package provider

import "testing"

func TestParseGoogleResponseIncludesThinkingTokens(t *testing.T) {
	body := []byte(`{
		"usageMetadata": {
			"promptTokenCount": 7,
			"candidatesTokenCount": 2,
			"thoughtsTokenCount": 24,
			"totalTokenCount": 33
		},
		"modelVersion": "gemini-2.5-flash"
	}`)

	got := ParseResponse("google", body)
	if got.Model != "gemini-2.5-flash" {
		t.Fatalf("model = %q, want gemini-2.5-flash", got.Model)
	}
	if got.Usage.PromptTokens != 7 {
		t.Fatalf("prompt tokens = %d, want 7", got.Usage.PromptTokens)
	}
	if got.Usage.CompletionTokens != 26 {
		t.Fatalf("completion tokens = %d, want 26", got.Usage.CompletionTokens)
	}
	if got.Usage.TotalTokens != 33 {
		t.Fatalf("total tokens = %d, want 33", got.Usage.TotalTokens)
	}
}
