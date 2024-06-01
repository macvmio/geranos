package cmd

import (
	"fmt"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/tomekjarosik/geranos/pkg/imageservice"
	"google.golang.org/grpc"
	runtimeapi "k8s.io/cri-api/pkg/apis/runtime/v1"
	"log"
	"net"
)

func NewServeCommand() *cobra.Command {
	var serveCommand = &cobra.Command{
		Use:   "serve",
		Short: "TODO",
		Long:  `TODO`,
		Args:  cobra.ExactArgs(0),
		Run: func(cmd *cobra.Command, args []string) {
			socketPath := viper.GetString("SOCKET")
			if socketPath == "" {
				socketPath = "/var/run/geranos/geranosd.sock"
			}
			lis, err := net.Listen("unix", socketPath)
			if err != nil {
				log.Fatalf("failed to listen: %v", err)
			}

			imagesDirectory := viper.GetString("images_directory")

			// Creates a new gRPC server with the unary interceptor
			s := grpc.NewServer(grpc.UnaryInterceptor(imageservice.LoggingInterceptor))
			runtimeapi.RegisterImageServiceServer(s, imageservice.NewImageService(imagesDirectory))

			go func() {
				<-cmd.Context().Done()
				s.GracefulStop()
				fmt.Println("stopped gracefully")
			}()

			if err := s.Serve(lis); err != nil {
				log.Fatalf("failed to serve: %v", err)
			}
		},
	}
	return serveCommand
}
