package services

import (
	"bytes"
	"context"
	"fmt"
	"image/png"
	"log"

	"github.com/mbenaiss/whatsapp-mcp/db"
	"github.com/mbenaiss/whatsapp-mcp/models"
	"github.com/mbenaiss/whatsapp-mcp/whatsapp"
	"github.com/skip2/go-qrcode"
)

type Service interface {
	GetStatus() (models.Status, error)
	SendMessage(ctx context.Context, recipient string, message string) error
	GetChats(ctx context.Context) ([]models.Chat, error)
	GetMessages(ctx context.Context, chatJID string, limit int) ([]models.Message, error)
	GetQR(ctx context.Context) ([]byte, error)
	IsConnected() bool
	Login(ctx context.Context) error
}

type service struct {
	whatsapp *whatsapp.Whatsapp
	db       db.DB
}

// NewService creates a new Service instance with the provided WhatsApp client
func NewService(whatsapp *whatsapp.Whatsapp, db db.DB) Service {
	s := &service{whatsapp: whatsapp, db: db}

	go func() {
		for chat := range whatsapp.ChatChan {
			err := s.storeChatAndMessage(context.Background(), chat)
			if err != nil {
				fmt.Println("Error storing chat and message:", err)
			}
		}
	}()

	return s
}

// GetQR returns the QR code for the WhatsApp client
func (s *service) GetQR(ctx context.Context) ([]byte, error) {
	if s.IsConnected() || s.whatsapp.IsLoggedIn() {
		log.Println("WhatsApp is already connected")
		return nil, nil
	}

	qr, err := s.whatsapp.GetQR(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get QR code: %v", err)
	}

	if qr != "" {
		qrCode, err := qrcode.New(qr, qrcode.Medium)
		if err != nil {
			return nil, fmt.Errorf("failed to generate QR code image: %v", err)
		}

		var buf bytes.Buffer
		png.Encode(&buf, qrCode.Image(256))

		return buf.Bytes(), nil
	}

	return nil, nil
}

// Login connects to the WhatsApp client
func (s *service) Login(ctx context.Context) error {
	err := s.whatsapp.Connect()
	if err != nil {
		return fmt.Errorf("failed to connect: %v", err)
	}

	// err = s.whatsapp.BuildHistorySync(ctx)
	// if err != nil {
	// 	return fmt.Errorf("failed to build history sync: %v", err)
	// }

	return nil
}

// GetStatus returns the current status of the WhatsApp client
func (s *service) GetStatus() (models.Status, error) {
	return s.whatsapp.GetStatus()
}

// SendMessage sends a message to the specified recipient
func (s *service) SendMessage(ctx context.Context, recipient string, message string) error {
	return s.whatsapp.SendMessage(ctx, recipient, message)
}

// GetChats retrieves all available chats
func (s *service) GetChats(ctx context.Context) ([]models.Chat, error) {
	chats, err := s.db.GetChats(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get chats: %v", err)
	}

	return chats, nil
}

// GetMessages retrieves messages from a specific chat with the given limit
func (s *service) GetMessages(ctx context.Context, chatJID string, limit int) ([]models.Message, error) {
	return s.db.GetMessages(ctx, chatJID, limit)
}

// IsConnected checks if the WhatsApp client is connected
func (s *service) IsConnected() bool {
	return s.whatsapp.IsConnected()
}

func (s *service) storeChatAndMessage(ctx context.Context, chat models.Chat) error {
	err := s.db.StoreChat(ctx, chat)
	if err != nil {
		return fmt.Errorf("error storing chat: %v", err)
	}

	for _, msg := range chat.Messages {
		err = s.db.StoreMessage(ctx, msg)
		if err != nil {
			return fmt.Errorf("error storing message: %v", err)
		}
	}

	return nil
}
