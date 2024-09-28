package cmd

import (
	"fmt"
	"github.com/mobileinf/geranos/pkg/transporter"
	"github.com/spf13/cobra"
)

func NewCmdRemove() *cobra.Command {
	var removeCommand = &cobra.Command{
		Use:   "remove [image ref]",
		Short: "Remove locally stored image",
		Long:  ``,
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			src := args[0]
			opts := []transporter.Option{
				transporter.WithImagesPath(TheAppConfig.ImagesDirectory),
			}
			// TODO(tjarosik): Override command using TheAppConfig current contexts if present
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
