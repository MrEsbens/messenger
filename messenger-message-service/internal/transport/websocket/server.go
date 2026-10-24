package websocket

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/MrEsbens/messenger-message-service/internal/service"
	"github.com/MrEsbens/messenger-message-service/pkg/redis"
	"github.com/gorilla/websocket"
)

type Server struct {
	httpServer *http.Server
	hub        *Hub
	port       int
	upgrader   websocket.Upgrader
	mu         sync.Mutex
	running    bool
}

func NewWebSocketServer(
	port int,
	redisClient *redis.Client,
	messageService service.MessageServiceInterface,
	redisPrefix string,
) *Server {
	handler := NewHandler(messageService, redisPrefix)

	hub := NewHub(redisClient, redisPrefix, handler)

	upgrader := websocket.Upgrader{
		// Размер буфера для чтения (должен быть >= maxMessageSize в client.go)
		ReadBufferSize: 1024,
		// Размер буфера для записи
		WriteBufferSize: 1024,

		CheckOrigin: func(r *http.Request) bool {
			// 🔥 В продакшене заменить на проверку белого списка:
			// origin := r.Header.Get("Origin")
			// return origin == "https://app.messenger.com" || origin == "https://messenger.com"
			return true // ⚠️ Только для разработки!
		},

		Error: func(w http.ResponseWriter, r *http.Request, status int, reason error) {
			log.Printf("⚠️ WebSocket upgrade error: status=%d, reason=%v, path=%s",
				status, reason, r.URL.Path)
		},
	}

	mux := http.NewServeMux()

	mux.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		userID, err := ExtractUserFromRequest(r)
		if err != nil {
			http.Error(w, "unauthorized: "+err.Error(), http.StatusUnauthorized)
			return
		}

		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			log.Printf("❌ Failed to upgrade connection for user %s: %v", userID, err)
			return
		}

		client := NewClient(userID, conn, hub)

		select {
		case hub.Register <- client:
		default:
			log.Printf("⚠️ Hub register channel full, closing client %s", userID)
			_ = conn.Close()
			return
		}

		go client.Start()
	})

	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		stats := hub.GetStats()
		if !stats.IsRunning {
			http.Error(w, "hub not running", http.StatusServiceUnavailable)
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	mux.HandleFunc("/stats", func(w http.ResponseWriter, r *http.Request) {
		stats := hub.GetStats()
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(stats)
	})

	httpServer := &http.Server{
		Addr:         fmt.Sprintf(":%d", port),
		Handler:      mux,
		ReadTimeout:  15 * time.Second, // Макс. время на чтение заголовков запроса
		WriteTimeout: 15 * time.Second, // Макс. время на запись ответа
		IdleTimeout:  60 * time.Second, // Макс. время простоя соединения (для keep-alive)
	}

	return &Server{
		httpServer: httpServer,
		hub:        hub,
		port:       port,
		upgrader:   upgrader,
		running:    false,
	}
}

func (s *Server) Start(ctx context.Context) error {
	s.mu.Lock()
	if s.running {
		s.mu.Unlock()
		return fmt.Errorf("server is already running on port %d", s.port)
	}
	s.running = true
	s.mu.Unlock()

	go s.hub.Run()

	log.Printf("🚀 WebSocket server starting on port %d", s.port)
	log.Printf("📡 WebSocket endpoint: ws://localhost:%d/ws", s.port)
	log.Printf("🏥 Health check: http://localhost:%d/health", s.port)

	errChan := make(chan error, 1)
	go func() {
		if err := s.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errChan <- fmt.Errorf("HTTP server failed: %w", err)
		}
		close(errChan)
	}()

	select {
	case err := <-errChan:
		s.mu.Lock()
		s.running = false
		s.mu.Unlock()
		return err

	case <-ctx.Done():
		log.Println("🛑 Received shutdown signal, stopping WebSocket server...")
		return s.Stop(ctx)
	}
}

func (s *Server) Stop(ctx context.Context) error {
	s.mu.Lock()
	if !s.running {
		s.mu.Unlock()
		return nil // Уже остановлен
	}
	s.running = false
	s.mu.Unlock()

	s.hub.Stop()

	stopCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	if err := s.httpServer.Shutdown(stopCtx); err != nil {
		_ = s.httpServer.Close()
		return fmt.Errorf("failed to shutdown HTTP server: %w", err)
	}

	log.Printf("✅ WebSocket server stopped on port %d", s.port)
	return nil
}

func (s *Server) IsRunning() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.running
}

func (s *Server) GetHub() *Hub {
	return s.hub
}

func (s *Server) GetPort() int {
	return s.port
}

// ───────────────────────────────────────────────────────────
// Middleware (опционально, для расширения)
// ───────────────────────────────────────────────────────────

// WithLogging добавляет логирование всех запросов
// Можно использовать для отладки или аудита
func WithLogging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		log.Printf("📥 %s %s", r.Method, r.URL.Path)
		next.ServeHTTP(w, r)
		log.Printf("📤 %s %s (%v)", r.Method, r.URL.Path, time.Since(start))
	})
}

// WithCORS добавает заголовки CORS для браузерных клиентов
// В продакшене настраивать белый список доменов
func WithCORS(allowedOrigins []string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")

			// Проверяем, разрешён ли этот origin
			allowed := false
			for _, o := range allowedOrigins {
				if o == "*" || o == origin {
					allowed = true
					break
				}
			}

			if allowed {
				w.Header().Set("Access-Control-Allow-Origin", origin)
				w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
				w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-User-ID")
			}

			// Обрабатываем preflight запрос
			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusOK)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
