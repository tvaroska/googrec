package cmd

import (
	"fmt"
	"os"

	"github.com/tvaroska/googrec/internal/api"
	"github.com/spf13/cobra"
)

func newAudioCmd() *cobra.Command {
	var output string

	cmd := &cobra.Command{
		Use:   "audio <id>",
		Short: "Download audio for a recording",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			client, err := api.NewClient()
			if err != nil {
				return err
			}

			if output != "" {
				f, err := os.Create(output)
				if err != nil {
					return err
				}
				defer f.Close()

				result, err := client.GetAudio(ctx, args[0], f)
				if err != nil {
					os.Remove(output)
					return err
				}

				sizeMB := float64(result.BytesWritten) / (1024 * 1024)
				fmt.Printf("Audio saved to %s (%.1f MB, %s)\n", output, sizeMB, result.ContentType)
				return nil
			}

			// No output flag — need filename from server headers
			tmpFile, err := os.CreateTemp(".", ".googrec-audio-*")
			if err != nil {
				return err
			}
			tmpPath := tmpFile.Name()

			result, err := client.GetAudio(ctx, args[0], tmpFile)
			tmpFile.Close()
			if err != nil {
				os.Remove(tmpPath)
				return err
			}

			outPath := result.Filename
			if err := os.Rename(tmpPath, outPath); err != nil {
				os.Remove(tmpPath)
				return err
			}

			sizeMB := float64(result.BytesWritten) / (1024 * 1024)
			fmt.Printf("Audio saved to %s (%.1f MB, %s)\n", outPath, sizeMB, result.ContentType)
			return nil
		},
	}

	cmd.Flags().StringVarP(&output, "output", "o", "", "Output file path (default: server-provided filename)")
	return cmd
}
