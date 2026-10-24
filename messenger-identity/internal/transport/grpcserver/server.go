package grpcserver

import (
	"fmt"
	"net"

	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/reflection"

	identityv1 "github.com/MrEsbens/messenger-identity/api/identity/v1"
	"github.com/MrEsbens/messenger-identity/internal/service"
)

type Server struct {
	*grpc.Server
	port string
}

func NewServer(port string, identitySvc *service.IdentityService) (*Server, error) {
	grpcServer := grpc.NewServer()

	healthServer := health.NewServer()
	healthServer.SetServingStatus("", healthpb.HealthCheckResponse_SERVING)
	healthpb.RegisterHealthServer(grpcServer, healthServer)

	identityv1.RegisterIdentityServiceServer(grpcServer, identitySvc)

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
	
	fmt.Printf("🚀 gRPC server listening at %s\n", s.port)
	return s.Serve(lis)
}