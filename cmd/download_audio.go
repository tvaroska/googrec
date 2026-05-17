package cmd

import (
	"fmt"
	"os"

	"github.com/boris/googrec/internal/api"
	"github.com/spf13/cobra"
)

func newDownloadAudioCmd() *cobra.Command {
	var (
		outDir       string
		limit        int
		since        string
		skipExisting bool
		concurrency  int
	)

	cmd := &cobra.Command{
		Use:   "download-audio",
		Short: "Bulk download audio files for multiple recordings",
		RunE: func(cmd *cobra.Command, args []string) error {
			if limit <= 0 {
				return fmt.Errorf("--limit must be a positive integer")
			}

			client, err := api.NewClient()
			if err != nil {
				return err
			}

			return runBulkDownload(cmd.Context(), client, bulkConfig{
				outDir:       outDir,
				limit:        limit,
				since:        since,
				skipExisting: skipExisting,
				concurrency:  concurrency,
			}, "m4a", func(client *api.Client, r api.Recording, filepath string) (string, error) {
				f, err := os.Create(filepath)
				if err != nil {
					return "", err
				}
				defer f.Close()

				result, err := client.GetAudio(cmd.Context(), r.ID, f)
				if err != nil {
					os.Remove(filepath)
					return "", err
				}
				sizeMB := float64(result.BytesWritten) / (1024 * 1024)
				return fmt.Sprintf("done (%.1f MB)", sizeMB), nil
			})
		},
	}

	cmd.Flags().StringVarP(&outDir, "output", "o", ".", "Output directory")
	cmd.Flags().IntVarP(&limit, "limit", "n", 50, "Maximum recordings to process")
	cmd.Flags().StringVar(&since, "since", "", "Only recordings after this date (YYYY-MM-DD or ISO 8601)")
	cmd.Flags().BoolVar(&skipExisting, "skip-existing", false, "Skip recordings that already have an audio file")
	cmd.Flags().IntVar(&concurrency, "concurrency", 3, "Number of concurrent downloads")

	return cmd
}
