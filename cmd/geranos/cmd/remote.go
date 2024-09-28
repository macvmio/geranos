package cmd

import (
	"fmt"
	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/mobileinf/geranos/pkg/transporter"
	"github.com/spf13/cobra"
)

func NewCmdRemoteRepos() *cobra.Command {

	var remoteReposCmd = &cobra.Command{
		Use:       "remote",
		Short:     "Manipulate remote repositories",
		Long:      `Manipulate remote repositories`,
		ValidArgs: []string{"catalog", "images", "tag"},
		Args:      cobra.MatchAll(cobra.ExactArgs(1), cobra.OnlyValidArgs),
		Run: func(cmd *cobra.Command, args []string) {
		},
	}

	var catalogCmd = &cobra.Command{
		Use:   "catalog [remote name]",
		Short: "List remote repositories",
		Long:  `List remote repositories`,
		Args:  cobra.MaximumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) == 0 {
				args = append(args, TheAppConfig.CurrentRegistry())
			}
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
			args[0] = TheAppConfig.Override(args[0])
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

	var tagImage = &cobra.Command{
		Use:   "tag <srcRef> <dstRef>",
		Short: "Tag remotely src tag as dst tag",
		Long:  `This is operation on remote: source tag will be retagged as destination tag`,
		Args:  cobra.ExactArgs(2),
		Run: func(cmd *cobra.Command, args []string) {
			args[0] = TheAppConfig.Override(args[0])
			args[1] = TheAppConfig.Override(args[1])
			err := transporter.RetagRemotely(args[0], args[1])
			if err != nil {
				fmt.Printf("Unable to retag '%s' to '%s': %v\n", args[0], args[1], err)
			}
		},
	}

	remoteReposCmd.AddCommand(catalogCmd)
	remoteReposCmd.AddCommand(listImages)
	remoteReposCmd.AddCommand(tagImage)
	return remoteReposCmd
}
