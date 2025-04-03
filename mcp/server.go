package mcp

import (
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// NewMCPServer creates a new MCP server
func NewMCPServer(name string, version string) *server.MCPServer {
	s := server.NewMCPServer(
		name,
		version,
	)

	searchContactsTool := mcp.NewTool("search_contacts",
		mcp.WithDescription("Search WhatsApp contacts by name or phone number"),
		mcp.WithString("query",
			mcp.Required(),
			mcp.Description("Search term for names or phone numbers"),
		),
	)

	listMessagesTool := mcp.NewTool("list_messages",
		mcp.WithDescription("Retrieve WhatsApp messages matching specified criteria with optional context"),
		mcp.WithArray("date_range",
			mcp.Description("Optional tuple of (start_date, end_date) to filter messages by date"),
		),
		mcp.WithString("sender_phone_number",
			mcp.Description("Optional phone number to filter messages by sender"),
		),
		mcp.WithString("chat_jid",
			mcp.Description("Optional chat JID to filter messages by chat"),
		),
		mcp.WithString("query",
			mcp.Description("Optional search term to filter messages by content"),
		),
		mcp.WithNumber("limit",
			mcp.Description("Maximum number of messages to return (default 20)"),
		),
		mcp.WithNumber("page",
			mcp.Description("Page number for pagination (default 0)"),
		),
		mcp.WithBoolean("include_context",
			mcp.Description("Whether to include messages before and after matches (default true)"),
		),
		mcp.WithNumber("context_before",
			mcp.Description("Number of messages to include before each match (default 1)"),
		),
		mcp.WithNumber("context_after",
			mcp.Description("Number of messages to include after each match (default 1)"),
		),
	)

	listChatsTool := mcp.NewTool("list_chats",
		mcp.WithDescription("Retrieve WhatsApp chats matching specified criteria"),
		mcp.WithString("query",
			mcp.Description("Optional search term to filter chats by name or JID"),
		),
		mcp.WithNumber("limit",
			mcp.Description("Maximum number of chats to return (default 20)"),
		),
		mcp.WithNumber("page",
			mcp.Description("Page number for pagination (default 0)"),
		),
		mcp.WithBoolean("include_last_message",
			mcp.Description("Whether to include the last message in each chat (default true)"),
		),
		mcp.WithString("sort_by",
			mcp.Description("Field to sort results by, either 'last_active' or 'name' (default 'last_active')"),
		),
	)

	getChatTool := mcp.NewTool("get_chat",
		mcp.WithDescription("Retrieve metadata of a WhatsApp chat by JID"),
		mcp.WithString("chat_jid",
			mcp.Required(),
			mcp.Description("JID of the chat to retrieve"),
		),
		mcp.WithBoolean("include_last_message",
			mcp.Description("Whether to include the last message (default true)"),
		),
	)

	getDirectChatByContactTool := mcp.NewTool("get_direct_chat_by_contact",
		mcp.WithDescription("Retrieve metadata of a WhatsApp chat by sender's phone number"),
		mcp.WithString("sender_phone_number",
			mcp.Required(),
			mcp.Description("Phone number to search for"),
		),
	)

	getContactChatsTool := mcp.NewTool("get_contact_chats",
		mcp.WithDescription("Retrieve all WhatsApp chats involving the contact"),
		mcp.WithString("jid",
			mcp.Required(),
			mcp.Description("JID of the contact to search for"),
		),
		mcp.WithNumber("limit",
			mcp.Description("Maximum number of chats to return (default 20)"),
		),
		mcp.WithNumber("page",
			mcp.Description("Page number for pagination (default 0)"),
		),
	)

	getLastInteractionTool := mcp.NewTool("get_last_interaction",
		mcp.WithDescription("Retrieve the most recent WhatsApp message involving the contact"),
		mcp.WithString("jid",
			mcp.Required(),
			mcp.Description("JID of the contact to search for"),
		),
	)

	getMessageContextTool := mcp.NewTool("get_message_context",
		mcp.WithDescription("Retrieve context around a specific WhatsApp message"),
		mcp.WithString("message_id",
			mcp.Required(),
			mcp.Description("ID of the message to get context for"),
		),
		mcp.WithNumber("before",
			mcp.Description("Number of messages to include before the target message (default 5)"),
		),
		mcp.WithNumber("after",
			mcp.Description("Number of messages to include after the target message (default 5)"),
		),
	)

	sendMessageTool := mcp.NewTool("send_message",
		mcp.WithDescription("Send a WhatsApp message to a person or group. For group chats, use the JID"),
		mcp.WithString("recipient",
			mcp.Required(),
			mcp.Description("The recipient - either a phone number with country code but without + or other symbols, or a JID (e.g. '123456789@s.whatsapp.net' or a group JID like '123456789@g.us')"),
		),
		mcp.WithString("message",
			mcp.Required(),
			mcp.Description("The text of the message to send"),
		),
	)

	s.AddTool(searchContactsTool, searchContactsHandler)
	s.AddTool(listMessagesTool, listMessagesHandler)
	s.AddTool(listChatsTool, listChatsHandler)
	s.AddTool(getChatTool, getChatHandler)
	s.AddTool(getDirectChatByContactTool, getDirectChatByContactHandler)
	s.AddTool(getContactChatsTool, getContactChatsHandler)
	s.AddTool(getLastInteractionTool, getLastInteractionHandler)
	s.AddTool(getMessageContextTool, getMessageContextHandler)
	s.AddTool(sendMessageTool, sendMessageHandler)

	return s
}

// StartMCPServer starts the MCP server
func StartMCPServer(s *server.MCPServer) error {
	return server.ServeStdio(s)
}
