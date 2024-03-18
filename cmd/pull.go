package cmd

import "github.com/spf13/cobra"

func createPullCommand() *cobra.Command {
	var pullCmd = &cobra.Command{
		Use:   "pull [image name] [destination path]",
		Short: "Pull an OCI image from a registry and extract the file.",
		Long:  `Downloads an OCI image from a specified container registry and extracts the file to a specified local path.`,
		Args:  cobra.ExactArgs(2),
		Run: func(cmd *cobra.Command, args []string) {
			// Implementation of the pull functionality
		},
	}

	return pullCmd
}
