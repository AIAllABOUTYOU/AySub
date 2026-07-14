package apicompat

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAnthropicStreamingMaxTokens_MapsToIncomplete(t *testing.T) {
	state := NewAnthropicEventToResponsesState()

	AnthropicEventToResponsesEvents(&AnthropicStreamEvent{
		Type:    "message_start",
		Message: &AnthropicResponse{ID: "msg_test", Model: "claude-opus-4-6", Role: "assistant"},
	}, state)
	AnthropicEventToResponsesEvents(&AnthropicStreamEvent{
		Type:  "message_delta",
		Delta: &AnthropicDelta{StopReason: "max_tokens"},
		Usage: &AnthropicUsage{OutputTokens: 4096},
	}, state)

	require.Equal(t, "max_tokens", state.StopReason)
	events := AnthropicEventToResponsesEvents(&AnthropicStreamEvent{Type: "message_stop"}, state)

	var terminal *ResponsesStreamEvent
	for i := range events {
		if events[i].Type == "response.completed" || events[i].Type == "response.incomplete" {
			terminal = &events[i]
			break
		}
	}
	require.NotNil(t, terminal, "should have terminal event")
	assert.Equal(t, "response.incomplete", terminal.Type)
	require.NotNil(t, terminal.Response)
	assert.Equal(t, "incomplete", terminal.Response.Status)
	require.NotNil(t, terminal.Response.IncompleteDetails)
	assert.Equal(t, "max_output_tokens", terminal.Response.IncompleteDetails.Reason)
}

func TestAnthropicStreamingEndTurn_MapsToCompleted(t *testing.T) {
	state := NewAnthropicEventToResponsesState()

	AnthropicEventToResponsesEvents(&AnthropicStreamEvent{
		Type:    "message_start",
		Message: &AnthropicResponse{ID: "msg_test", Model: "claude-opus-4-6", Role: "assistant"},
	}, state)
	AnthropicEventToResponsesEvents(&AnthropicStreamEvent{
		Type:  "message_delta",
		Delta: &AnthropicDelta{StopReason: "end_turn"},
		Usage: &AnthropicUsage{OutputTokens: 100},
	}, state)

	events := AnthropicEventToResponsesEvents(&AnthropicStreamEvent{Type: "message_stop"}, state)

	var terminal *ResponsesStreamEvent
	for i := range events {
		if events[i].Type == "response.completed" {
			terminal = &events[i]
			break
		}
	}
	require.NotNil(t, terminal)
	assert.Equal(t, "completed", terminal.Response.Status)
	assert.Nil(t, terminal.Response.IncompleteDetails)
}

func TestResponsesToChatCompletions_ContentFilter(t *testing.T) {
	resp := &ResponsesResponse{
		ID:     "resp_cf",
		Status: "incomplete",
		IncompleteDetails: &ResponsesIncompleteDetails{
			Reason: "content_filter",
		},
		Output: []ResponsesOutput{{
			Type:    "message",
			Content: []ResponsesContentPart{{Type: "output_text", Text: "partial"}},
		}},
		Usage: &ResponsesUsage{InputTokens: 10, OutputTokens: 5},
	}

	cc := ResponsesToChatCompletions(resp, "gpt-5.5")
	require.Len(t, cc.Choices, 1)
	assert.Equal(t, "content_filter", cc.Choices[0].FinishReason)
}

func TestResponsesToChatCompletionsStreaming_ContentFilter(t *testing.T) {
	state := NewResponsesEventToChatState()
	state.ID = "resp_cf"
	state.Model = "gpt-5.5"
	state.SentRole = true

	events := ResponsesEventToChatChunks(&ResponsesStreamEvent{
		Type: "response.incomplete",
		Response: &ResponsesResponse{
			ID:     "resp_cf",
			Status: "incomplete",
			IncompleteDetails: &ResponsesIncompleteDetails{
				Reason: "content_filter",
			},
		},
	}, state)

	hasContentFilter := false
	for _, chunk := range events {
		for _, choice := range chunk.Choices {
			if choice.FinishReason != nil && *choice.FinishReason == "content_filter" {
				hasContentFilter = true
			}
		}
	}
	assert.True(t, hasContentFilter, "streaming content_filter should map to finish_reason content_filter")
}
