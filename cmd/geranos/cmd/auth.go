package cmd

import (
	"errors"
	"github.com/docker/cli/cli/config"
	"github.com/docker/cli/cli/config/types"
	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/name"
	"github.com/spf13/cobra"
	"golang.org/x/term"
	"log"
	"os"
	"strings"
	"syscall"
)

// Inspired by https://github.com/google/go-containerregistry/blob/main/cmd/crane/cmd/auth.go

// NewCmdAuthLogin creates a new `crane auth login` command.
func NewCmdAuthLogin() *cobra.Command {
	var opts loginOptions

	eg := `  # Log in to oci.jarosik.online
  geranos login oci.jarosik.online -u <usernamne> -p <password>`

	cmd := &cobra.Command{
		Use:     "login [OPTIONS] [SERVER]",
		Short:   "Log in to a registry",
		Example: eg,
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			reg, err := name.NewRegistry(args[0])
			if err != nil {
				return err
			}

			opts.serverAddress = reg.Name()

			return login(opts)
		},
	}

	flags := cmd.Flags()

	flags.StringVarP(&opts.user, "username", "u", "", "Username")
	flags.StringVarP(&opts.password, "password", "p", "", "Password")
	flags.BoolVarP(&opts.passwordStdin, "password-stdin", "", false, "Take the password from stdin")

	return cmd
}

type loginOptions struct {
	serverAddress string
	user          string
	password      string
	passwordStdin bool
}

func login(opts loginOptions) error {
	if opts.passwordStdin {
		bytePassword, err := term.ReadPassword(syscall.Stdin)
		if err != nil {
			return err
		}

		opts.password = strings.TrimSuffix(string(bytePassword), "\n")
		opts.password = strings.TrimSuffix(opts.password, "\r")
	}
	if opts.user == "" || opts.password == "" {
		return errors.New("username and password required")
	}
	cf, err := config.Load(os.Getenv("DOCKER_CONFIG"))
	if err != nil {
		return err
	}
	creds := cf.GetCredentialsStore(opts.serverAddress)
	if opts.serverAddress == name.DefaultRegistry {
		opts.serverAddress = authn.DefaultAuthKey
	}
	if err := creds.Store(types.AuthConfig{
		ServerAddress: opts.serverAddress,
		Username:      opts.user,
		Password:      opts.password,
	}); err != nil {
		return err
	}

	if err := cf.Save(); err != nil {
		return err
	}
	log.Printf("logged in via %s", cf.Filename)
	return nil
}

// NewCmdAuthLogout creates a new `crane auth logout` command.
func NewCmdAuthLogout() *cobra.Command {
	eg := `  # Log out of oci.jarosik.online
  geranos logout oci.jarosik.online`

	cmd := &cobra.Command{
		Use:     "logout [SERVER]",
		Short:   "Log out of a registry",
		Example: eg,
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			reg, err := name.NewRegistry(args[0])
			if err != nil {
				return err
			}
			serverAddress := reg.Name()

			cf, err := config.Load(os.Getenv("DOCKER_CONFIG"))
			if err != nil {
				return err
			}
			creds := cf.GetCredentialsStore(serverAddress)
			if serverAddress == name.DefaultRegistry {
				serverAddress = authn.DefaultAuthKey
			}
			if err := creds.Erase(serverAddress); err != nil {
				return err
			}

			if err := cf.Save(); err != nil {
				return err
			}
			log.Printf("logged out via %s", cf.Filename)
			return nil
		},
	}
	return cmd
}
