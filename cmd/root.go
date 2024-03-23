package cmd

import (
	"fmt"
	"github.com/spf13/cobra"
	"os"
	"strings"
)

func InitializeCommands() *cobra.Command {
	cobra.OnInitialize(initConfig)
	var rootCmd = &cobra.Command{
		Use:   "geranos",
		Short: "Geranos is a tool to transport big files as OCI images.",
		Long: `Geranos is designed to efficiently transport large files packaged as OCI container images,
ensuring fast, reliable, and secure transfers across different environments.
It relies on sparse files and Copy-on-Write filesystem features to optimize disk usage`,
		// This function can be used to execute any code when the root command is called without any subcommands
		Args:                       cobra.ExactArgs(1),
		SuggestionsMinimumDistance: 2,
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Println(cmd.Short)
			return nil
		},
	}

	// Customizing unknown command handling
	rootCmd.SilenceErrors = false
	rootCmd.SilenceUsage = false
	rootCmd.SetHelpFunc(func(cmd *cobra.Command, args []string) {
		fmt.Printf("Error: '%s' is not a known command\n\n", strings.Join(args, " "))
		fmt.Println("Use 'geranos --help' for a list of available commands.")
	})

	rootCmd.AddCommand(createPullCommand())
	rootCmd.AddCommand(createPushCommand())
	rootCmd.AddCommand(createInspectCommand())
	rootCmd.AddCommand(createListCommand())
	rootCmd.AddCommand(createAdoptCommand())
	rootCmd.AddCommand(createCloneCommand())
	rootCmd.AddCommand(createRemoveCommand())

	return rootCmd
}

func Execute(rootCmd *cobra.Command) {
	rootCmd.Version = "v1"
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %s\n", err)
		os.Exit(-1)
	}
}
