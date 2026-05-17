package auth

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

type AuthData struct {
	SAPISID  string `json:"sapisid"`
	Cookies  string `json:"cookies"`
	AuthUser int    `json:"authUser"`
	APIKey   string `json:"apiKey"`
	SavedAt  string `json:"savedAt"`
}

var sapisidRe = regexp.MustCompile(`(?:^|;\s*)SAPISID=([^;]+)`)

func ConfigDir() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "googrec")
}

func AuthFilePath() string {
	return filepath.Join(ConfigDir(), "auth.json")
}

func Load() (*AuthData, error) {
	data, err := os.ReadFile(AuthFilePath())
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, err
	}
	var auth AuthData
	if err := json.Unmarshal(data, &auth); err != nil {
		return nil, fmt.Errorf("corrupt auth file: %w", err)
	}
	return &auth, nil
}

func Save(cookies string, authUser int, apiKey string) error {
	sapisid, err := ExtractSAPISID(cookies)
	if err != nil {
		return err
	}

	dir := ConfigDir()
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("create config dir: %w", err)
	}

	// Preserve existing API key if not provided
	if apiKey == "" {
		if existing, _ := Load(); existing != nil {
			apiKey = existing.APIKey
		}
	}

	auth := AuthData{
		SAPISID:  sapisid,
		Cookies:  cookies,
		AuthUser: authUser,
		APIKey:   apiKey,
		SavedAt:  time.Now().UTC().Format(time.RFC3339),
	}

	data, err := json.MarshalIndent(auth, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(AuthFilePath(), data, 0600)
}

func SaveAPIKey(apiKey string) error {
	auth, err := Load()
	if err != nil {
		return err
	}
	if auth == nil {
		return errors.New("no auth file found — run `googrec auth` first")
	}
	auth.APIKey = apiKey
	auth.SavedAt = time.Now().UTC().Format(time.RFC3339)

	data, err := json.MarshalIndent(auth, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(AuthFilePath(), data, 0600)
}

func ExtractSAPISID(cookies string) (string, error) {
	m := sapisidRe.FindStringSubmatch(cookies)
	if m == nil {
		return "", errors.New("SAPISID cookie not found in provided cookies")
	}
	return strings.TrimSpace(m[1]), nil
}
