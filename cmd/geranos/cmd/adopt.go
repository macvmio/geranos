package cmd

import (
	"github.com/macvmio/geranos/pkg/transporter"
	"github.com/spf13/cobra"
)

func NewCmdAdopt() *cobra.Command {
	var adoptCommand = &cobra.Command{
		Use:   "adopt [dir name] [image name]",
		Short: "Adopt a directory as an image under current local registry",
		Long: "Provided directory can be anywhere on your disk. It will be adopted as provided reference under current local registry." +
			"This will ensure that later it is available for use with other commands",
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			src := args[0]
			ref := TheAppConfig.Override(args[1])

			opts := []transporter.Option{
				transporter.WithImagesPath(TheAppConfig.ImagesDirectory),
			}
			return transporter.Adopt(src, ref, opts...)
		},
	}

	return adoptCommand
}
