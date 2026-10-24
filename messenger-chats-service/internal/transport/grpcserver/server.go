package grpcserver

import (
	"context"
	"fmt"
	"net"

	chatsv1 "github.com/MrEsbens/messenger-chats-service/api/chats/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

type Server struct {
	grpcServer *grpc.Server
	port       int
}

func NewServer(handler *GRPCHandler, port int) *Server {
	grpcServer := grpc.NewServer()
	chatsv1.RegisterChatServiceServer(grpcServer, handler)
	reflection.Register(grpcServer) // для grpcurl и отладки

	return &Server{
		grpcServer: grpcServer,
		port:       port,
	}
}

func (s *Server) Start(ctx context.Context) error {
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", s.port))
	if err != nil {
		return fmt.Errorf("failed to listen: %w", err)
	}

	done := make(chan struct{})
	go func() {
		defer close(done)
		if err := s.grpcServer.Serve(lis); err != nil {
			select {
			case <-ctx.Done():
				// Нормальное завершение
			default:
				fmt.Printf("gRPC server error: %v\n", err)
			}
		}
	}()

	// Ожидание контекста для мягкого завершения
	<-ctx.Done()
	s.grpcServer.GracefulStop()
	<-done

	return nil
}
