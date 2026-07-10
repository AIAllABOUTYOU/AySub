package service

import "strings"

const (
	VideoBillingResolution480P  = "480p"
	VideoBillingResolution720P  = "720p"
	VideoBillingResolution1080P = "1080p"
)

// AySub's Grok adapter accepts 6/10/12/16/20 second jobs and defaults to 6 seconds.
const VideoBillingDefaultDurationSeconds = 6

func NormalizeVideoBillingDurationSecondsOrDefault(durationSeconds int) int {
	switch durationSeconds {
	case 6, 10, 12, 16, 20:
		return durationSeconds
	default:
		return VideoBillingDefaultDurationSeconds
	}
}

func NormalizeVideoBillingResolutionOrDefault(resolution string) string {
	switch strings.ToLower(strings.TrimSpace(resolution)) {
	case "480", "480p", "sd":
		return VideoBillingResolution480P
	case "720", "720p", "hd":
		return VideoBillingResolution720P
	case "1080", "1080p", "full_hd", "full-hd", "fhd":
		return VideoBillingResolution1080P
	default:
		return VideoBillingResolution480P
	}
}
