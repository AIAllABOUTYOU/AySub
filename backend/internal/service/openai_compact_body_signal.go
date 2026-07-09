package service

import "github.com/tidwall/gjson"

// HasCompactionTriggerInInput detects Codex remote compact v2 body-signal
// requests sent to POST /v1/responses with an input item of type
// "compaction_trigger".
func HasCompactionTriggerInInput(body []byte) bool {
	if len(body) == 0 {
		return false
	}
	input := gjson.GetBytes(body, "input")
	if !input.IsArray() {
		return false
	}
	found := false
	input.ForEach(func(_, item gjson.Result) bool {
		if item.Get("type").String() == "compaction_trigger" {
			found = true
			return false
		}
		return true
	})
	return found
}
