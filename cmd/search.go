package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/tvaroska/googrec/internal/api"
	"github.com/spf13/cobra"
)

func newSearchCmd() *cobra.Command {
	var (
		limit  int
		asJSON bool
	)

	cmd := &cobra.Command{
		Use:   "search <query>",
		Short: "Search recordings by title",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			query := strings.ToLower(args[0])

			client, err := api.NewClient()
			if err != nil {
				return err
			}

			recordings, err := client.ListRecordings(cmd.Context(), 0)
			if err != nil {
				return err
			}

			var matches []api.Recording
			for _, r := range recordings {
				if strings.Contains(strings.ToLower(r.Title), query) {
					matches = append(matches, r)
					if len(matches) >= limit {
						break
					}
				}
			}

			if asJSON {
				enc := json.NewEncoder(os.Stdout)
				enc.SetIndent("", "  ")
				return enc.Encode(matches)
			}

			if len(matches) == 0 {
				fmt.Printf("No recordings matching %q.\n", args[0])
				return nil
			}

			fmt.Printf("Found %d recording(s) matching %q:\n\n", len(matches), args[0])
			for _, r := range matches {
				t, _ := time.Parse(time.RFC3339, r.Date)
				fmt.Printf("  %s\n", r.ID)
				fmt.Printf("    %s\n", r.Title)
				fmt.Printf("    %s | %s\n\n", t.Local().Format("Jan 2 2006 3:04 PM"), r.Duration)
			}
			return nil
		},
	}

	cmd.Flags().IntVarP(&limit, "limit", "n", 20, "Maximum results")
	cmd.Flags().BoolVar(&asJSON, "json", false, "Output as JSON")

	return cmd
}
