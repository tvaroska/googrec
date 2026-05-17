package cmd

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/boris/googrec/internal/api"
	"github.com/spf13/cobra"
)

func newTranscriptCmd() *cobra.Command {
	var (
		output string
		asJSON bool
		plain  bool
	)

	cmd := &cobra.Command{
		Use:   "transcript <id>",
		Short: "Download transcript for a single recording",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := api.NewClient()
			if err != nil {
				return err
			}

			transcript, err := client.GetTranscript(cmd.Context(), args[0])
			if err != nil {
				return err
			}
			if transcript == nil {
				fmt.Fprintf(os.Stderr, "No transcript found for recording: %s\n", args[0])
				os.Exit(1)
			}

			var text string
			switch {
			case asJSON:
				data, _ := json.MarshalIndent(transcript, "", "  ")
				text = string(data)
			case plain:
				for i, s := range transcript.Segments {
					if i > 0 {
						text += "\n\n"
					}
					text += s.Text
				}
			default:
				text = transcript.RawText
			}

			if output != "" {
				if err := os.WriteFile(output, []byte(text), 0644); err != nil {
					return err
				}
				fmt.Fprintf(os.Stderr, "Transcript saved to %s\n", output)
			} else {
				fmt.Println(text)
			}

			return nil
		},
	}

	cmd.Flags().StringVarP(&output, "output", "o", "", "Output file path (default: stdout)")
	cmd.Flags().BoolVar(&asJSON, "json", false, "Output as JSON (with speaker segments)")
	cmd.Flags().BoolVar(&plain, "plain", false, "Output plain text without speaker labels")

	return cmd
}
