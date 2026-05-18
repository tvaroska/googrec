package cmd

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/tvaroska/googrec/internal/auth"
	"github.com/spf13/cobra"
)

func envOrNone(key string) string {
	if v := os.Getenv(key); v != "" {
		return "set"
	}
	return "not set"
}

func newConfigCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "config",
		Short: "Show current configuration",
		RunE: func(cmd *cobra.Command, args []string) error {
			path := auth.AuthFilePath()
			fmt.Println("Configuration")
			fmt.Println("=============")
			fmt.Printf("Auth file: %s\n", path)

			if _, err := os.Stat(path); os.IsNotExist(err) {
				fmt.Println("Auth file exists: false")
				return nil
			}

			fmt.Println("Auth file exists: true")

			data, err := os.ReadFile(path)
			if err != nil {
				fmt.Println("(Could not read auth file)")
				return nil
			}

			var raw map[string]json.RawMessage
			if err := json.Unmarshal(data, &raw); err != nil {
				fmt.Println("(Could not parse auth file)")
				return nil
			}

			a, _ := auth.Load()
			if a != nil {
				fmt.Printf("Auth user:    %d\n", a.AuthUser)
				fmt.Printf("Saved at:     %s\n", a.SavedAt)
				fmt.Printf("Has cookies:  %v\n", a.Cookies != "")
				fmt.Printf("Has SAPISID:  %v\n", a.SAPISID != "")
				fmt.Printf("Has API key:  %v\n", a.APIKey != "")
			}

			fmt.Println()
			fmt.Println("Environment overrides:")
			fmt.Printf("  GOOGREC_COOKIES:  %s\n", envOrNone("GOOGREC_COOKIES"))
			fmt.Printf("  GOOGREC_API_KEY:  %s\n", envOrNone("GOOGREC_API_KEY"))
			fmt.Printf("  GOOGREC_AUTHUSER: %s\n", envOrNone("GOOGREC_AUTHUSER"))

			return nil
		},
	}
}
