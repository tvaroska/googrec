package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/tvaroska/googrec/internal/api"
	"github.com/spf13/cobra"
)

func newListCmd() *cobra.Command {
	var (
		limit    int
		asJSON   bool
	)

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List recent recordings",
		RunE: func(cmd *cobra.Command, args []string) error {
			if limit <= 0 {
				return fmt.Errorf("--limit must be a positive integer")
			}

			client, err := api.NewClient()
			if err != nil {
				return err
			}

			recordings, err := client.ListRecordings(cmd.Context(), limit)
			if err != nil {
				return err
			}

			if asJSON {
				enc := json.NewEncoder(os.Stdout)
				enc.SetIndent("", "  ")
				return enc.Encode(recordings)
			}

			if len(recordings) == 0 {
				fmt.Println("No recordings found.")
				return nil
			}

			fmt.Printf("Found %d recording(s):\n\n", len(recordings))
			for _, r := range recordings {
				t, _ := time.Parse(time.RFC3339, r.Date)
				loc := ""
				if r.Location != "" {
					loc = " | " + r.Location
				}
				fmt.Printf("  %s\n", r.ID)
				fmt.Printf("    %s\n", r.Title)
				fmt.Printf("    %s | %s%s\n\n", t.Local().Format("Jan 2 2006 3:04 PM"), r.Duration, loc)
			}
			return nil
		},
	}

	cmd.Flags().IntVarP(&limit, "limit", "n", 20, "Maximum number of recordings")
	cmd.Flags().BoolVar(&asJSON, "json", false, "Output as JSON")

	return cmd
}
