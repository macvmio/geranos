package cmd

import "github.com/spf13/cobra"

func createListCommand() *cobra.Command {
	var listCmd = &cobra.Command{
		Use:   "list [repository]",
		Short: "List all OCI images in a specific repository.",
		Long:  `Lists all available OCI images in the specified container registry or repository, providing a quick overview of the stored images.`,
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			// Implementation of the list functionality
		},
	}

	return listCmd
}
