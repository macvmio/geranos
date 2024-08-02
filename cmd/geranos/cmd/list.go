package cmd

import (
	"fmt"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/tomekjarosik/geranos/pkg/transporter"
)

func NewListCommand() *cobra.Command {
	var listCmd = &cobra.Command{
		Use:     "list",
		Short:   "List all OCI images in a specific repository.",
		Long:    `Lists all available OCI images in the specified container registry or repository, providing a quick overview of the stored images.`,
		Args:    cobra.ExactArgs(0),
		Aliases: []string{"ls"},
		Run: func(cmd *cobra.Command, args []string) {
			imagesDir := viper.GetString("images_directory")
			if len(imagesDir) == 0 {
				fmt.Printf("undefined image directory")
				return
			}
			opts := []transporter.Option{
				transporter.WithImagesPath(imagesDir),
			}
			err := transporter.List(opts...)
			if err != nil {
				fmt.Printf("%v", err)
			}
		},
	}

	return listCmd
}
