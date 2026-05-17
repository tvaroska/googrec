package cmd

import (
	"github.com/spf13/cobra"
)

var (
	jsonOutput bool
)

func NewRootCmd() *cobra.Command {
	root := &cobra.Command{
		Use:   "googrec",
		Short: "CLI for downloading transcripts and audio from Google Recorder",
	}

	root.AddCommand(
		newAuthCmd(),
		newConfigCmd(),
		newListCmd(),
		newInfoCmd(),
		newTranscriptCmd(),
		newSearchCmd(),
		newDownloadCmd(),
		newAudioCmd(),
		newDownloadAudioCmd(),
	)

	return root
}
