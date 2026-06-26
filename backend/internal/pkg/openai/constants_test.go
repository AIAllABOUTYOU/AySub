package openai

import "testing"

func TestDefaultModels_ContainsImageMiniModel(t *testing.T) {
	byID := make(map[string]Model, len(DefaultModels))
	for _, model := range DefaultModels {
		byID[model.ID] = model
	}

	model, ok := byID["gpt-image-1-mini"]
	if !ok {
		t.Fatalf("expected gpt-image-1-mini to be exposed in DefaultModels")
	}
	if model.DisplayName != "GPT Image 1 Mini" {
		t.Fatalf("unexpected display name: %q", model.DisplayName)
	}
}
