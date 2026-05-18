package cmd

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/tvaroska/googrec/internal/api"
	"github.com/tvaroska/googrec/internal/auth"
	"github.com/spf13/cobra"
)

func newAuthCmd() *cobra.Command {
	var (
		check    bool
		apiKey   string
		authuser int
	)

	cmd := &cobra.Command{
		Use:   "auth",
		Short: "Set up or test authentication with Google Recorder",
		RunE: func(cmd *cobra.Command, args []string) error {
			if apiKey != "" {
				if err := auth.SaveAPIKey(apiKey); err != nil {
					return err
				}
				fmt.Println("API key saved.")
				return nil
			}

			if check {
				return runAuthCheck(cmd.Context())
			}

			return runAuthSetup(cmd.Context(), authuser)
		},
	}

	cmd.Flags().BoolVar(&check, "check", false, "Test existing authentication")
	cmd.Flags().StringVar(&apiKey, "api-key", "", "Set the API key (from X-Goog-Api-Key header in DevTools)")
	cmd.Flags().IntVar(&authuser, "authuser", 0, "Google account index for multi-account users")

	return cmd
}

func runAuthCheck(ctx context.Context) error {
	a, err := auth.Load()
	if err != nil {
		return err
	}
	if a == nil {
		fmt.Printf("No authentication found.\nAuth file: %s\nRun `googrec auth` to authenticate.\n", auth.AuthFilePath())
		os.Exit(1)
	}

	fmt.Printf("Auth file:  %s\n", auth.AuthFilePath())
	fmt.Printf("Auth user:  %d\n", a.AuthUser)
	fmt.Printf("Saved at:   %s\n", a.SavedAt)
	fmt.Printf("Has API key: %v\n", a.APIKey != "")

	fmt.Println("\nTesting authentication...")
	client, err := api.NewClient()
	if err != nil {
		fmt.Printf("Authentication invalid: %v\n", err)
		os.Exit(1)
	}
	if err := client.TestAuth(ctx); err != nil {
		fmt.Printf("Authentication failed: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("Authentication is valid.")
	return nil
}

func runAuthSetup(ctx context.Context, authUser int) error {
	fmt.Println("Google Recorder Authentication")
	fmt.Println("==============================")
	fmt.Println()
	fmt.Println("To authenticate, copy your cookies from Chrome DevTools:")
	fmt.Println()
	fmt.Println("1. Open https://recorder.google.com in Chrome")
	fmt.Println("2. Open DevTools (Cmd+Option+I on Mac, F12 on Windows/Linux)")
	fmt.Println("3. Go to the Network tab and refresh the page")
	fmt.Println("4. Click any request to pixelrecorder-pa.clients6.google.com")
	fmt.Println("5. In Headers → Request Headers, find \"Cookie\"")
	fmt.Println("6. Right-click the value and select \"Copy value\"")
	fmt.Println("7. Paste below and press Enter")
	fmt.Println()

	reader := bufio.NewReader(os.Stdin)

	fmt.Print("Cookies: ")
	cookies, _ := reader.ReadString('\n')
	cookies = strings.TrimSpace(cookies)
	if cookies == "" {
		fmt.Println("No cookies provided.")
		os.Exit(1)
	}

	if _, err := auth.ExtractSAPISID(cookies); err != nil {
		return fmt.Errorf("invalid cookies: %w", err)
	}

	fmt.Println()
	fmt.Println("Now paste the API key (X-Goog-Api-Key header from any request to pixelrecorder-pa).")
	fmt.Println("You can skip this and set it later with: googrec auth --api-key <key>")
	fmt.Print("API Key (or press Enter to skip): ")
	apiKeyInput, _ := reader.ReadString('\n')
	apiKeyInput = strings.TrimSpace(apiKeyInput)

	if err := auth.Save(cookies, authUser, apiKeyInput); err != nil {
		return err
	}

	fmt.Printf("\nCredentials saved to %s\n", auth.AuthFilePath())

	if apiKeyInput != "" {
		fmt.Println("\nTesting authentication...")
		client, err := api.NewClient()
		if err != nil {
			fmt.Printf("Warning: %v\n", err)
			return nil
		}
		if err := client.TestAuth(ctx); err != nil {
			fmt.Printf("Authentication test failed: %v\n", err)
			fmt.Println("Cookies saved but may be invalid. Try re-authenticating.")
		} else {
			fmt.Println("Authentication successful!")
		}
	} else {
		fmt.Println("\nNote: Set the API key to complete setup:")
		fmt.Println("  googrec auth --api-key <key>")
	}

	return nil
}
