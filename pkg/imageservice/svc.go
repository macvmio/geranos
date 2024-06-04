package imageservice

import (
	"context"
	"fmt"
	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/tomekjarosik/geranos/pkg/layout"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	runtimeapi "k8s.io/cri-api/pkg/apis/runtime/v1"
	"log"
)

type Server struct {
	lm *layout.Mapper
	runtimeapi.UnimplementedImageServiceServer
}

func (s *Server) ListImages(ctx context.Context, in *runtimeapi.ListImagesRequest) (*runtimeapi.ListImagesResponse, error) {
	props, err := s.lm.List()
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

func (s *Server) ImageStatus(ctx context.Context, req *runtimeapi.ImageStatusRequest) (*runtimeapi.ImageStatusResponse, error) {
	imgPtr := req.GetImage()
	if imgPtr == nil {
		return nil, status.Errorf(codes.InvalidArgument, "empty image")
	}
	ref, err := name.ParseReference(imgPtr.Image, name.StrictValidation)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "image refereence parse error: %s", err.Error())
	}
	_, err = s.lm.Read(ctx, ref)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "reading image from disk resulted in error: %s", err.Error())
	}
	return &runtimeapi.ImageStatusResponse{
		Image: &runtimeapi.Image{
			Id:          ref.String(),
			RepoTags:    []string{ref.String()},
			RepoDigests: []string{},
			Size_:       123, // TODO: Read from real size
			Uid:         nil,
			Username:    "todo",
			Spec:        nil,
			Pinned:      false,
		},
		Info: nil,
	}, nil
}

func (s *Server) ImageFsInfo(ctx context.Context, in *runtimeapi.ImageFsInfoRequest) (*runtimeapi.ImageFsInfoResponse, error) {
	return &runtimeapi.ImageFsInfoResponse{}, nil
}

func (s *Server) PullImage(ctx context.Context, req *runtimeapi.PullImageRequest) (*runtimeapi.PullImageResponse, error) {
	a := req.GetAuth()
	opts := make([]remote.Option, 0)
	if a != nil {
		// NOTE: a.ServerAddress is not used yet
		log.Printf("using non-empty auth")
		opts = append(opts, remote.WithAuth(authn.FromConfig(authn.AuthConfig{
			Username:      a.Username,
			Password:      a.Password,
			Auth:          "", // TODO: what is it?
			IdentityToken: a.IdentityToken,
			RegistryToken: a.RegistryToken,
		})))
	} else {
		log.Printf("using default auth")
		opts = append(opts, remote.WithAuthFromKeychain(authn.DefaultKeychain))
	}
	imgPtr := req.GetImage()
	if imgPtr == nil {
		return nil, status.Errorf(codes.InvalidArgument, "empty image")
	}
	ref, err := name.ParseReference(imgPtr.Image, name.StrictValidation)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "image reference parse error: %s", err.Error())
	}
	img, err := remote.Image(ref, opts...)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to pull image: %s", err.Error())
	}
	err = s.lm.Write(ctx, img, ref)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "image write error: %s", err.Error())
	}
	return &runtimeapi.PullImageResponse{
		ImageRef: ref.String(),
	}, nil
}

func (s *Server) RemoveImage(ctx context.Context, req *runtimeapi.RemoveImageRequest) (*runtimeapi.RemoveImageResponse, error) {
	imgPtr := req.GetImage()
	if imgPtr == nil {
		return nil, status.Errorf(codes.InvalidArgument, "empty image")
	}
	ref, err := name.ParseReference(imgPtr.Image, name.StrictValidation)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "image reference parse error: %s", err.Error())
	}
	err = s.lm.Remove(ref)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to remove image: %s", err.Error())
	}
	return &runtimeapi.RemoveImageResponse{}, nil
}

// LoggingInterceptor is an example interceptor that logs requests.
func LoggingInterceptor(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
	log.Printf("Received RPC: %v\n", info.FullMethod)
	log.Printf("  request: %v\n", req)
	// Pass on to the handler to complete the normal execution of a unary RPC.
	res, err := handler(ctx, req)
	log.Printf("End RPC: %v\n", info.FullMethod)
	log.Printf("  result: %v, err=%v", res, err)
	return res, err
}

func NewImageService(dir string) *Server {
	return &Server{lm: layout.NewMapper(dir)}
}
