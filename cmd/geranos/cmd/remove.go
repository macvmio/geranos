package cmd

import (
	"fmt"
	"github.com/mobileinf/geranos/pkg/transporter"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func NewCmdRemove() *cobra.Command {
	var removeCommand = &cobra.Command{
		Use:   "remove [image ref]",
		Short: "Remove locally stored image",
		Long:  ``,
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			imagesDir := viper.GetString("images_directory")
			if len(imagesDir) == 0 {
				fmt.Println("undefined image directory")
				return
			}
			src := args[0]
			opts := []transporter.Option{
				transporter.WithImagesPath(imagesDir),
			}
			err := transporter.Remove(src, opts...)
			if err != nil {
				fmt.Printf("unable to remove: %v\n", err)
			} else {
				fmt.Printf("successfully removed %v\n", src)
			}
		},
	}

	return removeCommand
}
