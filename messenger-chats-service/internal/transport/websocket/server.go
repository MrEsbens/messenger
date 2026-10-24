package websocket

import (
	"context"
	"fmt"
	"net/http"
	"time"
)

// Server представляет WebSocket сервер
type Server struct {
	httpServer *http.Server
	hub        *Hub
	port       int
}

// NewServer создаёт новый сервер
func NewServer(hub *Hub, port int) *Server {
	handler := NewHandler(hub)
	mux := http.NewServeMux()
	mux.Handle("/ws", handler)

	return &Server{
		httpServer: &http.Server{
			Addr:         fmt.Sprintf(":%d", port),
			Handler:      mux,
			ReadTimeout:  10 * time.Second,
			WriteTimeout: 10 * time.Second,
			IdleTimeout:  60 * time.Second,
		},
		hub:  hub,
		port: port,
	}
}

// Start запускает сервер
func (s *Server) Start(ctx context.Context) error {
	// Запускаем хаб в отдельной горутине
	go s.hub.Run()

	done := make(chan error, 1)
	go func() {
		if err := s.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			done <- fmt.Errorf("websocket server error: %w", err)
		}
		close(done)
	}()

	// Ожидание контекста для мягкого завершения
	<-ctx.Done()

	// Graceful shutdown
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	s.hub.Stop()
	if err := s.httpServer.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("server shutdown error: %w", err)
	}

	return <-done
}
