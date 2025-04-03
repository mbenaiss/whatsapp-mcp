package api

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

func (s *Server) handleQR(c *gin.Context) {
	qrCode, err := s.service.GetQR(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, Response{
			Success: false,
			Message: fmt.Sprintf("Failed to get QR code: %v", err),
		})
		return
	}

	time.Sleep(2 * time.Second)

	if !s.service.IsConnected() {
		c.JSON(http.StatusInternalServerError, Response{
			Success: false,
			Message: "Failed to establish stable connection",
		})
		return
	}

	// If qrCode is empty, it means we're already connected
	if qrCode == nil {
		c.JSON(http.StatusOK, Response{
			Success: true,
			Message: "Already connected to WhatsApp",
		})
		return
	}

	c.Header("Content-Type", "image/png")
	c.Data(http.StatusOK, "image/png", qrCode)
}

func (s *Server) handleStatus(c *gin.Context) {
	status, err := s.service.GetStatus()
	if err != nil {
		c.JSON(http.StatusInternalServerError, Response{
			Success: false,
			Message: fmt.Sprintf("Failed to get status: %v", err),
		})
		return
	}

	c.JSON(http.StatusOK, Response{
		Success: true,
		Data:    status,
	})
}

func (s *Server) handleSendMessage(c *gin.Context) {
	var req SendMessageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, Response{
			Success: false,
			Message: "Invalid request body",
		})
		return
	}

	// Parse recipient JID
	recipient := req.Recipient
	if recipient == "" || req.Message == "" {
		c.JSON(http.StatusBadRequest, Response{
			Success: false,
			Message: "Recipient and message are required",
		})
		return
	}

	err := s.service.SendMessage(c.Request.Context(), recipient, req.Message)
	if err != nil {
		c.JSON(http.StatusInternalServerError, Response{
			Success: false,
			Message: fmt.Sprintf("Failed to send message: %v", err),
		})
		return
	}

	c.JSON(http.StatusOK, Response{
		Success: true,
		Message: "Message sent successfully",
	})
}

func (s *Server) handleGetChats(c *gin.Context) {
	chats, err := s.service.GetChats(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, Response{
			Success: false,
			Message: fmt.Sprintf("Failed to get chats: %v", err),
		})
		return
	}

	c.JSON(http.StatusOK, Response{
		Success: true,
		Data:    chats,
	})
}

func (s *Server) handleGetMessages(c *gin.Context) {
	chatJID := c.Query("chat")
	if chatJID == "" {
		c.JSON(http.StatusBadRequest, Response{
			Success: false,
			Message: "Missing chat parameter",
		})
		return
	}

	limit := 50 // Default limit
	if limitStr := c.Query("limit"); limitStr != "" {
		if n, err := strconv.Atoi(limitStr); err == nil && n > 0 {
			limit = n
		}
	}

	messages, err := s.service.GetMessages(c.Request.Context(), chatJID, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, Response{
			Success: false,
			Message: fmt.Sprintf("Failed to get messages: %v", err),
		})
		return
	}

	c.JSON(http.StatusOK, Response{
		Success: true,
		Data:    messages,
	})
}

func (s *Server) handleLogin(c *gin.Context) {
	err := s.service.Login(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, Response{
			Success: false,
			Message: fmt.Sprintf("Failed to login: %v", err),
		})
		return
	}

	c.JSON(http.StatusOK, Response{
		Success: true,
		Message: "Login successful",
	})
}
