package cmd

import (
	"testing"

	"github.com/tvaroska/googrec/internal/api"
)

func TestSafeFilename(t *testing.T) {
	tests := []struct {
		title  string
		maxLen int
		want   string
	}{
		{"Simple Title", 80, "Simple Title"},
		{"Hello/World:Test", 80, "HelloWorldTest"},
		{"Meeting (2025)", 80, "Meeting 2025"},
		{"a/b\\c<d>e|f?g*h", 80, "abcdefgh"},
		{"", 80, ""},
		{"A very long title that exceeds the limit", 10, "A very lon"},
		{"  spaces  ", 80, "spaces"},
		{"café résumé", 80, "caf rsum"},
		{"file.with.dots", 80, "filewithdots"},
	}
	for _, tt := range tests {
		got := safeFilename(tt.title, tt.maxLen)
		if got != tt.want {
			t.Errorf("safeFilename(%q, %d) = %q, want %q", tt.title, tt.maxLen, got, tt.want)
		}
	}
}

func TestFilterByDateRange(t *testing.T) {
	recordings := []api.Recording{
		{ID: "1", Date: "2025-01-15T10:00:00Z"},
		{ID: "2", Date: "2025-03-01T12:00:00Z"},
		{ID: "3", Date: "2025-06-20T08:00:00Z"},
		{ID: "4", Date: "2025-09-10T14:00:00Z"},
		{ID: "5", Date: "2025-12-25T00:00:00Z"},
	}

	tests := []struct {
		name    string
		since   string
		until   string
		wantIDs []string
		wantErr bool
	}{
		{
			name:    "no filter",
			wantIDs: []string{"1", "2", "3", "4", "5"},
		},
		{
			name:    "since only",
			since:   "2025-06-01",
			wantIDs: []string{"3", "4", "5"},
		},
		{
			name:    "until only",
			until:   "2025-06-01",
			wantIDs: []string{"1", "2"},
		},
		{
			name:    "both since and until",
			since:   "2025-03-01",
			until:   "2025-10-01",
			wantIDs: []string{"2", "3", "4"},
		},
		{
			name:    "since RFC3339",
			since:   "2025-06-20T08:00:00Z",
			wantIDs: []string{"3", "4", "5"},
		},
		{
			name:    "until RFC3339",
			until:   "2025-03-01T12:00:00Z",
			wantIDs: []string{"1"},
		},
		{
			name:    "no matches",
			since:   "2026-01-01",
			wantIDs: nil,
		},
		{
			name:    "invalid since",
			since:   "not-a-date",
			wantErr: true,
		},
		{
			name:    "invalid until",
			until:   "not-a-date",
			wantErr: true,
		},
		{
			name:    "until before since",
			since:   "2025-06-01",
			until:   "2025-01-01",
			wantErr: true,
		},
		{
			name:    "until equals since",
			since:   "2025-06-01",
			until:   "2025-06-01",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := filterByDateRange(recordings, tt.since, tt.until)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			var gotIDs []string
			for _, r := range got {
				gotIDs = append(gotIDs, r.ID)
			}

			if len(gotIDs) != len(tt.wantIDs) {
				t.Fatalf("got %d recordings %v, want %d %v", len(gotIDs), gotIDs, len(tt.wantIDs), tt.wantIDs)
			}
			for i := range gotIDs {
				if gotIDs[i] != tt.wantIDs[i] {
					t.Errorf("recording[%d] = %q, want %q", i, gotIDs[i], tt.wantIDs[i])
				}
			}
		})
	}
}

func TestParseDate(t *testing.T) {
	tests := []struct {
		input   string
		wantErr bool
	}{
		{"2025-01-15", false},
		{"2025-06-20T08:00:00Z", false},
		{"2025-06-20T08:00:00+02:00", false},
		{"not-a-date", true},
		{"01/15/2025", true},
		{"2025", true},
	}
	for _, tt := range tests {
		_, err := parseDate(tt.input)
		if (err != nil) != tt.wantErr {
			t.Errorf("parseDate(%q): err=%v, wantErr=%v", tt.input, err, tt.wantErr)
		}
	}
}
