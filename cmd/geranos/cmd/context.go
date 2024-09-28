package cmd

import (
	"fmt"
	"github.com/mobileinf/geranos/pkg/appconfig"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func NewCmdContext() *cobra.Command {
	contextCmd := &cobra.Command{
		Use:   "context",
		Short: "Manage contexts",
	}

	var contextSetCmd = &cobra.Command{
		Use:   "set [name] --registry=REGISTRY --user=USER --password=PASSWORD",
		Short: "Set a new context or modify an existing one",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			name := args[0]
			registry, _ := cmd.Flags().GetString("registry")
			user, _ := cmd.Flags().GetString("user")
			password, _ := cmd.Flags().GetString("password")

			newContext := appconfig.Context{Name: name, Registry: registry, User: user, Password: password}

			// Add or update context
			found := false
			for i, ctx := range TheAppConfig.Contexts {
				if ctx.Name == name {
					TheAppConfig.Contexts[i] = newContext
					found = true
					break
				}
			}
			if !found {
				TheAppConfig.Contexts = append(TheAppConfig.Contexts, newContext)
			}

			viper.Set("contexts", TheAppConfig.Contexts)
			if err := viper.WriteConfig(); err != nil {
				fmt.Println("Error writing config:", err)
			}
			fmt.Printf("Context %s set/updated successfully.\n", name)
		},
	}

	contextSetCmd.Flags().String("registry", "", "Registry URL")
	contextSetCmd.Flags().String("user", "", "Registry username")
	contextSetCmd.Flags().String("password", "", "Registry password")

	var contextUnsetCmd = &cobra.Command{
		Use:   "unset",
		Short: "Unset the current context, leaving no active context",
		Run: func(cmd *cobra.Command, args []string) {
			// Check if there is a current context set
			currentContext := viper.GetString("current_context")
			if currentContext == "" {
				fmt.Println("No current context is set.")
				return
			}

			// Unset the current context by setting it to an empty value
			viper.Set("current_context", "")

			// Write the changes to the config file
			if err := viper.WriteConfig(); err != nil {
				fmt.Printf("Error updating config: %v\n", err)
				return
			}

			fmt.Println("Context unset successfully. No active context.")
		},
	}

	var contextUseCmd = &cobra.Command{
		Use:   "use [name]",
		Short: "Switch to a different context",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			name := args[0]

			// Check if context exists
			found := false
			for _, ctx := range TheAppConfig.Contexts {
				if ctx.Name == name {
					TheAppConfig.CurrentContext = name
					viper.Set("current_context", name)
					if err := viper.WriteConfig(); err != nil {
						fmt.Println("Error writing config:", err)
					}
					found = true
					fmt.Printf("Switched to context %s.\n", name)
					break
				}
			}
			if !found {
				fmt.Printf("Context %s not found.\n", name)
			}
		},
	}

	var contextGetCmd = &cobra.Command{
		Use:   "get",
		Short: "Get details of the current context",
		Run: func(cmd *cobra.Command, args []string) {
			for _, ctx := range TheAppConfig.Contexts {
				if ctx.Name == TheAppConfig.CurrentContext {
					fmt.Printf("Current context: %s\nRegistry: %s\nUser: %s\n",
						ctx.Name, ctx.Registry, ctx.User)
					return
				}
			}
			fmt.Println("No current context set.")
		},
	}

	var contextListCmd = &cobra.Command{
		Use:   "list",
		Short: "List all available contexts",
		Run: func(cmd *cobra.Command, args []string) {
			for _, ctx := range TheAppConfig.Contexts {
				status := ""
				if ctx.Name == TheAppConfig.CurrentContext {
					status = "(current)"
				}
				fmt.Printf("%s %s\n", ctx.Name, status)
			}
		},
	}

	var contextDeleteCmd = &cobra.Command{
		Use:   "delete [name]",
		Short: "Delete a context",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			name := args[0]

			newContexts := []appconfig.Context{}
			found := false
			for _, ctx := range TheAppConfig.Contexts {
				if ctx.Name != name {
					newContexts = append(newContexts, ctx)
				} else {
					found = true
				}
			}

			if !found {
				fmt.Printf("Context %s not found.\n", name)
				return
			}

			TheAppConfig.Contexts = newContexts
			viper.Set("contexts", newContexts)
			if err := viper.WriteConfig(); err != nil {
				fmt.Println("Error writing config:", err)
			}

			fmt.Printf("Context %s deleted successfully.\n", name)
		},
	}

	contextCmd.AddCommand(
		contextSetCmd,
		contextUnsetCmd,
		contextUseCmd,
		contextGetCmd,
		contextListCmd,
		contextDeleteCmd)

	return contextCmd
}
