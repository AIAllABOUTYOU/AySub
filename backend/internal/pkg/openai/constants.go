// Package openai provides helpers and types for OpenAI API integration.
package openai

import _ "embed"

// Model represents an OpenAI model
type Model struct {
	ID          string `json:"id"`
	Object      string `json:"object"`
	Created     int64  `json:"created"`
	OwnedBy     string `json:"owned_by"`
	Type        string `json:"type"`
	DisplayName string `json:"display_name"`
}

// DefaultModels OpenAI models list
var DefaultModels = []Model{
	{ID: "gpt-5.6-sol", Object: "model", Created: 1780876800, OwnedBy: "openai", Type: "model", DisplayName: "GPT-5.6 Sol"},
	{ID: "gpt-5.6-terra", Object: "model", Created: 1780876800, OwnedBy: "openai", Type: "model", DisplayName: "GPT-5.6 Terra"},
	{ID: "gpt-5.6-luna", Object: "model", Created: 1780876800, OwnedBy: "openai", Type: "model", DisplayName: "GPT-5.6 Luna"},
	{ID: "gpt-5.5", Object: "model", Created: 1776873600, OwnedBy: "openai", Type: "model", DisplayName: "GPT-5.5"},
	{ID: "gpt-5.4", Object: "model", Created: 1738368000, OwnedBy: "openai", Type: "model", DisplayName: "GPT-5.4"},
	{ID: "gpt-5.4-mini", Object: "model", Created: 1738368000, OwnedBy: "openai", Type: "model", DisplayName: "GPT-5.4 Mini"},
	{ID: "gpt-5.3-codex", Object: "model", Created: 1735689600, OwnedBy: "openai", Type: "model", DisplayName: "GPT-5.3 Codex"},
	{ID: "gpt-5.3-codex-spark", Object: "model", Created: 1735689600, OwnedBy: "openai", Type: "model", DisplayName: "GPT-5.3 Codex Spark"},
	{ID: "gpt-5.2", Object: "model", Created: 1733875200, OwnedBy: "openai", Type: "model", DisplayName: "GPT-5.2"},
	{ID: "gpt-5-search-api", Object: "model", Created: 1760400000, OwnedBy: "openai", Type: "model", DisplayName: "GPT-5 Search API"},
	{ID: "gpt-image-1", Object: "model", Created: 1733875200, OwnedBy: "openai", Type: "model", DisplayName: "GPT Image 1"},
	{ID: "gpt-image-1-mini", Object: "model", Created: 1733875200, OwnedBy: "openai", Type: "model", DisplayName: "GPT Image 1 Mini"},
	{ID: "gpt-image-1.5", Object: "model", Created: 1735689600, OwnedBy: "openai", Type: "model", DisplayName: "GPT Image 1.5"},
	{ID: "gpt-image-2", Object: "model", Created: 1738368000, OwnedBy: "openai", Type: "model", DisplayName: "GPT Image 2"},
}

// XAIDefaultModels is the curated Grok/xAI compatible model list exposed for
// xAI API-key and Grok Cookie groups.
var XAIDefaultModels = []Model{
	{ID: "grok-4.5", Object: "model", Created: 1773014400, OwnedBy: "xai", Type: "model", DisplayName: "Grok 4.5"},
	{ID: "grok-4.20-0309-non-reasoning", Object: "model", Created: 1773014400, OwnedBy: "xai", Type: "model", DisplayName: "Grok 4.20 Fast"},
	{ID: "grok-4.20-0309", Object: "model", Created: 1773014400, OwnedBy: "xai", Type: "model", DisplayName: "Grok 4.20 Auto"},
	{ID: "grok-4.20-0309-reasoning", Object: "model", Created: 1773014400, OwnedBy: "xai", Type: "model", DisplayName: "Grok 4.20 Expert"},
	{ID: "grok-4.20-0309-non-reasoning-super", Object: "model", Created: 1773014400, OwnedBy: "xai", Type: "model", DisplayName: "Grok 4.20 Fast Super"},
	{ID: "grok-4.20-0309-super", Object: "model", Created: 1773014400, OwnedBy: "xai", Type: "model", DisplayName: "Grok 4.20 Auto Super"},
	{ID: "grok-4.20-0309-reasoning-super", Object: "model", Created: 1773014400, OwnedBy: "xai", Type: "model", DisplayName: "Grok 4.20 Expert Super"},
	{ID: "grok-4.20-0309-non-reasoning-heavy", Object: "model", Created: 1773014400, OwnedBy: "xai", Type: "model", DisplayName: "Grok 4.20 Fast Heavy"},
	{ID: "grok-4.20-0309-heavy", Object: "model", Created: 1773014400, OwnedBy: "xai", Type: "model", DisplayName: "Grok 4.20 Auto Heavy"},
	{ID: "grok-4.20-0309-reasoning-heavy", Object: "model", Created: 1773014400, OwnedBy: "xai", Type: "model", DisplayName: "Grok 4.20 Expert Heavy"},
	{ID: "grok-4.20-multi-agent-0309", Object: "model", Created: 1773014400, OwnedBy: "xai", Type: "model", DisplayName: "Grok 4.20 Multi Agent"},
	{ID: "grok-4.20-fast", Object: "model", Created: 1773014400, OwnedBy: "xai", Type: "model", DisplayName: "Grok 4.20 Fast"},
	{ID: "grok-4.20-auto", Object: "model", Created: 1773014400, OwnedBy: "xai", Type: "model", DisplayName: "Grok 4.20 Auto"},
	{ID: "grok-4.20-expert", Object: "model", Created: 1773014400, OwnedBy: "xai", Type: "model", DisplayName: "Grok 4.20 Expert"},
	{ID: "grok-4.20-heavy", Object: "model", Created: 1773014400, OwnedBy: "xai", Type: "model", DisplayName: "Grok 4.20 Heavy"},
	{ID: "grok-4.3-fast", Object: "model", Created: 1773014400, OwnedBy: "xai", Type: "model", DisplayName: "Grok 4.3 Fast"},
	{ID: "grok-4.3-beta", Object: "model", Created: 1773014400, OwnedBy: "xai", Type: "model", DisplayName: "Grok 4.3 Beta"},
	{ID: "grok-4.3-console", Object: "model", Created: 1773014400, OwnedBy: "xai", Type: "model", DisplayName: "Grok 4.3 Console"},
	{ID: "grok-4.3-low", Object: "model", Created: 1773014400, OwnedBy: "xai", Type: "model", DisplayName: "Grok 4.3 Low"},
	{ID: "grok-4.3-medium", Object: "model", Created: 1773014400, OwnedBy: "xai", Type: "model", DisplayName: "Grok 4.3 Medium"},
	{ID: "grok-4.3-high", Object: "model", Created: 1773014400, OwnedBy: "xai", Type: "model", DisplayName: "Grok 4.3 High"},
	{ID: "grok-4.20-0309-reasoning-console", Object: "model", Created: 1773014400, OwnedBy: "xai", Type: "model", DisplayName: "Grok 4.20 Reasoning Console"},
	{ID: "grok-4.20-0309-console", Object: "model", Created: 1773014400, OwnedBy: "xai", Type: "model", DisplayName: "Grok 4.20 Console"},
	{ID: "grok-4.20-0309-non-reasoning-console", Object: "model", Created: 1773014400, OwnedBy: "xai", Type: "model", DisplayName: "Grok 4.20 Fast Console"},
	{ID: "grok-4.20-multi-agent-console", Object: "model", Created: 1773014400, OwnedBy: "xai", Type: "model", DisplayName: "Grok 4.20 Multi Agent Console"},
	{ID: "grok-4.20-multi-agent-low", Object: "model", Created: 1773014400, OwnedBy: "xai", Type: "model", DisplayName: "Grok 4.20 Multi Agent Low"},
	{ID: "grok-4.20-multi-agent-medium", Object: "model", Created: 1773014400, OwnedBy: "xai", Type: "model", DisplayName: "Grok 4.20 Multi Agent Medium"},
	{ID: "grok-4.20-multi-agent-high", Object: "model", Created: 1773014400, OwnedBy: "xai", Type: "model", DisplayName: "Grok 4.20 Multi Agent High"},
	{ID: "grok-4.20-multi-agent-xhigh", Object: "model", Created: 1773014400, OwnedBy: "xai", Type: "model", DisplayName: "Grok 4.20 Multi Agent XHigh"},
	{ID: "grok-build-console", Object: "model", Created: 1773014400, OwnedBy: "xai", Type: "model", DisplayName: "Grok Build Console"},
	{ID: "grok-imagine-image-lite", Object: "model", Created: 1773014400, OwnedBy: "xai", Type: "model", DisplayName: "Grok Imagine Image Lite"},
	{ID: "grok-imagine-image", Object: "model", Created: 1773014400, OwnedBy: "xai", Type: "model", DisplayName: "Grok Imagine Image"},
	{ID: "grok-imagine-image-pro", Object: "model", Created: 1773014400, OwnedBy: "xai", Type: "model", DisplayName: "Grok Imagine Image Pro"},
	{ID: "grok-imagine-image-edit", Object: "model", Created: 1773014400, OwnedBy: "xai", Type: "model", DisplayName: "Grok Imagine Image Edit"},
	{ID: "grok-imagine-video", Object: "model", Created: 1773014400, OwnedBy: "xai", Type: "model", DisplayName: "Grok Imagine Video"},
}

// DefaultModelIDs returns the default model ID list
func DefaultModelIDs() []string {
	ids := make([]string, len(DefaultModels))
	for i, m := range DefaultModels {
		ids[i] = m.ID
	}
	return ids
}

// XAIDefaultModelIDs returns the default Grok/xAI model ID list.
func XAIDefaultModelIDs() []string {
	ids := make([]string, len(XAIDefaultModels))
	for i, m := range XAIDefaultModels {
		ids[i] = m.ID
	}
	return ids
}

// DefaultTestModel default model for testing OpenAI accounts
const DefaultTestModel = "gpt-5.4"

// DefaultInstructions default instructions for non-Codex CLI requests
// Content loaded from instructions.txt at compile time
//
//go:embed instructions.txt
var DefaultInstructions string
