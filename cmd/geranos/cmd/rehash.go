package cmd

import (
	"github.com/macvmio/geranos/pkg/transporter"
	"github.com/spf13/cobra"
)

func NewCmdRehash() *cobra.Command {
	var rehashCmd = &cobra.Command{
		Use:   "rehash [image name]",
		Short: "Recalculates the manifest for a given local OCI image.",
		Long:  `Recalculates a new manifest for the specified OCI image by rehashing its configuration and file contents.`,
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			src := TheAppConfig.Override(args[0])
			return transporter.Rehash(src,
				transporter.WithContext(cmd.Context()),
				transporter.WithImagesPath(TheAppConfig.ImagesDirectory))
		},
	}

	return rehashCmd
}
