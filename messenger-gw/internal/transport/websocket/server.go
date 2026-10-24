package websocket

import (
	"context"
	"log"
	"net/http"
	"time"

	"github.com/MrEsbens/messenger-gw/internal/config"
	"github.com/MrEsbens/messenger-gw/internal/transport/grpcclient/chats"
	"github.com/MrEsbens/messenger-gw/internal/transport/grpcclient/file"
	"github.com/MrEsbens/messenger-gw/internal/transport/grpcclient/identity"
	"github.com/MrEsbens/messenger-gw/internal/transport/grpcclient/messages"
	"github.com/MrEsbens/messenger-gw/internal/transport/grpcclient/servers"
)

type Server struct {
	httpServer *http.Server
	hub        *Hub
	handler    *WSHandler
	log        *log.Logger
}

func NewServer(
	cfg *config.Config,
	mc messages.MessagesClientInterface,
	ic identity.IdentityClientInterface,
	cc chats.ChatsClientInterface,
	sc servers.ServersClientInterface,
	fc file.FileClientInterface,
) *Server {
	h := NewHub(cfg)
	upgrader := NewUpgrader()
	hdlr := NewWSHandler(h, upgrader, cfg.JWTSecret, mc, ic, cc, sc, fc)

	h.SetMessageHandler(hdlr.HandleMessage)

	mux := http.NewServeMux()
	mux.HandleFunc("/ws", hdlr.ServeHTTP)
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	httpServer := &http.Server{
		Addr:         ":" + cfg.HTTPPort,
		Handler:      mux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	return &Server{
		httpServer: httpServer,
		hub:        h,
		handler:    hdlr,
		log:        log.Default(),
	}
}

func (s *Server) Start(ctx context.Context) {
	go s.hub.Run(ctx)
	s.log.Printf("WebSocket server starting on %s", s.httpServer.Addr)

	go func() {
		if err := s.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			s.log.Printf("HTTP server error: %v", err)
		}
	}()
}

func (s *Server) Shutdown(ctx context.Context) error {
	return s.httpServer.Shutdown(ctx)
}

func (s *Server) GetHandler() *WSHandler {
	return s.handler
}
