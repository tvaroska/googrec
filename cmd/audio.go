package cmd

import (
	"fmt"
	"os"

	"github.com/boris/googrec/internal/api"
	"github.com/spf13/cobra"
)

func newAudioCmd() *cobra.Command {
	var output string

	cmd := &cobra.Command{
		Use:   "audio <id>",
		Short: "Download audio for a recording",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := api.NewClient()
			if err != nil {
				return err
			}

			result, err := client.GetAudio(args[0])
			if err != nil {
				return err
			}

			outPath := output
			if outPath == "" {
				outPath = result.Filename
			}

			if err := os.WriteFile(outPath, result.Data, 0644); err != nil {
				return err
			}

			sizeMB := float64(len(result.Data)) / (1024 * 1024)
			fmt.Printf("Audio saved to %s (%.1f MB, %s)\n", outPath, sizeMB, result.ContentType)
			return nil
		},
	}

	cmd.Flags().StringVarP(&output, "output", "o", "", "Output file path (default: server-provided filename)")
	return cmd
}
