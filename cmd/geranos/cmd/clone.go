package cmd

import (
	"fmt"
	"github.com/mobileinf/geranos/pkg/transporter"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func NewCmdClone() *cobra.Command {
	var cloneCmd = &cobra.Command{
		Use:   "clone [src ref] [dst ref]",
		Short: "Clone one reference to other name",
		Long:  ``,
		Args:  cobra.ExactArgs(2),
		Run: func(cmd *cobra.Command, args []string) {
			imagesDir := viper.GetString("images_directory")
			if len(imagesDir) == 0 {
				fmt.Println("undefined image directory")
				return
			}
			src := args[0]
			dst := args[1]
			opts := []transporter.Option{
				transporter.WithImagesPath(imagesDir),
			}
			err := transporter.Clone(src, dst, opts...)
			if err != nil {
				fmt.Printf("error while cloning: %v", err)
			} else {
				fmt.Println("cloned successfully")
			}
		},
	}

	return cloneCmd
}
