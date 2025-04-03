package models

import "time"

// Message represents a chat message
type Message struct {
	ID        string    `json:"id"`
	ChatJID   string    `json:"chat_jid"`
	Sender    string    `json:"sender"`
	Content   string    `json:"content"`
	Timestamp time.Time `json:"timestamp"`
	IsFromMe  bool      `json:"is_from_me"`
	ChatName  string    `json:"chat_name"`
}

// Chat represents a WhatsApp chat
type Chat struct {
	JID             string    `json:"jid"`
	Name            string    `json:"name"`
	LastMessageTime time.Time `json:"last_message_time"`
	LastMessage     string    `json:"last_message"`
	LastSender      string    `json:"last_sender"`
	LastIsFromMe    bool      `json:"last_is_from_me"`
	Messages        []Message `json:"messages"`
}

// Contact represents a WhatsApp contact
type Contact struct {
	PhoneNumber string `json:"phone_number"`
	Name        string `json:"name"`
	JID         string `json:"jid"`
}

// Status represents the status of the WhatsApp client
type Status struct {
	Connected bool   `json:"connected"`
	LoggedIn  bool   `json:"logged_in"`
	PushName  string `json:"push_name"`
}
