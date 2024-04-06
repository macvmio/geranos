package cmd

import (
	"fmt"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/tomekjarosik/geranos/pkg/transporter"
)

func NewCmdInspect() *cobra.Command {
	var inspectCmd = &cobra.Command{
		Use:   "inspect [image name]",
		Short: "Inspect details of a specific OCI image.",
		Long:  `Provides detailed information about a specific OCI image, such as size, creation date, tags, and custom metadata.`,
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			imagesDir := viper.GetString("images_directory")
			if len(imagesDir) == 0 {
				fmt.Printf("undefined image directory")
				return
			}
			src := args[0]
			opts := []transporter.Option{
				transporter.WithImagesPath(imagesDir),
				transporter.WithContext(cmd.Context()),
			}
			out, err := transporter.Inspect(src, opts...)
			if err != nil {
				fmt.Printf("%v", err)
			} else {
				fmt.Println(out)
			}
		},
	}

	return inspectCmd
}
