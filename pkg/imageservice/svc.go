package imageservice

import (
	"context"
	"fmt"
	"github.com/tomekjarosik/geranos/pkg/layout"
	"google.golang.org/grpc"
	runtimeapi "k8s.io/cri-api/pkg/apis/runtime/v1"
	"log"
)

type Server struct {
	rootDir string
	runtimeapi.UnimplementedImageServiceServer
}

func (s *Server) ListImages(ctx context.Context, in *runtimeapi.ListImagesRequest) (*runtimeapi.ListImagesResponse, error) {
	lm := layout.NewMapper(s.rootDir)
	props, err := lm.List()
	if err != nil {
		return nil, fmt.Errorf("unable to obtain list of images: %w", err)
	}
	images := make([]*runtimeapi.Image, 0, len(props))
	for _, prop := range props {
		images = append(images, &runtimeapi.Image{
			Id:          prop.Ref.String(),
			RepoTags:    []string{prop.Ref.String()},
			RepoDigests: []string{},
			Size_:       uint64(prop.Size),
			Uid:         nil,
			Username:    "testuser",
			Spec: &runtimeapi.ImageSpec{
				Image:              prop.Ref.String(),
				Annotations:        nil,
				UserSpecifiedImage: "testx",
				RuntimeHandler:     "curie",
			},
			Pinned: false,
		})
	}
	return &runtimeapi.ListImagesResponse{Images: images}, nil
}

func (s *Server) ImageStatus(ctx context.Context, in *runtimeapi.ImageStatusRequest) (*runtimeapi.ImageStatusResponse, error) {
	return &runtimeapi.ImageStatusResponse{}, nil
}

func (s *Server) ImageFsInfo(ctx context.Context, in *runtimeapi.ImageFsInfoRequest) (*runtimeapi.ImageFsInfoResponse, error) {
	return &runtimeapi.ImageFsInfoResponse{}, nil
}

func (s *Server) PullImage(ctx context.Context, in *runtimeapi.PullImageRequest) (*runtimeapi.PullImageResponse, error) {
	return &runtimeapi.PullImageResponse{}, nil
}

func (s *Server) RemoveImage(ctx context.Context, in *runtimeapi.RemoveImageRequest) (*runtimeapi.RemoveImageResponse, error) {
	return &runtimeapi.RemoveImageResponse{}, nil
}

// LoggingInterceptor is an example interceptor that logs requests.
func LoggingInterceptor(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
	log.Printf("Received RPC: %v", info.FullMethod)
	// Pass on to the handler to complete the normal execution of a unary RPC.
	return handler(ctx, req)
}

func NewImageService(dir string) *Server {
	return &Server{rootDir: dir}
}
