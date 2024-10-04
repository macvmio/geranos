package cmd

import (
	"fmt"
	"github.com/macvmio/geranos/pkg/transporter"
	"github.com/spf13/cobra"
)

func NewCmdClone() *cobra.Command {
	var cloneCmd = &cobra.Command{
		Use:   "clone [src ref] [dst ref]",
		Short: "Locally clone one reference to other name",
		Long:  ``,
		Args:  cobra.ExactArgs(2),
		Run: func(cmd *cobra.Command, args []string) {
			src := TheAppConfig.Override(args[0])
			dst := TheAppConfig.Override(args[1])
			opts := []transporter.Option{
				transporter.WithImagesPath(TheAppConfig.ImagesDirectory),
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
