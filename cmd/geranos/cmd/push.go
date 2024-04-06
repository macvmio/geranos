package cmd

import (
	"fmt"
	"github.com/google/go-containerregistry/pkg/name"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/tomekjarosik/geranos/pkg/transporter"
)

func NewCmdPush() *cobra.Command {
	var (
		mountedReference string // Declares a variable to hold the value of the "--mountable-image" flag.
	)

	var pushCmd = &cobra.Command{
		Use:   "push [image name]",
		Short: "Push a large file as an OCI image to a registry.",
		Long:  `Uploads a specified file from the local system and packages it as an OCI image to be pushed to a specified container registry.`,
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
			}

			// Since mountedReference is directly bound to the flag,
			// we can just check if it's not empty and append the option.
			if mountedReference != "" {
				ref, err := name.ParseReference(mountedReference, name.StrictValidation)
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
	pushCmd.Flags().StringVar(&mountedReference, "mount", "",
		"Specifies an image reference that can be mounted to avoid uploading layers that exists in the registry")

	return pushCmd
}
