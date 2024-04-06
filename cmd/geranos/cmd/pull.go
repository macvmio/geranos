package cmd

import (
	"errors"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/tomekjarosik/geranos/pkg/transporter"
)

func NewCmdPull() *cobra.Command {
	var pullCmd = &cobra.Command{
		Use:   "pull [image name]",
		Short: "Pull an OCI image from a registry and extract the file.",
		Long:  `Downloads an OCI image from a specified container registry and extracts the file to a specified local path.`,
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			imagesDir := viper.GetString("images_directory")
			if len(imagesDir) == 0 {
				return errors.New("undefined image directory")
			}
			src := args[0]
			opts := []transporter.Option{
				transporter.WithImagesPath(imagesDir),
				transporter.WithContext(cmd.Context()),
			}
			return transporter.Pull(src, opts...)
		},
	}

	return pullCmd
}
