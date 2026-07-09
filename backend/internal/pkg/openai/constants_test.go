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

func TestDefaultModels_ContainsSearchAPIModel(t *testing.T) {
	byID := make(map[string]Model, len(DefaultModels))
	for _, model := range DefaultModels {
		byID[model.ID] = model
	}

	model, ok := byID["gpt-5-search-api"]
	if !ok {
		t.Fatalf("expected gpt-5-search-api to be exposed in DefaultModels")
	}
	if model.DisplayName != "GPT-5 Search API" {
		t.Fatalf("unexpected display name: %q", model.DisplayName)
	}
}

func TestDefaultModels_ContainsGPT56Models(t *testing.T) {
	byID := make(map[string]Model, len(DefaultModels))
	for _, model := range DefaultModels {
		byID[model.ID] = model
	}

	for id, displayName := range map[string]string{
		"gpt-5.6-sol":   "GPT-5.6 Sol",
		"gpt-5.6-terra": "GPT-5.6 Terra",
		"gpt-5.6-luna":  "GPT-5.6 Luna",
	} {
		model, ok := byID[id]
		if !ok {
			t.Fatalf("expected %s to be exposed in DefaultModels", id)
		}
		if model.DisplayName != displayName {
			t.Fatalf("unexpected display name for %s: %q", id, model.DisplayName)
		}
	}
}

func TestXAIDefaultModels_ContainsGrok45Model(t *testing.T) {
	byID := make(map[string]Model, len(XAIDefaultModels))
	for _, model := range XAIDefaultModels {
		byID[model.ID] = model
	}

	model, ok := byID["grok-4.5"]
	if !ok {
		t.Fatalf("expected grok-4.5 to be exposed in XAIDefaultModels")
	}
	if model.DisplayName != "Grok 4.5" {
		t.Fatalf("unexpected display name: %q", model.DisplayName)
	}
}
