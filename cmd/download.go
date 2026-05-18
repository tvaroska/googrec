package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/tvaroska/googrec/internal/api"
	"github.com/spf13/cobra"
)

func newDownloadCmd() *cobra.Command {
	var (
		outDir       string
		limit        int
		all          bool
		since        string
		until        string
		format       string
		asJSON       bool
		skipExisting bool
		concurrency  int
	)

	cmd := &cobra.Command{
		Use:   "download",
		Short: "Bulk download transcripts for multiple recordings",
		RunE: func(cmd *cobra.Command, args []string) error {
			if all {
				limit = 0
			} else if limit <= 0 {
				return fmt.Errorf("--limit must be a positive integer")
			}

			client, err := api.NewClient()
			if err != nil {
				return err
			}

			actualFormat := format
			if asJSON {
				actualFormat = "json"
			}

			ext := "txt"
			if actualFormat == "json" {
				ext = "json"
			}

			return runBulkDownload(cmd.Context(), client, bulkConfig{
				outDir:       outDir,
				limit:        limit,
				since:        since,
				until:        until,
				skipExisting: skipExisting,
				concurrency:  concurrency,
			}, ext, func(ctx context.Context, client *api.Client, r api.Recording, filepath string) (string, error) {
				transcript, err := client.GetTranscript(ctx, r.ID)
				if err != nil {
					return "", err
				}
				if transcript == nil {
					return "skipped", nil
				}

				var content string
				if actualFormat == "json" {
					data, _ := json.MarshalIndent(map[string]any{
						"recording": map[string]any{
							"id":       r.ID,
							"title":    r.Title,
							"date":     r.Date,
							"duration": r.Duration,
						},
						"transcript": transcript,
					}, "", "  ")
					content = string(data)
				} else {
					content = fmt.Sprintf("Recording: %s\nDate: %s\nDuration: %s\nID: %s\n\n=== Transcript ===\n\n%s",
						r.Title, r.Date, r.Duration, r.ID, transcript.RawText)
				}

				return "done", os.WriteFile(filepath, []byte(content), 0644)
			})
		},
	}

	cmd.Flags().StringVarP(&outDir, "output", "o", ".", "Output directory")
	cmd.Flags().IntVarP(&limit, "limit", "n", 50, "Maximum recordings to process")
	cmd.Flags().BoolVar(&all, "all", false, "Download all recordings (paginate through all pages)")
	cmd.Flags().StringVar(&since, "since", "", "Only recordings after this date (YYYY-MM-DD or ISO 8601)")
	cmd.Flags().StringVar(&until, "until", "", "Only recordings before this date (YYYY-MM-DD or ISO 8601)")
	cmd.Flags().StringVar(&format, "format", "txt", "Output format: txt or json")
	cmd.Flags().BoolVar(&asJSON, "json", false, "Alias for --format json")
	cmd.Flags().BoolVar(&skipExisting, "skip-existing", false, "Skip recordings that already have a transcript file")
	cmd.Flags().IntVar(&concurrency, "concurrency", 3, "Number of concurrent downloads")

	return cmd
}
