package cmd

import (
	"github.com/macvmio/geranos/pkg/transporter"
	"github.com/spf13/cobra"
)

func NewCmdPull() *cobra.Command {
	var pullCmd = &cobra.Command{
		Use:   "pull [image name]",
		Short: "Pull an OCI image from a registry and extract the file.",
		Long:  `Downloads an OCI image from a specified container registry and extracts the file to a specified local path.`,
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			src := TheAppConfig.Override(args[0])
			progress := make(chan transporter.ProgressUpdate)
			defer close(progress)

			opts := []transporter.Option{
				transporter.WithImagesPath(TheAppConfig.ImagesDirectory),
				transporter.WithContext(cmd.Context()),
				transporter.WithVerbose(TheAppConfig.Verbose),
				transporter.WithProgressChannel(progress),
			}
			go transporter.PrintProgress(progress)
			return transporter.Pull(src, opts...)
		},
	}

	return pullCmd
}
