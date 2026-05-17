package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/boris/googrec/internal/api"
	"github.com/spf13/cobra"
)

func newInfoCmd() *cobra.Command {
	var asJSON bool

	cmd := &cobra.Command{
		Use:   "info <id>",
		Short: "Show detailed information about a recording",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id := args[0]
			client, err := api.NewClient()
			if err != nil {
				return err
			}

			recordings, err := client.ListRecordings(100)
			if err != nil {
				return err
			}

			var found *api.Recording
			for i := range recordings {
				if recordings[i].ID == id || recordings[i].DeviceID == id {
					found = &recordings[i]
					break
				}
			}

			if found == nil {
				fmt.Fprintf(os.Stderr, "Recording not found: %s\nUse `googrec list` to see available recordings.\n", id)
				os.Exit(1)
			}

			if asJSON {
				enc := json.NewEncoder(os.Stdout)
				enc.SetIndent("", "  ")
				return enc.Encode(found)
			}

			t, _ := time.Parse(time.RFC3339, found.Date)
			fmt.Println("Recording Details")
			fmt.Println("=================")
			fmt.Printf("ID:       %s\n", found.ID)
			if found.DeviceID != "" && found.DeviceID != found.ID {
				fmt.Printf("Device:   %s\n", found.DeviceID)
			}
			fmt.Printf("Title:    %s\n", found.Title)
			fmt.Printf("Date:     %s\n", t.Local().Format("Jan 2 2006 3:04 PM"))
			fmt.Printf("Duration: %s\n", found.Duration)
			if found.Location != "" {
				fmt.Printf("Location: %s\n", found.Location)
			}
			fmt.Printf("Has transcript: %v\n", found.HasTranscript)

			return nil
		},
	}

	cmd.Flags().BoolVar(&asJSON, "json", false, "Output as JSON")
	return cmd
}
