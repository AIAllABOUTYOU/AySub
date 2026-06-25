//go:build unit

package service

import "testing"

func TestValidateJitter(t *testing.T) {
	tests := []struct {
		name     string
		jitter   int
		interval int
		wantErr  error
	}{
		{name: "zero jitter", jitter: 0, interval: 60},
		{name: "keeps minimum floor", jitter: 45, interval: 60},
		{name: "negative jitter", jitter: -1, interval: 60, wantErr: ErrChannelMonitorInvalidJitter},
		{name: "below minimum floor", jitter: 46, interval: 60, wantErr: ErrChannelMonitorInvalidJitter},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateJitter(tt.jitter, tt.interval)
			if err != tt.wantErr {
				t.Fatalf("validateJitter(%d, %d) = %v, want %v", tt.jitter, tt.interval, err, tt.wantErr)
			}
		})
	}
}
