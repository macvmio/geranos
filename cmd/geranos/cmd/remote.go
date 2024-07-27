package cmd

import (
	"fmt"
	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/spf13/cobra"
)

func NewCommandRemoteRepos() *cobra.Command {

	var remoteReposCmd = &cobra.Command{
		Use:   "remote",
		Short: "Manipulate remote repositories",
		Long:  `Manipulate remote repositories`,
		Args:  cobra.ExactArgs(0),
		Run: func(cmd *cobra.Command, args []string) {

		},
	}

	var catalogCmd = &cobra.Command{
		Use:   "catalog [remote name]",
		Short: "List remote repositories",
		Long:  `List remote repositories`,
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			reg, err := name.NewRegistry(args[0])
			if err != nil {
				fmt.Println("Error parsing registry name:", err)
				return
			}
			catalog, err := remote.Catalog(cmd.Context(), reg, remote.WithAuthFromKeychain(authn.DefaultKeychain))
			if err != nil {
				fmt.Println("Error fetching catalog:", err)
				return
			}
			for _, repo := range catalog {
				fmt.Println(repo)
			}
		},
	}

	var listImages = &cobra.Command{
		Use:   "images [full qualified repo name]",
		Short: "List remote images in a remote repository",
		Long:  `List remote images in a remote repository`,
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			repo, err := name.NewRepository(args[0])
			if err != nil {
				fmt.Println("Error parsing registry name:", err)
				return
			}
			images, err := remote.List(repo, remote.WithAuthFromKeychain(authn.DefaultKeychain))
			if err != nil {
				fmt.Println("Error fetching repos:", err)
				return
			}
			for _, image := range images {
				fmt.Println(image)
			}
		},
	}

	remoteReposCmd.AddCommand(catalogCmd)
	remoteReposCmd.AddCommand(listImages)
	return remoteReposCmd
}
