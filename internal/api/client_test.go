package api

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestComputeSAPISIDHash(t *testing.T) {
	hash := ComputeSAPISIDHash("test-sapisid")
	if !strings.HasPrefix(hash, "SAPISIDHASH ") {
		t.Errorf("hash should start with 'SAPISIDHASH ', got %q", hash)
	}
	parts := strings.SplitN(hash, " ", 2)
	if len(parts) != 2 {
		t.Fatalf("expected 2 parts, got %d", len(parts))
	}
	tsParts := strings.SplitN(parts[1], "_", 2)
	if len(tsParts) != 2 {
		t.Fatalf("expected timestamp_hash, got %q", parts[1])
	}
	if len(tsParts[1]) != 40 {
		t.Errorf("SHA1 hex should be 40 chars, got %d", len(tsParts[1]))
	}
}

func TestValidateUUID(t *testing.T) {
	tests := []struct {
		id      string
		wantErr bool
	}{
		{"bf3451e0-4ea6-424e-8e77-fbef4c0fe17c", false},
		{"BF3451E0-4EA6-424E-8E77-FBEF4C0FE17C", false},
		{"not-a-uuid", true},
		{"", true},
		{"bf3451e04ea6424e8e77fbef4c0fe17c", true},
	}
	for _, tt := range tests {
		err := ValidateUUID(tt.id)
		if (err != nil) != tt.wantErr {
			t.Errorf("ValidateUUID(%q): err=%v, wantErr=%v", tt.id, err, tt.wantErr)
		}
	}
}

func TestFormatDuration(t *testing.T) {
	tests := []struct {
		ms   int64
		want string
	}{
		{0, "0:00"},
		{5230, "0:05"},
		{60000, "1:00"},
		{323000, "5:23"},
		{3723000, "1:02:03"},
	}
	for _, tt := range tests {
		got := FormatDuration(tt.ms)
		if got != tt.want {
			t.Errorf("FormatDuration(%d) = %q, want %q", tt.ms, got, tt.want)
		}
	}
}

func TestFormatTime(t *testing.T) {
	tests := []struct {
		ms   int64
		want string
	}{
		{0, "00:00"},
		{5000, "00:05"},
		{65000, "01:05"},
		{3600000, "60:00"},
	}
	for _, tt := range tests {
		got := FormatTime(tt.ms)
		if got != tt.want {
			t.Errorf("FormatTime(%d) = %q, want %q", tt.ms, got, tt.want)
		}
	}
}

func TestParseRecording(t *testing.T) {
	raw := `["device-id-123","Test Recording",["1706140050","207000000"],["1992","0"],0,0,"",null,null,null,null,"",null,"bf3451e0-4ea6-424e-8e77-fbef4c0fe17c"]`

	var items []json.RawMessage
	if err := json.Unmarshal([]byte(raw), &items); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	r, err := parseRecording(items)
	if err != nil {
		t.Fatalf("parseRecording: %v", err)
	}

	if r.ID != "bf3451e0-4ea6-424e-8e77-fbef4c0fe17c" {
		t.Errorf("ID = %q", r.ID)
	}
	if r.DeviceID != "device-id-123" {
		t.Errorf("DeviceID = %q", r.DeviceID)
	}
	if r.Title != "Test Recording" {
		t.Errorf("Title = %q", r.Title)
	}
	if r.Duration != "33:12" {
		t.Errorf("Duration = %q, want 33:12", r.Duration)
	}
}

func TestParseRecordingMissingWebID(t *testing.T) {
	raw := `["device-id","Title",["1706140050","0"],["60","0"],0,0,""]`

	var items []json.RawMessage
	if err := json.Unmarshal([]byte(raw), &items); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	r, err := parseRecording(items)
	if err != nil {
		t.Fatalf("parseRecording: %v", err)
	}

	if r.ID != "device-id" {
		t.Errorf("ID should fall back to deviceID, got %q", r.ID)
	}
}
