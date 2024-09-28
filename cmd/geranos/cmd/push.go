package cmd

import (
	"fmt"
	"github.com/google/go-containerregistry/pkg/name"
	"github.com/mobileinf/geranos/pkg/transporter"
	"github.com/spf13/cobra"
)

func NewCmdPush() *cobra.Command {
	var (
		flagMountedReference  string // Declares a variable to hold the value of the "--mountable-image" flag.
		flagConcurrentWorkers int
	)

	var pushCmd = &cobra.Command{
		Use:   "push [image name]",
		Short: "Push a large file as an OCI image to a registry.",
		Long:  `Uploads a specified file from the local system and packages it as an OCI image to be pushed to a specified container registry.`,
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			src := TheAppConfig.Override(args[0])
			opts := []transporter.Option{
				transporter.WithImagesPath(TheAppConfig.ImagesDirectory),
				transporter.WithContext(cmd.Context()),
				transporter.WithWorkersCount(flagConcurrentWorkers),
			}

			// Since mountedReference is directly bound to the flag,
			// we can just check if it's not empty and append the option.
			if flagMountedReference != "" {
				ref, err := name.ParseReference(flagMountedReference, name.StrictValidation)
				if err != nil {
					fmt.Println("invalid format of mounted reference")
					return
				}
				opts = append(opts, transporter.WithMountedReference(ref))
			}

			err := transporter.Push(src, opts...)
			if err != nil {
				fmt.Println(err)
			} else {
				fmt.Println("push has completed successfully")
			}
		},
	}

	// Binding the "--mountable-image" flag directly to the mountedReference variable.
	pushCmd.Flags().StringVar(&flagMountedReference, "mount", "",
		"Specifies an image reference that can be mounted to avoid uploading layers that exists in the registry")

	pushCmd.Flags().IntVar(&flagConcurrentWorkers, "concurrent-workers", 8,
		"Specifies number of concurrent workers to use when uploading layers to a registry")

	return pushCmd
}
