package cmd

import (
	"context"
	"fmt"
	"github.com/google/go-containerregistry/cmd/crane/cmd"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"os"
	"os/signal"
)

func InitializeCommands() *cobra.Command {
	var rootCmd = &cobra.Command{
		Use:   "geranos",
		Short: "Geranos is a tool to transport big files as OCI images.",
		Long: `Geranos is designed to efficiently transport large files packaged as OCI container images,
ensuring fast, reliable, and secure transfers across different environments.
It relies on sparse files and Copy-on-Write filesystem features to optimize disk usage`,
		// This function can be used to execute any code when the root command is called without any subcommands
		Args:                       cobra.ExactArgs(1),
		SuggestionsMinimumDistance: 2,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			if err := initConfig(); err != nil {
				return fmt.Errorf("failed to initialize config: %v", err)
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Println(cmd.Short)
			return nil
		},
	}

	// Define the --verbose global flag
	var verbose bool
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "enable verbose output")

	// Bind the verbose flag to Viper
	viper.BindPFlag("verbose", rootCmd.PersistentFlags().Lookup("verbose"))

	rootCmd.AddCommand(
		NewCmdPull(),
		NewCmdPush(),
		NewCmdInspect(),
		NewCmdList(),
		NewCmdAdopt(),
		NewCmdClone(),
		NewCmdRemove(),
		NewCmdAuthLogin(),
		NewCmdAuthLogout(),
		NewCmdVersion(),
		NewCmdRemoteRepos(),
		NewCmdContext(),
		NewCmdRehash(),
	)

	return rootCmd
}

func Execute(rootCmd *cobra.Command) {
	rootCmd.Version = cmd.Version
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()
	if err := rootCmd.ExecuteContext(ctx); err != nil {
		cancel()
		fmt.Fprintf(os.Stderr, "Error: %s\n", err)
		os.Exit(1)
	}
}
