package mcp

import (
	"encoding/json"
	"testing"
)

func TestParseRequest_Valid(t *testing.T) {
	data := []byte(`{"jsonrpc":"2.0","id":1,"method":"tools/list"}`)
	req := ParseRequest(data)
	if req == nil {
		t.Fatal("expected valid request, got nil")
	}
	if req.Method != "tools/list" {
		t.Errorf("method = %q, want %q", req.Method, "tools/list")
	}
}

func TestParseRequest_WithParams(t *testing.T) {
	data := []byte(`{"jsonrpc":"2.0","id":42,"method":"tools/call","params":{"name":"get_weather","arguments":{"city":"Berlin"}}}`)
	req := ParseRequest(data)
	if req == nil {
		t.Fatal("expected valid request, got nil")
	}
	if req.Method != "tools/call" {
		t.Errorf("method = %q, want %q", req.Method, "tools/call")
	}

	var params ToolCallParams
	if err := json.Unmarshal(req.Params, &params); err != nil {
		t.Fatalf("unmarshal params: %v", err)
	}
	if params.Name != "get_weather" {
		t.Errorf("params.Name = %q, want %q", params.Name, "get_weather")
	}
}

func TestParseRequest_InvalidJSON(t *testing.T) {
	data := []byte(`not json`)
	req := ParseRequest(data)
	if req != nil {
		t.Error("expected nil for invalid JSON")
	}
}

func TestParseRequest_WrongVersion(t *testing.T) {
	data := []byte(`{"jsonrpc":"1.0","id":1,"method":"tools/list"}`)
	req := ParseRequest(data)
	if req != nil {
		t.Error("expected nil for non-2.0 version")
	}
}

func TestParseRequest_MissingMethod(t *testing.T) {
	data := []byte(`{"jsonrpc":"2.0","id":1}`)
	req := ParseRequest(data)
	if req != nil {
		t.Error("expected nil for missing method")
	}
}

func TestNewMethodNotFoundError(t *testing.T) {
	id := json.RawMessage(`42`)
	body := NewMethodNotFoundError(id, "delete_file")

	var resp Response
	if err := json.Unmarshal(body, &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if resp.JSONRPC != "2.0" {
		t.Errorf("jsonrpc = %q, want %q", resp.JSONRPC, "2.0")
	}

	if string(resp.ID) != "42" {
		t.Errorf("id = %s, want 42", string(resp.ID))
	}

	if resp.Error == nil {
		t.Fatal("expected error, got nil")
	}

	if resp.Error.Code != -32601 {
		t.Errorf("error.code = %d, want %d", resp.Error.Code, -32601)
	}

	var data map[string]string
	if err := json.Unmarshal(resp.Error.Data, &data); err != nil {
		t.Fatalf("unmarshal error data: %v", err)
	}
	if data["tool"] != "delete_file" {
		t.Errorf("error.data.tool = %q, want %q", data["tool"], "delete_file")
	}
}

func TestParseRequest_StringID(t *testing.T) {
	data := []byte(`{"jsonrpc":"2.0","id":"abc-123","method":"tools/list"}`)
	req := ParseRequest(data)
	if req == nil {
		t.Fatal("expected valid request, got nil")
	}
	if string(req.ID) != `"abc-123"` {
		t.Errorf("id = %s, want %q", string(req.ID), `"abc-123"`)
	}
}
