package cmd

import "github.com/spf13/cobra"

func createPushCommand() *cobra.Command {
	var pushCmd = &cobra.Command{
		Use:   "push [file path] [image name]",
		Short: "Push a large file as an OCI image to a registry.",
		Long:  `Uploads a specified file from the local system and packages it as an OCI image to be pushed to a specified container registry.`,
		Args:  cobra.ExactArgs(2),
		Run: func(cmd *cobra.Command, args []string) {
			// Implementation of the push functionality
		},
	}

	return pushCmd
}
