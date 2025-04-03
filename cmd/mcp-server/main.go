package main

import (
	"log"

	"github.com/mbenaiss/whatsapp-mcp/mcp"
)

func main() {
	mcpServer := mcp.NewMCPServer("WhatsApp MCP API", "1.0.0")
	if err := mcp.StartMCPServer(mcpServer); err != nil {
		log.Fatalf("Failed to start MCP server: %v", err)
	}
}
