package cmd

import (
	"errors"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/tomekjarosik/geranos/pkg/transporter"
)

func createAdoptCommand() *cobra.Command {
	var adoptCommand = &cobra.Command{
		Use:   "adopt [dir name] [image name]",
		Short: "Adopt directory under provided reference, which can be later used for other commands",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			imagesDir := viper.GetString("images_directory")
			if len(imagesDir) == 0 {
				return errors.New("undefined image directory")
			}
			src := args[0]
			ref := args[1]
			opts := []transporter.Option{
				transporter.WithImagesPath(imagesDir),
			}
			return transporter.Adopt(src, ref, opts...)
		},
	}

	return adoptCommand
}
