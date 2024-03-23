package cmd

import (
	"fmt"
	"github.com/spf13/cobra"
	"os"
)

func InitializeCommands() *cobra.Command {
	var rootCmd = &cobra.Command{
		Use:   "geranos",
		Short: "Geranos is a tool to transport big files as OCI images.",
		Long: `Geranos is designed to efficiently transport large files packaged as OCI container images,
ensuring fast, reliable, and secure transfers across different environments.
It relies on sparse files and Copy-on-Write filesystem features to optimize disk usage`,
		// This function can be used to execute any code when the root command is called without any subcommands
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println(cmd.Short)
		},
	}

	rootCmd.AddCommand(createPullCommand())
	rootCmd.AddCommand(createPushCommand())
	rootCmd.AddCommand(createInspectCommand())
	rootCmd.AddCommand(createListCommand())

	return rootCmd
}

func Execute(rootCmd *cobra.Command) {
	rootCmd.Version = "v1"
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %s\n", err)
		os.Exit(-1)
	}
}
