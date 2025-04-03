package api

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/mbenaiss/whatsapp-mcp/services"
)

// Server represents the API handler
type Server struct {
	service services.Service
	router  *gin.Engine
	server  *http.Server
}

// NewServer creates a new API server
func NewServer(service services.Service, port string) *Server {
	router := gin.Default()

	return &Server{
		service: service,
		router:  router,
		server: &http.Server{
			Addr:    ":" + port,
			Handler: router,
		},
	}
}

// SendMessageRequest represents the request body for sending messages
type SendMessageRequest struct {
	Recipient string `json:"recipient"`
	Message   string `json:"message"`
}

// Response represents a generic API response
type Response struct {
	Success bool   `json:"success"`
	Message string `json:"message,omitempty"`
	Data    any    `json:"data,omitempty"`
}

// RegisterRoutes registers all API routes
func (s *Server) registerRoutes(router *gin.Engine) {
	api := router.Group("/api")
	{
		api.GET("/login", s.handleLogin)
		api.GET("/qr", s.handleQR)
		api.GET("/status", s.handleStatus)
		api.POST("/send", s.handleSendMessage)
		api.GET("/chats", s.handleGetChats)
		api.GET("/messages", s.handleGetMessages)
	}
}

func (s *Server) Start() error {
	s.registerRoutes(s.router)

	return s.server.ListenAndServe()
}

func (s *Server) Stop(ctx context.Context) error {
	return s.server.Shutdown(ctx)
}
