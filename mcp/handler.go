package mcp

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
)

func searchContactsHandler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	query, ok := request.Params.Arguments["query"].(string)
	if !ok {
		return nil, errors.New("query must be a string")
	}

	contacts, err := SearchContacts(query)
	if err != nil {
		return nil, err
	}

	contactsData, err := json.Marshal(contacts)
	if err != nil {
		return nil, err
	}

	return mcp.NewToolResultText(string(contactsData)), nil
}

func listMessagesHandler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	var dateRange []time.Time
	var senderPhoneNumber, chatJID, query string
	limit := 20
	page := 0
	includeContext := true
	contextBefore := 1
	contextAfter := 1

	if dr, ok := request.Params.Arguments["date_range"].([]interface{}); ok && len(dr) == 2 {
		if startStr, ok := dr[0].(string); ok {
			if start, err := time.Parse(time.RFC3339, startStr); err == nil {
				if endStr, ok := dr[1].(string); ok {
					if end, err := time.Parse(time.RFC3339, endStr); err == nil {
						dateRange = []time.Time{start, end}
					}
				}
			}
		}
	}

	if s, ok := request.Params.Arguments["sender_phone_number"].(string); ok {
		senderPhoneNumber = s
	}

	if c, ok := request.Params.Arguments["chat_jid"].(string); ok {
		chatJID = c
	}

	if q, ok := request.Params.Arguments["query"].(string); ok {
		query = q
	}

	if l, ok := request.Params.Arguments["limit"].(float64); ok {
		limit = int(l)
	}

	if p, ok := request.Params.Arguments["page"].(float64); ok {
		page = int(p)
	}

	if ic, ok := request.Params.Arguments["include_context"].(bool); ok {
		includeContext = ic
	}

	if cb, ok := request.Params.Arguments["context_before"].(float64); ok {
		contextBefore = int(cb)
	}

	if ca, ok := request.Params.Arguments["context_after"].(float64); ok {
		contextAfter = int(ca)
	}

	messages, err := ListMessages(dateRange, senderPhoneNumber, chatJID, query, limit, page, includeContext, contextBefore, contextAfter)
	if err != nil {
		return nil, err
	}

	messagesData, err := json.Marshal(messages)
	if err != nil {
		return nil, err
	}

	return mcp.NewToolResultText(string(messagesData)), nil
}

func listChatsHandler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	var query string
	limit := 20
	page := 0
	includeLastMessage := true
	sortBy := "last_active"

	if q, ok := request.Params.Arguments["query"].(string); ok {
		query = q
	}

	if l, ok := request.Params.Arguments["limit"].(float64); ok {
		limit = int(l)
	}

	if p, ok := request.Params.Arguments["page"].(float64); ok {
		page = int(p)
	}

	if ilm, ok := request.Params.Arguments["include_last_message"].(bool); ok {
		includeLastMessage = ilm
	}

	if sb, ok := request.Params.Arguments["sort_by"].(string); ok {
		sortBy = sb
	}

	chats, err := ListChats(query, limit, page, includeLastMessage, sortBy)
	if err != nil {
		return nil, err
	}

	chatsData, err := json.Marshal(chats)
	if err != nil {
		return nil, err
	}

	return mcp.NewToolResultText(string(chatsData)), nil
}

func getChatHandler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	chatJID, ok := request.Params.Arguments["chat_jid"].(string)
	if !ok {
		return nil, errors.New("chat_jid must be a string")
	}

	includeLastMessage := true
	if ilm, ok := request.Params.Arguments["include_last_message"].(bool); ok {
		includeLastMessage = ilm
	}

	chat, err := GetChat(chatJID, includeLastMessage)
	if err != nil {
		return nil, err
	}

	chatData, err := json.Marshal(chat)
	if err != nil {
		return nil, err
	}

	return mcp.NewToolResultText(string(chatData)), nil
}

func getDirectChatByContactHandler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	senderPhoneNumber, ok := request.Params.Arguments["sender_phone_number"].(string)
	if !ok {
		return nil, errors.New("sender_phone_number must be a string")
	}

	chat, err := GetDirectChatByContact(senderPhoneNumber)
	if err != nil {
		return nil, err
	}

	chatData, err := json.Marshal(chat)
	if err != nil {
		return nil, err
	}

	return mcp.NewToolResultText(string(chatData)), nil
}

func getContactChatsHandler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	jid, ok := request.Params.Arguments["jid"].(string)
	if !ok {
		return nil, errors.New("jid must be a string")
	}

	limit := 20
	if l, ok := request.Params.Arguments["limit"].(float64); ok {
		limit = int(l)
	}

	page := 0
	if p, ok := request.Params.Arguments["page"].(float64); ok {
		page = int(p)
	}

	chats, err := GetContactChats(jid, limit, page)
	if err != nil {
		return nil, err
	}

	chatsData, err := json.Marshal(chats)
	if err != nil {
		return nil, err
	}

	return mcp.NewToolResultText(string(chatsData)), nil
}

func getLastInteractionHandler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	jid, ok := request.Params.Arguments["jid"].(string)
	if !ok {
		return nil, errors.New("jid must be a string")
	}

	message, err := GetLastInteraction(jid)
	if err != nil {
		return nil, err
	}

	messageData, err := json.Marshal(message)
	if err != nil {
		return nil, err
	}

	return mcp.NewToolResultText(string(messageData)), nil
}

func getMessageContextHandler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	messageID, ok := request.Params.Arguments["message_id"].(string)
	if !ok {
		return nil, errors.New("message_id must be a string")
	}

	before := 5
	if b, ok := request.Params.Arguments["before"].(float64); ok {
		before = int(b)
	}

	after := 5
	if a, ok := request.Params.Arguments["after"].(float64); ok {
		after = int(a)
	}

	context, err := GetMessageContext(messageID, before, after)
	if err != nil {
		return nil, err
	}

	contextData, err := json.Marshal(context)
	if err != nil {
		return nil, err
	}

	return mcp.NewToolResultText(string(contextData)), nil
}

func sendMessageHandler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	recipient, ok := request.Params.Arguments["recipient"].(string)
	if !ok {
		return nil, errors.New("recipient must be a string")
	}

	message, ok := request.Params.Arguments["message"].(string)
	if !ok {
		return nil, errors.New("message must be a string")
	}

	success, statusMessage := SendMessage(recipient, message)

	result := map[string]interface{}{
		"success": success,
		"message": statusMessage,
	}

	resultData, err := json.Marshal(result)
	if err != nil {
		return nil, err
	}

	return mcp.NewToolResultText(string(resultData)), nil
}
