package auth

import (
	"os"
	"path/filepath"
	"testing"
)

func TestExtractSAPISID(t *testing.T) {
	tests := []struct {
		name    string
		cookies string
		want    string
		wantErr bool
	}{
		{
			name:    "simple",
			cookies: "SAPISID=abc123def",
			want:    "abc123def",
		},
		{
			name:    "among other cookies",
			cookies: "SID=x; HSID=y; SAPISID=abc123def; APISID=z",
			want:    "abc123def",
		},
		{
			name:    "missing",
			cookies: "SID=x; HSID=y",
			wantErr: true,
		},
		{
			name:    "empty",
			cookies: "",
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ExtractSAPISID(tt.cookies)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestSaveAndLoad(t *testing.T) {
	tmp := t.TempDir()
	origHome := os.Getenv("HOME")
	t.Setenv("HOME", tmp)
	defer os.Setenv("HOME", origHome)

	// Ensure config dir uses temp home
	expectedDir := filepath.Join(tmp, ".config", "googrec")

	err := Save("SID=x; SAPISID=testvalue; HSID=y", 0, "test-api-key")
	if err != nil {
		t.Fatalf("Save: %v", err)
	}

	// Check file permissions
	info, err := os.Stat(filepath.Join(expectedDir, "auth.json"))
	if err != nil {
		t.Fatalf("stat: %v", err)
	}
	if perm := info.Mode().Perm(); perm != 0600 {
		t.Errorf("file perm = %o, want 0600", perm)
	}

	auth, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if auth == nil {
		t.Fatal("Load returned nil")
	}
	if auth.SAPISID != "testvalue" {
		t.Errorf("SAPISID = %q, want %q", auth.SAPISID, "testvalue")
	}
	if auth.APIKey != "test-api-key" {
		t.Errorf("APIKey = %q, want %q", auth.APIKey, "test-api-key")
	}
	if auth.AuthUser != 0 {
		t.Errorf("AuthUser = %d, want 0", auth.AuthUser)
	}
}

func TestLoadMissing(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)

	auth, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if auth != nil {
		t.Errorf("expected nil for missing file, got %+v", auth)
	}
}

func TestSavePreservesAPIKey(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)

	// Save with API key
	err := Save("SAPISID=val1", 0, "my-key")
	if err != nil {
		t.Fatalf("first Save: %v", err)
	}

	// Save again without API key — should preserve
	err = Save("SAPISID=val2", 0, "")
	if err != nil {
		t.Fatalf("second Save: %v", err)
	}

	auth, _ := Load()
	if auth.APIKey != "my-key" {
		t.Errorf("APIKey = %q, want %q", auth.APIKey, "my-key")
	}
	if auth.SAPISID != "val2" {
		t.Errorf("SAPISID = %q, want %q", auth.SAPISID, "val2")
	}
}
