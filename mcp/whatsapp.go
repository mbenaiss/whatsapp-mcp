package mcp

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

// Constants for paths and API URL
var (
	MessagesDBPath     string
	WhatsappAPIBaseURL = "http://localhost:8080/api"
)

func init() {
	execPath, err := os.Executable()
	if err != nil {
		log.Fatalf("Unable to determine execution path: %v", err)
	}
	execDir := filepath.Dir(execPath)
	MessagesDBPath = filepath.Join(execDir, "..", "whatsapp-bridge", "store", "messages.db")
}

// Message represents a WhatsApp message
type Message struct {
	Timestamp time.Time
	Sender    string
	Content   string
	IsFromMe  bool
	ChatJID   string
	ID        string
	ChatName  string
}

// Chat represents a WhatsApp conversation
type Chat struct {
	JID             string
	Name            string
	LastMessageTime time.Time
	LastMessage     string
	LastSender      string
	LastIsFromMe    bool
}

// IsGroup determines if the chat is a group based on JID pattern
func (c *Chat) IsGroup() bool {
	return strings.HasSuffix(c.JID, "@g.us")
}

// Contact represents a WhatsApp contact
type Contact struct {
	PhoneNumber string
	Name        string
	JID         string
}

// MessageContext represents a message with its context (messages before and after)
type MessageContext struct {
	Message Message
	Before  []Message
	After   []Message
}

// PrintMessage displays a message with consistent formatting
func PrintMessage(message Message, showChatInfo bool) {
	direction := "→"
	if !message.IsFromMe {
		direction = "←"
	}

	if showChatInfo && message.ChatName != "" {
		fmt.Printf("[%s] %s Chat: %s (%s)\n", message.Timestamp.Format("2006-01-02 15:04:05"), direction, message.ChatName, message.ChatJID)
	} else {
		fmt.Printf("[%s] %s\n", message.Timestamp.Format("2006-01-02 15:04:05"), direction)
	}

	sender := "Me"
	if !message.IsFromMe {
		sender = message.Sender
	}
	fmt.Printf("From: %s\n", sender)
	fmt.Printf("Message: %s\n", message.Content)
	fmt.Println(strings.Repeat("-", 100))
}

// PrintMessagesList displays a list of messages with a title and consistent formatting
func PrintMessagesList(messages []Message, title string, showChatInfo bool) {
	if len(messages) == 0 {
		fmt.Println("No messages to display.")
		return
	}

	if title != "" {
		fmt.Printf("\n%s\n", title)
		fmt.Println(strings.Repeat("-", 100))
	}

	for _, message := range messages {
		PrintMessage(message, showChatInfo)
	}
}

// PrintChat displays a chat with consistent formatting
func PrintChat(chat Chat) {
	fmt.Printf("Chat: %s (%s)\n", chat.Name, chat.JID)
	if !chat.LastMessageTime.IsZero() {
		fmt.Printf("Last active: %s\n", chat.LastMessageTime.Format("2006-01-02 15:04:05"))
		direction := "→"
		if !chat.LastIsFromMe {
			direction = "←"
		}
		sender := "Me"
		if !chat.LastIsFromMe {
			sender = chat.LastSender
		}
		fmt.Printf("Last message: %s %s: %s\n", direction, sender, chat.LastMessage)
	}
	fmt.Println(strings.Repeat("-", 100))
}

// PrintChatsList displays a list of chats with a title and consistent formatting
func PrintChatsList(chats []Chat, title string) {
	if len(chats) == 0 {
		fmt.Println("No chats to display.")
		return
	}

	if title != "" {
		fmt.Printf("\n%s\n", title)
		fmt.Println(strings.Repeat("-", 100))
	}

	for _, chat := range chats {
		PrintChat(chat)
	}
}

// PrintPaginatedMessages displays a paginated list of messages with navigation indices
func PrintPaginatedMessages(messages []Message, page int, totalPages int, chatName string) {
	fmt.Printf("\nMessages for chat: %s\n", chatName)
	fmt.Printf("Page %d of %d\n", page, totalPages)
	fmt.Println(strings.Repeat("-", 100))

	PrintMessagesList(messages, "", false)

	if page > 1 {
		fmt.Printf("Use page=%d to see more recent messages\n", page-1)
	}
	if page < totalPages {
		fmt.Printf("Use page=%d to see older messages\n", page+1)
	}
}

// GetDB creates a connection to the SQLite database
func GetDB() (*sql.DB, error) {
	db, err := sql.Open("sqlite3", MessagesDBPath)
	if err != nil {
		return nil, fmt.Errorf("unable to open database: %v", err)
	}
	return db, nil
}

// PrintRecentMessages retrieves and displays recent messages
func PrintRecentMessages(limit int) ([]Message, error) {
	if limit <= 0 {
		limit = 10
	}

	db, err := GetDB()
	if err != nil {
		return nil, err
	}
	defer db.Close()

	query := `
	SELECT 
		m.timestamp,
		m.sender,
		c.name,
		m.content,
		m.is_from_me,
		c.jid,
		m.id
	FROM messages m
	JOIN chats c ON m.chat_jid = c.jid
	ORDER BY m.timestamp DESC
	LIMIT ?
	`

	rows, err := db.Query(query, limit)
	if err != nil {
		return nil, fmt.Errorf("error executing query: %v", err)
	}
	defer rows.Close()

	var messages []Message

	for rows.Next() {
		var msg Message
		var timestampStr string
		var chatName sql.NullString

		err := rows.Scan(
			&timestampStr,
			&msg.Sender,
			&chatName,
			&msg.Content,
			&msg.IsFromMe,
			&msg.ChatJID,
			&msg.ID,
		)
		if err != nil {
			return nil, fmt.Errorf("error reading data: %v", err)
		}

		msg.Timestamp, err = time.Parse(time.RFC3339, timestampStr)
		if err != nil {
			return nil, fmt.Errorf("error converting timestamp: %v", err)
		}

		if chatName.Valid {
			msg.ChatName = chatName.String
		} else {
			msg.ChatName = "Unknown Chat"
		}

		messages = append(messages, msg)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error traversing results: %v", err)
	}

	if len(messages) == 0 {
		fmt.Println("No messages found in the database.")
		return []Message{}, nil
	}

	PrintMessagesList(messages, fmt.Sprintf("Last %d messages:", limit), true)
	return messages, nil
}

// ListMessages retrieves messages matching specified criteria
func ListMessages(dateRange []time.Time, senderPhoneNumber, chatJID, query string, limit, page int, includeContext bool, contextBefore, contextAfter int) ([]Message, error) {
	if limit <= 0 {
		limit = 20
	}
	if page < 0 {
		page = 0
	}

	db, err := GetDB()
	if err != nil {
		return nil, err
	}
	defer db.Close()

	queryParts := []string{"SELECT messages.timestamp, messages.sender, chats.name, messages.content, messages.is_from_me, chats.jid, messages.id FROM messages"}
	queryParts = append(queryParts, "JOIN chats ON messages.chat_jid = chats.jid")
	whereClauses := []string{}
	params := []interface{}{}

	if len(dateRange) == 2 {
		whereClauses = append(whereClauses, "messages.timestamp BETWEEN ? AND ?")
		params = append(params, dateRange[0].Format(time.RFC3339), dateRange[1].Format(time.RFC3339))
	}

	if senderPhoneNumber != "" {
		whereClauses = append(whereClauses, "messages.sender = ?")
		params = append(params, senderPhoneNumber)
	}

	if chatJID != "" {
		whereClauses = append(whereClauses, "messages.chat_jid = ?")
		params = append(params, chatJID)
	}

	if query != "" {
		whereClauses = append(whereClauses, "LOWER(messages.content) LIKE LOWER(?)")
		params = append(params, "%"+query+"%")
	}

	if len(whereClauses) > 0 {
		queryParts = append(queryParts, "WHERE "+strings.Join(whereClauses, " AND "))
	}

	offset := page * limit
	queryParts = append(queryParts, "ORDER BY messages.timestamp DESC")
	queryParts = append(queryParts, "LIMIT ? OFFSET ?")
	params = append(params, limit, offset)

	rows, err := db.Query(strings.Join(queryParts, " "), params...)
	if err != nil {
		return nil, fmt.Errorf("error executing query: %v", err)
	}
	defer rows.Close()

	var messages []Message

	for rows.Next() {
		var msg Message
		var timestampStr string
		var chatName sql.NullString

		err := rows.Scan(
			&timestampStr,
			&msg.Sender,
			&chatName,
			&msg.Content,
			&msg.IsFromMe,
			&msg.ChatJID,
			&msg.ID,
		)
		if err != nil {
			return nil, fmt.Errorf("error reading data: %v", err)
		}

		msg.Timestamp, err = time.Parse(time.RFC3339, timestampStr)
		if err != nil {
			return nil, fmt.Errorf("error converting timestamp: %v", err)
		}

		if chatName.Valid {
			msg.ChatName = chatName.String
		} else {
			msg.ChatName = "Unknown Chat"
		}

		messages = append(messages, msg)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error traversing results: %v", err)
	}

	if includeContext && len(messages) > 0 {
		var messagesWithContext []Message
		for _, msg := range messages {
			context, err := GetMessageContext(msg.ID, contextBefore, contextAfter)
			if err != nil {
				continue
			}
			messagesWithContext = append(messagesWithContext, context.Before...)
			messagesWithContext = append(messagesWithContext, context.Message)
			messagesWithContext = append(messagesWithContext, context.After...)
		}
		return messagesWithContext, nil
	}

	return messages, nil
}

// GetMessageContext retrieves the context around a specific message
func GetMessageContext(messageID string, before, after int) (*MessageContext, error) {
	db, err := GetDB()
	if err != nil {
		return nil, err
	}
	defer db.Close()

	query := `
		SELECT messages.timestamp, messages.sender, chats.name, messages.content, messages.is_from_me, chats.jid, messages.id, messages.chat_jid
		FROM messages
		JOIN chats ON messages.chat_jid = chats.jid
		WHERE messages.id = ?
	`
	row := db.QueryRow(query, messageID)

	var targetMsg Message
	var timestampStr string
	var chatName sql.NullString
	var chatJID string
	err = row.Scan(
		&timestampStr,
		&targetMsg.Sender,
		&chatName,
		&targetMsg.Content,
		&targetMsg.IsFromMe,
		&targetMsg.ChatJID,
		&targetMsg.ID,
		&chatJID,
	)
	if err != nil {
		return nil, fmt.Errorf("message with ID %s not found: %v", messageID, err)
	}

	targetMsg.Timestamp, err = time.Parse(time.RFC3339, timestampStr)
	if err != nil {
		return nil, fmt.Errorf("error converting timestamp: %v", err)
	}

	if chatName.Valid {
		targetMsg.ChatName = chatName.String
	} else {
		targetMsg.ChatName = "Unknown Chat"
	}

	queryBefore := `
		SELECT messages.timestamp, messages.sender, chats.name, messages.content, messages.is_from_me, chats.jid, messages.id
		FROM messages
		JOIN chats ON messages.chat_jid = chats.jid
		WHERE messages.chat_jid = ? AND messages.timestamp < ?
		ORDER BY messages.timestamp DESC
		LIMIT ?
	`
	rowsBefore, err := db.Query(queryBefore, chatJID, timestampStr, before)
	if err != nil {
		return nil, fmt.Errorf("error retrieving previous messages: %v", err)
	}
	defer rowsBefore.Close()

	var beforeMessages []Message
	for rowsBefore.Next() {
		var msg Message
		var msgTimestampStr string
		var msgChatName sql.NullString

		err := rowsBefore.Scan(
			&msgTimestampStr,
			&msg.Sender,
			&msgChatName,
			&msg.Content,
			&msg.IsFromMe,
			&msg.ChatJID,
			&msg.ID,
		)
		if err != nil {
			return nil, fmt.Errorf("error reading data: %v", err)
		}

		msg.Timestamp, err = time.Parse(time.RFC3339, msgTimestampStr)
		if err != nil {
			return nil, fmt.Errorf("error converting timestamp: %v", err)
		}

		if msgChatName.Valid {
			msg.ChatName = msgChatName.String
		} else {
			msg.ChatName = "Unknown Chat"
		}

		beforeMessages = append(beforeMessages, msg)
	}

	queryAfter := `
		SELECT messages.timestamp, messages.sender, chats.name, messages.content, messages.is_from_me, chats.jid, messages.id
		FROM messages
		JOIN chats ON messages.chat_jid = chats.jid
		WHERE messages.chat_jid = ? AND messages.timestamp > ?
		ORDER BY messages.timestamp ASC
		LIMIT ?
	`
	rowsAfter, err := db.Query(queryAfter, chatJID, timestampStr, after)
	if err != nil {
		return nil, fmt.Errorf("error retrieving following messages: %v", err)
	}
	defer rowsAfter.Close()

	var afterMessages []Message
	for rowsAfter.Next() {
		var msg Message
		var msgTimestampStr string
		var msgChatName sql.NullString

		err := rowsAfter.Scan(
			&msgTimestampStr,
			&msg.Sender,
			&msgChatName,
			&msg.Content,
			&msg.IsFromMe,
			&msg.ChatJID,
			&msg.ID,
		)
		if err != nil {
			return nil, fmt.Errorf("error reading data: %v", err)
		}

		msg.Timestamp, err = time.Parse(time.RFC3339, msgTimestampStr)
		if err != nil {
			return nil, fmt.Errorf("error converting timestamp: %v", err)
		}

		if msgChatName.Valid {
			msg.ChatName = msgChatName.String
		} else {
			msg.ChatName = "Unknown Chat"
		}

		afterMessages = append(afterMessages, msg)
	}

	return &MessageContext{
		Message: targetMsg,
		Before:  beforeMessages,
		After:   afterMessages,
	}, nil
}

// ListChats retrieves chats matching specified criteria
func ListChats(query string, limit, page int, includeLastMessage bool, sortBy string) ([]Chat, error) {
	if limit <= 0 {
		limit = 20
	}
	if page < 0 {
		page = 0
	}

	db, err := GetDB()
	if err != nil {
		return nil, err
	}
	defer db.Close()

	queryParts := []string{`
		SELECT 
			chats.jid,
			chats.name,
			chats.last_message_time,
			messages.content as last_message,
			messages.sender as last_sender,
			messages.is_from_me as last_is_from_me
		FROM chats
	`}

	if includeLastMessage {
		queryParts = append(queryParts, `
			LEFT JOIN messages ON chats.jid = messages.chat_jid 
			AND chats.last_message_time = messages.timestamp
		`)
	}

	whereClauses := []string{}
	params := []interface{}{}

	if query != "" {
		whereClauses = append(whereClauses, "(LOWER(chats.name) LIKE LOWER(?) OR chats.jid LIKE ?)")
		params = append(params, "%"+query+"%", "%"+query+"%")
	}

	if len(whereClauses) > 0 {
		queryParts = append(queryParts, "WHERE "+strings.Join(whereClauses, " AND "))
	}

	orderBy := "chats.last_message_time DESC"
	if sortBy == "name" {
		orderBy = "chats.name"
	}
	queryParts = append(queryParts, "ORDER BY "+orderBy)

	offset := page * limit
	queryParts = append(queryParts, "LIMIT ? OFFSET ?")
	params = append(params, limit, offset)

	rows, err := db.Query(strings.Join(queryParts, " "), params...)
	if err != nil {
		return nil, fmt.Errorf("error executing query: %v", err)
	}
	defer rows.Close()

	var chats []Chat

	for rows.Next() {
		var chat Chat
		var timestampStr sql.NullString
		var name, lastMessage, lastSender sql.NullString
		var lastIsFromMe sql.NullBool

		err := rows.Scan(
			&chat.JID,
			&name,
			&timestampStr,
			&lastMessage,
			&lastSender,
			&lastIsFromMe,
		)
		if err != nil {
			return nil, fmt.Errorf("error reading data: %v", err)
		}

		if name.Valid {
			chat.Name = name.String
		}

		if timestampStr.Valid {
			chat.LastMessageTime, err = time.Parse(time.RFC3339, timestampStr.String)
			if err != nil {
				return nil, fmt.Errorf("error converting timestamp: %v", err)
			}
		}

		if lastMessage.Valid {
			chat.LastMessage = lastMessage.String
		}

		if lastSender.Valid {
			chat.LastSender = lastSender.String
		}

		if lastIsFromMe.Valid {
			chat.LastIsFromMe = lastIsFromMe.Bool
		}

		chats = append(chats, chat)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error traversing results: %v", err)
	}

	return chats, nil
}

// SearchContacts searches for contacts by name or phone number
func SearchContacts(query string) ([]Contact, error) {
	db, err := GetDB()
	if err != nil {
		return nil, err
	}
	defer db.Close()

	searchPattern := "%" + query + "%"

	queryStr := `
		SELECT DISTINCT 
			jid,
			name
		FROM chats
		WHERE 
			(LOWER(name) LIKE LOWER(?) OR LOWER(jid) LIKE LOWER(?))
			AND jid NOT LIKE '%@g.us'
		ORDER BY name, jid
		LIMIT 50
	`

	rows, err := db.Query(queryStr, searchPattern, searchPattern)
	if err != nil {
		return nil, fmt.Errorf("error executing query: %v", err)
	}
	defer rows.Close()

	var contacts []Contact

	for rows.Next() {
		var contact Contact
		var name sql.NullString

		err := rows.Scan(
			&contact.JID,
			&name,
		)
		if err != nil {
			return nil, fmt.Errorf("error reading data: %v", err)
		}

		if name.Valid {
			contact.Name = name.String
		}

		parts := strings.Split(contact.JID, "@")
		if len(parts) > 0 {
			contact.PhoneNumber = parts[0]
		}

		contacts = append(contacts, contact)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error traversing results: %v", err)
	}

	return contacts, nil
}

// SendMessage sends a WhatsApp message to the specified recipient
func SendMessage(recipient, message string) (bool, string) {
	if recipient == "" {
		return false, "Recipient must be provided"
	}

	url := fmt.Sprintf("%s/send", WhatsappAPIBaseURL)
	payload := map[string]string{
		"recipient": recipient,
		"message":   message,
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return false, fmt.Sprintf("JSON serialization error: %v", err)
	}

	resp, err := http.Post(url, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return false, fmt.Sprintf("Request error: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		var result map[string]interface{}
		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			return false, fmt.Sprintf("Response decoding error: %v", err)
		}

		success, ok := result["success"].(bool)
		if !ok {
			return false, "Unexpected response format"
		}

		message, _ := result["message"].(string)
		return success, message
	}

	body, _ := io.ReadAll(resp.Body)
	return false, fmt.Sprintf("Error: HTTP %d - %s", resp.StatusCode, string(body))
}

// GetChat retrieves metadata for a WhatsApp chat by JID
func GetChat(chatJID string, includeLastMessage bool) (*Chat, error) {
	db, err := GetDB()
	if err != nil {
		return nil, err
	}
	defer db.Close()

	queryStr := `
		SELECT 
			chats.jid,
			chats.name,
			chats.last_message_time
	`

	if includeLastMessage {
		queryStr += `,
			messages.content as last_message,
			messages.sender as last_sender,
			messages.is_from_me as last_is_from_me
		FROM chats
		LEFT JOIN messages ON chats.jid = messages.chat_jid 
		AND chats.last_message_time = messages.timestamp
		`
	} else {
		queryStr += `
		FROM chats
		`
	}

	queryStr += `
		WHERE chats.jid = ?
	`

	row := db.QueryRow(queryStr, chatJID)

	var chat Chat
	var timestampStr sql.NullString
	var name, lastMessage, lastSender sql.NullString
	var lastIsFromMe sql.NullBool

	if includeLastMessage {
		err = row.Scan(
			&chat.JID,
			&name,
			&timestampStr,
			&lastMessage,
			&lastSender,
			&lastIsFromMe,
		)
	} else {
		err = row.Scan(
			&chat.JID,
			&name,
			&timestampStr,
		)
	}

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("chat with JID %s not found", chatJID)
		}
		return nil, fmt.Errorf("error reading data: %v", err)
	}

	if name.Valid {
		chat.Name = name.String
	}

	if timestampStr.Valid {
		chat.LastMessageTime, err = time.Parse(time.RFC3339, timestampStr.String)
		if err != nil {
			return nil, fmt.Errorf("error converting timestamp: %v", err)
		}
	}

	if includeLastMessage {
		if lastMessage.Valid {
			chat.LastMessage = lastMessage.String
		}

		if lastSender.Valid {
			chat.LastSender = lastSender.String
		}

		if lastIsFromMe.Valid {
			chat.LastIsFromMe = lastIsFromMe.Bool
		}
	}

	return &chat, nil
}

// GetDirectChatByContact retrieves metadata for a direct chat by phone number
func GetDirectChatByContact(phoneNumber string) (*Chat, error) {
	jid := phoneNumber
	if !strings.Contains(jid, "@") {
		jid = phoneNumber + "@s.whatsapp.net"
	}

	return GetChat(jid, true)
}

// GetContactChats retrieves all chats involving the contact
func GetContactChats(jid string, limit, page int) ([]Chat, error) {
	if limit <= 0 {
		limit = 20
	}
	if page < 0 {
		page = 0
	}

	db, err := GetDB()
	if err != nil {
		return nil, err
	}
	defer db.Close()

	phoneNumber := jid
	if strings.Contains(jid, "@") {
		parts := strings.Split(jid, "@")
		phoneNumber = parts[0]
	}

	queryStr := `
		SELECT DISTINCT
			chats.jid,
			chats.name,
			chats.last_message_time,
			messages.content as last_message,
			messages.sender as last_sender,
			messages.is_from_me as last_is_from_me
		FROM chats
		LEFT JOIN messages ON chats.jid = messages.chat_jid 
		AND chats.last_message_time = messages.timestamp
		JOIN messages as m ON chats.jid = m.chat_jid
		WHERE 
			(chats.jid = ? OR m.sender = ? OR m.sender LIKE ?)
		GROUP BY chats.jid
		ORDER BY chats.last_message_time DESC
		LIMIT ? OFFSET ?
	`

	offset := page * limit
	rows, err := db.Query(queryStr, jid, phoneNumber, "%"+phoneNumber+"%", limit, offset)
	if err != nil {
		return nil, fmt.Errorf("error executing query: %v", err)
	}
	defer rows.Close()

	var chats []Chat

	for rows.Next() {
		var chat Chat
		var timestampStr sql.NullString
		var name, lastMessage, lastSender sql.NullString
		var lastIsFromMe sql.NullBool

		err := rows.Scan(
			&chat.JID,
			&name,
			&timestampStr,
			&lastMessage,
			&lastSender,
			&lastIsFromMe,
		)
		if err != nil {
			return nil, fmt.Errorf("error reading data: %v", err)
		}

		if name.Valid {
			chat.Name = name.String
		}

		if timestampStr.Valid {
			chat.LastMessageTime, err = time.Parse(time.RFC3339, timestampStr.String)
			if err != nil {
				return nil, fmt.Errorf("error converting timestamp: %v", err)
			}
		}

		if lastMessage.Valid {
			chat.LastMessage = lastMessage.String
		}

		if lastSender.Valid {
			chat.LastSender = lastSender.String
		}

		if lastIsFromMe.Valid {
			chat.LastIsFromMe = lastIsFromMe.Bool
		}

		chats = append(chats, chat)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error traversing results: %v", err)
	}

	return chats, nil
}

// GetLastInteraction retrieves the most recent message involving the contact
func GetLastInteraction(jid string) (*Message, error) {
	db, err := GetDB()
	if err != nil {
		return nil, err
	}
	defer db.Close()

	phoneNumber := jid
	if strings.Contains(jid, "@") {
		parts := strings.Split(jid, "@")
		phoneNumber = parts[0]
	}

	queryStr := `
		SELECT 
			messages.timestamp,
			messages.sender,
			chats.name,
			messages.content,
			messages.is_from_me,
			chats.jid,
			messages.id
		FROM messages
		JOIN chats ON messages.chat_jid = chats.jid
		WHERE 
			(messages.chat_jid = ? OR messages.sender = ? OR messages.sender LIKE ?)
		ORDER BY messages.timestamp DESC
		LIMIT 1
	`

	row := db.QueryRow(queryStr, jid, phoneNumber, "%"+phoneNumber+"%")

	var msg Message
	var timestampStr string
	var chatName sql.NullString

	err = row.Scan(
		&timestampStr,
		&msg.Sender,
		&chatName,
		&msg.Content,
		&msg.IsFromMe,
		&msg.ChatJID,
		&msg.ID,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("no message found for contact with JID %s", jid)
		}
		return nil, fmt.Errorf("error reading data: %v", err)
	}

	msg.Timestamp, err = time.Parse(time.RFC3339, timestampStr)
	if err != nil {
		return nil, fmt.Errorf("error converting timestamp: %v", err)
	}

	if chatName.Valid {
		msg.ChatName = chatName.String
	} else {
		msg.ChatName = "Unknown Chat"
	}

	return &msg, nil
}
