package cmd

import "github.com/spf13/cobra"

func createInspectCommand() *cobra.Command {
	var inspectCmd = &cobra.Command{
		Use:   "inspect [image name]",
		Short: "Inspect details of a specific OCI image.",
		Long:  `Provides detailed information about a specific OCI image, such as size, creation date, tags, and custom metadata.`,
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			// Implementation of the inspect functionality
		},
	}

	return inspectCmd
}
