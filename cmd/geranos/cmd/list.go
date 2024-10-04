package cmd

import (
	"fmt"
	"github.com/macvmio/geranos/pkg/transporter"
	"github.com/spf13/cobra"
)

func NewCmdList() *cobra.Command {
	var listCmd = &cobra.Command{
		Use:     "list",
		Short:   "List all OCI images in a specific local registry",
		Long:    `Lists all available OCI images in the specified registry, providing a quick overview of the stored images.`,
		Args:    cobra.ExactArgs(0),
		Aliases: []string{"ls"},
		Run: func(cmd *cobra.Command, args []string) {
			opts := []transporter.Option{
				transporter.WithImagesPath(TheAppConfig.ImagesDirectory),
			}
			err := transporter.List(opts...)
			if err != nil {
				fmt.Printf("%v", err)
			}
		},
	}

	return listCmd
}
