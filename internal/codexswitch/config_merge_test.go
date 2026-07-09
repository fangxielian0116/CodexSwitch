package codexswitch

import (
	"strings"
	"testing"
)

func TestMergeManagedConfigRawAppliesAPIConfigAndPreservesLocalSections(t *testing.T) {
	currentRaw := `model = "gpt-5.4"
model_reasoning_effort = "medium"

[windows]
sandbox = "workspace-write"

[projects.'d:\codebuddy-project\codexswitch-master']
trust_level = "trusted"

[desktop]
localeOverride = "zh-CN"
`
	targetRaw := renderAPIConfig(APIProfileInput{
		BaseURL:              "https://api.example.com/v1",
		Model:                "gpt-5.4-mini",
		ModelReasoningEffort: "xhigh",
		ModelContextWindow:   "128000",
	})

	merged := mergeManagedConfigRaw(currentRaw, targetRaw)

	for _, want := range []string{
		`model_provider = "OpenAI"`,
		`model = "gpt-5.4-mini"`,
		`model_context_window = 128000`,
		`model_auto_compact_token_limit = 115200`,
		`base_url = "https://api.example.com/v1"`,
		`[projects.'d:\codebuddy-project\codexswitch-master']`,
		`trust_level = "trusted"`,
		`[desktop]`,
		`localeOverride = "zh-CN"`,
		`sandbox = "workspace-write"`,
	} {
		if !strings.Contains(merged, want) {
			t.Fatalf("expected merged config to contain %q, got %s", want, merged)
		}
	}
	if strings.Contains(merged, `sandbox = "elevated"`) {
		t.Fatalf("expected local windows sandbox to be preserved, got %s", merged)
	}
}

func TestMergeManagedConfigRawRestoresOfficialConfigWithoutDroppingLocalSections(t *testing.T) {
	currentRaw := renderAPIConfig(APIProfileInput{
		BaseURL:              "https://api.example.com/v1",
		Model:                "gpt-5.4-mini",
		ModelReasoningEffort: "xhigh",
		ModelContextWindow:   "128000",
	}) + `
[projects.'d:\codebuddy-project\codexswitch-master']
trust_level = "trusted"
`

	merged := mergeManagedConfigRaw(currentRaw, officialConfigTemplate)

	for _, want := range []string{
		`model = "gpt-5.4"`,
		`model_reasoning_effort = "xhigh"`,
		`[projects.'d:\codebuddy-project\codexswitch-master']`,
		`trust_level = "trusted"`,
	} {
		if !strings.Contains(merged, want) {
			t.Fatalf("expected merged config to contain %q, got %s", want, merged)
		}
	}
	for _, notWant := range []string{
		`model_provider = "OpenAI"`,
		`review_model =`,
		`model_context_window =`,
		`model_auto_compact_token_limit =`,
		`base_url =`,
		`wire_api =`,
		`requires_openai_auth =`,
	} {
		if strings.Contains(merged, notWant) {
			t.Fatalf("expected merged config not to contain %q, got %s", notWant, merged)
		}
	}
}
