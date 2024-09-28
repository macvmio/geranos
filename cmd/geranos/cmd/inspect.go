package cmd

import (
	"fmt"
	"github.com/mobileinf/geranos/pkg/transporter"
	"github.com/spf13/cobra"
)

func NewCmdInspect() *cobra.Command {
	var inspectCmd = &cobra.Command{
		Use:   "inspect [image name]",
		Short: "Inspect details of a specific OCI image.",
		Long:  `Provides detailed information about a specific OCI image, such as size, creation date, tags, and custom metadata.`,
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			src := TheAppConfig.Override(args[0])
			opts := []transporter.Option{
				transporter.WithImagesPath(TheAppConfig.ImagesDirectory),
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
