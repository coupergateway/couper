package mcp

import "encoding/json"

// Request represents a JSON-RPC 2.0 request.
type Request struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

// Response represents a JSON-RPC 2.0 response.
type Response struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *RPCError       `json:"error,omitempty"`
}

// RPCError represents a JSON-RPC 2.0 error object.
type RPCError struct {
	Code    int             `json:"code"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data,omitempty"`
}

// ToolCallParams represents the params of a tools/call request.
type ToolCallParams struct {
	Name      string          `json:"name"`
	Arguments json.RawMessage `json:"arguments,omitempty"`
}

// ToolsListResult represents the result of a tools/list response.
type ToolsListResult struct {
	Tools  []Tool `json:"tools"`
	Cursor string `json:"nextCursor,omitempty"`
}

// Tool represents a single tool in MCP tools/list.
type Tool struct {
	Name        string          `json:"name"`
	Description string          `json:"description,omitempty"`
	InputSchema json.RawMessage `json:"inputSchema,omitempty"`
}

// ParseRequest attempts to parse a JSON-RPC request from raw bytes.
// Returns nil if the data is not a valid JSON-RPC request.
func ParseRequest(data []byte) *Request {
	var req Request
	if err := json.Unmarshal(data, &req); err != nil {
		return nil
	}
	if req.JSONRPC != "2.0" || req.Method == "" {
		return nil
	}
	return &req
}

// NewMethodNotFoundError creates a JSON-RPC error response for a blocked tool call.
func NewMethodNotFoundError(id json.RawMessage, toolName string) []byte {
	errData, _ := json.Marshal(map[string]string{
		"tool":   toolName,
		"reason": "tool not allowed by gateway policy",
	})

	resp := Response{
		JSONRPC: "2.0",
		ID:      id,
		Error: &RPCError{
			Code:    -32601,
			Message: "Method not found",
			Data:    errData,
		},
	}

	b, _ := json.Marshal(resp)
	return b
}
