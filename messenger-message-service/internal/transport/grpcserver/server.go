package grpcserver

import (
	"fmt"
	"net"

	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/reflection"

	messagesv1 "github.com/MrEsbens/messenger-message-service/api/messages/v1"
	"github.com/MrEsbens/messenger-message-service/internal/service"
)

type Server struct {
	*grpc.Server
	port string
}

func NewServer(port int, messageService service.MessageServiceInterface) (*Server, error) {
	grpcServer := grpc.NewServer()

	healthServer := health.NewServer()
	healthServer.SetServingStatus("", healthpb.HealthCheckResponse_SERVING)
	healthpb.RegisterHealthServer(grpcServer, healthServer)

	handler := NewGRPCHandler(messageService)
	messagesv1.RegisterMessageServiceServer(grpcServer, handler)

	reflection.Register(grpcServer)

	return &Server{
		Server: grpcServer,
		port:   fmt.Sprintf(":%d", port),
	}, nil
}

func (s *Server) Start() error {
	lis, err := net.Listen("tcp", s.port)
	if err != nil {
		return fmt.Errorf("failed to listen on %s: %w", s.port, err)
	}

	fmt.Printf("🚀 gRPC server listening at %s\n", s.port)

	if err := s.Serve(lis); err != nil {
		return fmt.Errorf("gRPC server failed: %w", err)
	}

	return nil
}

func (s *Server) Stop() {
	if s.Server != nil {
		fmt.Println("🛑 Stopping gRPC server...")
		s.GracefulStop()
	}
}
