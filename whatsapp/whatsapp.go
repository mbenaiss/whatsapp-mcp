package whatsapp

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/mbenaiss/whatsapp-mcp/models"
	"github.com/mdp/qrterminal"
	"go.mau.fi/whatsmeow"
	waProto "go.mau.fi/whatsmeow/binary/proto"
	"go.mau.fi/whatsmeow/store/sqlstore"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
	waLog "go.mau.fi/whatsmeow/util/log"
	"google.golang.org/protobuf/proto"
)

// Whatsapp represents a WhatsApp client
type Whatsapp struct {
	client   *whatsmeow.Client
	ChatChan chan models.Chat
}

// NewWhatsapp creates a new Whatsapp client
func NewWhatsapp(storeDir string) (*Whatsapp, error) {
	container, err := sqlstore.New("sqlite3", fmt.Sprintf("file:%s/whatsapp.db?_foreign_keys=on", storeDir), waLog.Stdout("Database", "INFO", true))
	if err != nil {
		return nil, fmt.Errorf("failed to connect to WhatsApp database: %w", err)
	}
	deviceStore, err := container.GetFirstDevice()
	if err != nil {
		return nil, fmt.Errorf("failed to get device: %w", err)
	}

	client := whatsmeow.NewClient(deviceStore, waLog.Stdout("Client", "INFO", true))

	w := &Whatsapp{
		client: client,
	}

	w.ChatChan = make(chan models.Chat)

	// Set up event handler
	client.AddEventHandler(func(evt any) {
		switch v := evt.(type) {
		case *events.Message:
			msg, err := w.handleMessage(v)
			if err != nil {
				fmt.Println("Error handling message:", err)
			} else {
				w.ChatChan <- models.Chat{
					JID:             msg.ChatJID,
					Name:            msg.Sender,
					LastMessageTime: msg.Timestamp,
					Messages:        []models.Message{msg},
				}
			}
		case *events.HistorySync:
			chat, err := w.handleHistorySync(v)
			if err != nil {
				fmt.Println("Error handling history sync:", err)
			} else {
				w.ChatChan <- chat
			}
		case *events.Connected:
			fmt.Println("Connected to WhatsApp")
		case *events.LoggedOut:
			fmt.Println("Device logged out, please scan QR code to log in again")
		}
	})

	return w, nil
}

// Connect connects the client
func (w *Whatsapp) Connect() error {
	return w.client.Connect()
}

// IsLoggedIn returns true if the client is logged in
func (w *Whatsapp) IsLoggedIn() bool {
	return w.client.IsLoggedIn()
}

// IsConnected returns true if the client is connected
func (w *Whatsapp) IsConnected() bool {
	return w.client.IsConnected()
}

// Disconnect disconnects the client
func (w *Whatsapp) Disconnect() {
	w.client.Disconnect()
}

// GetQR returns the QR code for the client
func (w *Whatsapp) GetQR(ctx context.Context) (string, error) {
	if w.client.Store.ID != nil {
		err := w.client.Connect()
		if err != nil {
			return "", fmt.Errorf("failed to connect: %v", err)
		}
		return "", nil
	}

	qrChan, err := w.client.GetQRChannel(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to get QR channel: %v", err)
	}

	err = w.client.Connect()
	if err != nil {
		return "", fmt.Errorf("failed to connect to WhatsApp: %w", err)
	}

	if qrChan == nil {
		return "", fmt.Errorf("QR channel not available")
	}

	qr := ""
	connected := make(chan bool, 1)
	for evt := range qrChan {
		if evt.Event == "code" {
			qr = evt.Code
			qrterminal.GenerateHalfBlock(qr, qrterminal.L, os.Stdout)
		} else if evt.Event == "success" {
			connected <- true
			return "", nil
		} else {
			return "", fmt.Errorf("unexpected QR event: %v", evt.Event)
		}
	}

	select {
	case <-connected:
		fmt.Println("\nSuccessfully connected and authenticated!")
	case <-time.After(3 * time.Minute):
		return "", fmt.Errorf("Timeout waiting for QR code scan")
	}

	return qr, nil
}

// GetStatus returns the status of the client
func (w *Whatsapp) GetStatus() (models.Status, error) {
	return models.Status{
		Connected: w.client.IsConnected(),
		LoggedIn:  w.client.IsLoggedIn(),
		PushName:  w.client.Store.PushName,
	}, nil
}

// SendMessage sends a message to a recipient
func (w *Whatsapp) SendMessage(ctx context.Context, recipient string, message string) error {
	var recipientJID types.JID
	var err error

	if recipient[0] == '+' {
		recipient = recipient[1:]
	}

	if recipient[len(recipient)-5:] == "@s.whatsapp.net" {
		recipientJID, err = types.ParseJID(recipient)
	} else {
		recipientJID = types.NewJID(recipient, types.DefaultUserServer)
	}

	if err != nil {
		return fmt.Errorf("invalid recipient: %w", err)
	}

	msg := &waProto.Message{
		Conversation: proto.String(message),
	}

	_, err = w.client.SendMessage(ctx, recipientJID, msg)
	if err != nil {
		return fmt.Errorf("failed to send message: %w", err)
	}

	return nil
}

func (w *Whatsapp) handleMessage(msg *events.Message) (models.Message, error) {
	content := msg.Message.Conversation
	if content == nil {
		return models.Message{}, fmt.Errorf("message content is empty")
	}

	return models.Message{
		ID:        msg.Info.ID,
		ChatJID:   msg.Info.Chat.String(),
		Sender:    msg.Info.Sender.String(),
		Content:   *content,
		Timestamp: msg.Info.Timestamp,
		IsFromMe:  msg.Info.IsFromMe,
	}, nil
}

// HandleHistorySync processes message history sync events
func (w *Whatsapp) handleHistorySync(historySync *events.HistorySync) (models.Chat, error) {
	for _, conv := range historySync.Data.Conversations {
		var chat models.Chat

		chatJID := conv.GetId()
		if chatJID == "" {
			continue
		}

		var lastMessageTime time.Time

		for _, msg := range conv.GetMessages() {
			if msg.GetMessage() == nil {
				continue
			}

			content := msg.GetMessage().GetMessage().GetExtendedTextMessage().GetText()
			if content == "" {
				continue
			}

			timestamp := time.Unix(int64(msg.GetMessage().GetMessageTimestamp()), 0)
			if timestamp.After(lastMessageTime) {
				lastMessageTime = timestamp
			}

			message := models.Message{
				ID:        msg.GetMessage().GetKey().GetId(),
				ChatJID:   chatJID,
				Sender:    msg.GetMessage().GetKey().GetParticipant(),
				Content:   content,
				Timestamp: timestamp,
				IsFromMe:  msg.GetMessage().GetKey().GetFromMe(),
			}

			chat.Messages = append(chat.Messages, message)
		}

		chat.JID = chatJID
		chat.Name = conv.GetName()
		chat.LastMessageTime = lastMessageTime

		return chat, nil
	}

	return models.Chat{}, nil
}

// BuildHistorySync builds a history sync request
func (w *Whatsapp) BuildHistorySync(ctx context.Context) error {
	if w.client == nil {
		return errors.New("client is not initialized. Cannot request history sync")
	}

	if !w.client.IsConnected() {
		return errors.New("client is not connected. Please ensure you are connected to WhatsApp first")
	}

	if w.client.Store.ID == nil {
		return errors.New("client is not logged in. Please scan the QR code first")
	}

	historyMsg := w.client.BuildHistorySyncRequest(&types.MessageInfo{}, 100)
	if historyMsg == nil {
		return errors.New("failed to build history sync request")
	}

	_, err := w.client.SendMessage(ctx, types.JID{
		Server: types.DefaultUserServer,
		User:   "status",
	}, historyMsg)
	if err != nil {
		return fmt.Errorf("failed to send history sync request: %w", err)
	}

	return nil
}
