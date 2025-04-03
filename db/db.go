package db

import (
	"context"
	"database/sql"
	"fmt"
	"os"

	_ "github.com/mattn/go-sqlite3"
	"github.com/mbenaiss/whatsapp-mcp/models"
)

// DB handles storage in SQLite
type DB interface {
	StoreChat(ctx context.Context, chat models.Chat) error
	StoreMessage(ctx context.Context, msg models.Message) error
	GetMessages(ctx context.Context, chatJID string, limit int) ([]models.Message, error)
	GetChats(ctx context.Context) ([]models.Chat, error)
	GetChat(ctx context.Context, jid string) (*models.Chat, error)
	Close() error
}

type db struct {
	db *sql.DB
}

// NewDB creates a new database
func NewDB(ctx context.Context, dbPath string) (DB, error) {
	if err := os.MkdirAll(dbPath, 0755); err != nil {
		return nil, fmt.Errorf("failed to create store directory: %v", err)
	}

	conn, err := sql.Open("sqlite3", fmt.Sprintf("file:%s/messages.db?_foreign_keys=on", dbPath))
	if err != nil {
		return nil, fmt.Errorf("failed to open message database: %v", err)
	}

	db := &db{conn}
	if err := db.initDB(ctx); err != nil {
		return nil, fmt.Errorf("failed to initialize database: %v", err)
	}

	return db, nil
}

func (s *db) initDB(ctx context.Context) error {
	// Set pragmas for better performance
	_, err := s.db.ExecContext(ctx, `PRAGMA foreign_keys = ON;`)
	if err != nil {
		return fmt.Errorf("failed to set foreign keys pragma: %v", err)
	}

	_, err = s.db.ExecContext(ctx, `PRAGMA journal_mode = WAL;`)
	if err != nil {
		return fmt.Errorf("failed to set journal mode: %v", err)
	}

	// Create tables in separate statements for better error handling
	_, err = s.db.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS chats (
			jid TEXT PRIMARY KEY,
			name TEXT,
			last_message_time TIMESTAMP
		);
	`)
	if err != nil {
		return fmt.Errorf("failed to create chats table: %v", err)
	}

	_, err = s.db.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS messages (
			id TEXT,
			chat_jid TEXT,
			sender TEXT,
			content TEXT,
			timestamp TIMESTAMP,
			is_from_me BOOLEAN,
			PRIMARY KEY (id, chat_jid),
			FOREIGN KEY (chat_jid) REFERENCES chats(jid)
		);
	`)
	if err != nil {
		return fmt.Errorf("failed to create messages table: %v", err)
	}

	// Create indexes separately and concurrently for better performance
	_, err = s.db.ExecContext(ctx, `CREATE INDEX IF NOT EXISTS idx_messages_timestamp ON messages(timestamp);`)
	if err != nil {
		return fmt.Errorf("failed to create timestamp index: %v", err)
	}

	_, err = s.db.ExecContext(ctx, `CREATE INDEX IF NOT EXISTS idx_messages_chat_timestamp ON messages(chat_jid, timestamp);`)
	if err != nil {
		return fmt.Errorf("failed to create chat_timestamp index: %v", err)
	}

	return nil
}

func (s *db) Close() error {
	return s.db.Close()
}

// StoreChat stores a chat in the database
func (s *db) StoreChat(ctx context.Context, chat models.Chat) error {
	_, err := s.db.ExecContext(ctx,
		"INSERT OR REPLACE INTO chats (jid, name, last_message_time) VALUES (?, ?, ?)",
		chat.JID, chat.Name, chat.LastMessageTime,
	)
	return err
}

// StoreMessage stores a message in the database
func (s *db) StoreMessage(ctx context.Context, msg models.Message) error {
	if msg.Content == "" {
		return nil
	}

	_, err := s.db.ExecContext(ctx,
		`INSERT OR REPLACE INTO messages
		(id, chat_jid, sender, content, timestamp, is_from_me)
		VALUES (?, ?, ?, ?, ?, ?)`,
		msg.ID, msg.ChatJID, msg.Sender, msg.Content, msg.Timestamp, msg.IsFromMe,
	)
	return err
}

// GetMessages retrieves messages from a chat
func (s *db) GetMessages(ctx context.Context, chatJID string, limit int) ([]models.Message, error) {
	rows, err := s.db.QueryContext(ctx,
		"SELECT id, chat_jid, sender, content, timestamp, is_from_me FROM messages WHERE chat_jid = ? ORDER BY timestamp DESC LIMIT ?",
		chatJID, limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var messages []models.Message
	for rows.Next() {
		msg := models.Message{}
		err := rows.Scan(&msg.ID, &msg.ChatJID, &msg.Sender, &msg.Content, &msg.Timestamp, &msg.IsFromMe)
		if err != nil {
			return nil, err
		}
		messages = append(messages, msg)
	}

	return messages, nil
}

// GetChats retrieves all chats
func (s *db) GetChats(ctx context.Context) ([]models.Chat, error) {
	rows, err := s.db.QueryContext(ctx, "SELECT jid, name, last_message_time FROM chats ORDER BY last_message_time DESC")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var chats []models.Chat
	for rows.Next() {
		chat := models.Chat{}
		err := rows.Scan(&chat.JID, &chat.Name, &chat.LastMessageTime)
		if err != nil {
			return nil, err
		}
		chats = append(chats, chat)
	}

	return chats, nil
}

// GetChat retrieves a specific chat
func (s *db) GetChat(ctx context.Context, jid string) (*models.Chat, error) {
	chat := &models.Chat{}
	err := s.db.QueryRowContext(ctx,
		"SELECT jid, name, last_message_time FROM chats WHERE jid = ?",
		jid,
	).Scan(&chat.JID, &chat.Name, &chat.LastMessageTime)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return chat, nil
}
