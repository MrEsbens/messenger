// internal/transport/grpcserver/server.go (в messenger-file-service)

package grpcserver

import (
	"fmt"
	"net"

	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/reflection"

	"github.com/MrEsbens/messenger-file-service/internal/service"
	filev1 "github.com/MrEsbens/messenger-file-service/api/file/v1"
)

type Server struct {
	*grpc.Server
	port string
}

func NewServer(port string, fileSvc *service.FileService) (*Server, error) {
	grpcServer := grpc.NewServer()

	healthServer := health.NewServer()
	healthServer.SetServingStatus("", healthpb.HealthCheckResponse_SERVING)
	healthpb.RegisterHealthServer(grpcServer, healthServer)

	filev1.RegisterFileServiceServer(grpcServer, fileSvc)

	reflection.Register(grpcServer)

	return &Server{
		Server: grpcServer,
		port:   port,
	}, nil
}

func (s *Server) Start() error {
	lis, err := net.Listen("tcp", s.port)
	if err != nil {
		return fmt.Errorf("failed to listen: %v", err)
	}
	fmt.Printf("🚀 File gRPC server listening at %s\n", s.port)
	return s.Serve(lis)
}
