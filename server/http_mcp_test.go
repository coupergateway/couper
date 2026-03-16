package server_test

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/sirupsen/logrus"

	"github.com/coupergateway/couper/internal/test"
)

func TestMCPProxy_ToolsListFiltered(t *testing.T) {
	helper := test.New(t)

	shutdown, hook := newCouper("testdata/integration/mcp/01_couper.hcl", helper)
	defer shutdown()

	// tools/list — should filter to only allowed tools (get_weather, read_*) minus blocked (read_secret)
	body := `{"jsonrpc":"2.0","id":1,"method":"tools/list"}`
	req, err := http.NewRequest(http.MethodPost, "http://back.end:8080/mcp", strings.NewReader(body))
	helper.Must(err)
	req.Header.Set("Content-Type", "application/json")

	client := newClient()
	res, err := client.Do(req)
	helper.Must(err)

	if res.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", res.StatusCode)
	}

	respBody, err := io.ReadAll(res.Body)
	helper.Must(err)
	res.Body.Close()

	var rpcResp struct {
		Result struct {
			Tools []struct {
				Name string `json:"name"`
			} `json:"tools"`
		} `json:"result"`
	}
	helper.Must(json.Unmarshal(respBody, &rpcResp))

	toolNames := make(map[string]bool)
	for _, tool := range rpcResp.Result.Tools {
		toolNames[tool.Name] = true
	}

	// Should have: get_weather, read_file (matches read_*)
	// Should NOT have: read_secret (blocked), delete_file, exec_command, search_code (not in allowed)
	if !toolNames["get_weather"] {
		t.Error("expected get_weather to be allowed")
	}
	if !toolNames["read_file"] {
		t.Error("expected read_file to be allowed")
	}
	if toolNames["read_secret"] {
		t.Error("expected read_secret to be blocked")
	}
	if toolNames["delete_file"] {
		t.Error("expected delete_file to not be allowed")
	}
	if toolNames["exec_command"] {
		t.Error("expected exec_command to not be allowed")
	}
	if toolNames["search_code"] {
		t.Error("expected search_code to not be allowed")
	}

	// Verify log: Info entry for filtered tools/list with removed tool names
	var foundFilteredLog bool
	for _, e := range hook.AllEntries() {
		if e.Message == "mcp: filtered tools/list response" {
			foundFilteredLog = true
			if e.Level != logrus.InfoLevel {
				t.Errorf("expected info level, got %v", e.Level)
			}
			if total, ok := e.Data["total"].(int); !ok || total != 6 {
				t.Errorf("expected total=6, got %v", e.Data["total"])
			}
			if exposed, ok := e.Data["exposed"].(int); !ok || exposed != 2 {
				t.Errorf("expected exposed=2, got %v", e.Data["exposed"])
			}
			removed, _ := e.Data["removed"].(string)
			for _, name := range []string{"read_secret", "delete_file", "exec_command", "search_code"} {
				if !strings.Contains(removed, name) {
					t.Errorf("expected %q in removed list, got %q", name, removed)
				}
			}
			break
		}
	}
	if !foundFilteredLog {
		t.Error("expected 'mcp: filtered tools/list response' log entry")
	}
}

func TestMCPProxy_ToolsCallBlocked(t *testing.T) {
	helper := test.New(t)

	shutdown, hook := newCouper("testdata/integration/mcp/01_couper.hcl", helper)
	defer shutdown()

	// tools/call for a blocked tool — should return JSON-RPC error without calling backend
	body := `{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"delete_file","arguments":{}}}`
	req, err := http.NewRequest(http.MethodPost, "http://back.end:8080/mcp", strings.NewReader(body))
	helper.Must(err)
	req.Header.Set("Content-Type", "application/json")

	client := newClient()
	res, err := client.Do(req)
	helper.Must(err)

	if res.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", res.StatusCode)
	}

	respBody, err := io.ReadAll(res.Body)
	helper.Must(err)
	res.Body.Close()

	var rpcResp struct {
		Error *struct {
			Code    int    `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}
	helper.Must(json.Unmarshal(respBody, &rpcResp))

	if rpcResp.Error == nil {
		t.Fatal("expected JSON-RPC error for blocked tool")
	}
	if rpcResp.Error.Code != -32601 {
		t.Errorf("expected error code -32601, got %d", rpcResp.Error.Code)
	}

	// Verify log: Info entry for denied tool call with tool name
	var foundDeniedLog bool
	for _, e := range hook.AllEntries() {
		if e.Message == "mcp: tool call denied" {
			foundDeniedLog = true
			if e.Level != logrus.InfoLevel {
				t.Errorf("expected info level, got %v", e.Level)
			}
			if tool, ok := e.Data["tool"].(string); !ok || tool != "delete_file" {
				t.Errorf("expected tool=delete_file, got %v", e.Data["tool"])
			}
			break
		}
	}
	if !foundDeniedLog {
		t.Error("expected 'mcp: tool call denied' log entry")
	}
}

func TestMCPProxy_ToolsCallAllowed(t *testing.T) {
	helper := test.New(t)

	shutdown, hook := newCouper("testdata/integration/mcp/01_couper.hcl", helper)
	defer shutdown()

	// tools/call for an allowed tool — should pass through to backend
	body := `{"jsonrpc":"2.0","id":3,"method":"tools/call","params":{"name":"get_weather","arguments":{"city":"Berlin"}}}`
	req, err := http.NewRequest(http.MethodPost, "http://back.end:8080/mcp", strings.NewReader(body))
	helper.Must(err)
	req.Header.Set("Content-Type", "application/json")

	client := newClient()
	res, err := client.Do(req)
	helper.Must(err)

	if res.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", res.StatusCode)
	}

	respBody, err := io.ReadAll(res.Body)
	helper.Must(err)
	res.Body.Close()

	var rpcResp struct {
		Result interface{} `json:"result"`
		Error  *struct{}   `json:"error"`
	}
	helper.Must(json.Unmarshal(respBody, &rpcResp))

	if rpcResp.Error != nil {
		t.Error("expected no error for allowed tool call")
	}
	if rpcResp.Result == nil {
		t.Error("expected result for allowed tool call")
	}

	// Verify log: no denied entry, no filtered entry for this call
	for _, e := range hook.AllEntries() {
		if e.Message == "mcp: tool call denied" {
			t.Error("unexpected 'mcp: tool call denied' log entry for allowed tool")
		}
	}
}

func TestMCPProxy_Passthrough(t *testing.T) {
	helper := test.New(t)

	shutdown, hook := newCouper("testdata/integration/mcp/01_couper.hcl", helper)
	defer shutdown()

	// No filters set — all tools should pass through
	body := `{"jsonrpc":"2.0","id":4,"method":"tools/list"}`
	req, err := http.NewRequest(http.MethodPost, "http://back.end:8080/mcp-passthrough", strings.NewReader(body))
	helper.Must(err)
	req.Header.Set("Content-Type", "application/json")

	client := newClient()
	res, err := client.Do(req)
	helper.Must(err)

	if res.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", res.StatusCode)
	}

	respBody, err := io.ReadAll(res.Body)
	helper.Must(err)
	res.Body.Close()

	var rpcResp struct {
		Result struct {
			Tools []struct {
				Name string `json:"name"`
			} `json:"tools"`
		} `json:"result"`
	}
	helper.Must(json.Unmarshal(respBody, &rpcResp))

	if len(rpcResp.Result.Tools) != 6 {
		t.Errorf("expected all 6 tools in passthrough mode, got %d", len(rpcResp.Result.Tools))
	}

	// Verify log: no filtered entry in passthrough mode
	for _, e := range hook.AllEntries() {
		if e.Message == "mcp: filtered tools/list response" {
			t.Error("unexpected 'mcp: filtered tools/list response' in passthrough mode")
		}
	}
}

func TestMCPProxy_BlockOnly(t *testing.T) {
	helper := test.New(t)

	shutdown, hook := newCouper("testdata/integration/mcp/01_couper.hcl", helper)
	defer shutdown()

	// Only blocked_tools set — delete_* and exec_* should be removed
	body := `{"jsonrpc":"2.0","id":5,"method":"tools/list"}`
	req, err := http.NewRequest(http.MethodPost, "http://back.end:8080/mcp-block-only", strings.NewReader(body))
	helper.Must(err)
	req.Header.Set("Content-Type", "application/json")

	client := newClient()
	res, err := client.Do(req)
	helper.Must(err)

	if res.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", res.StatusCode)
	}

	respBody, err := io.ReadAll(res.Body)
	helper.Must(err)
	res.Body.Close()

	var rpcResp struct {
		Result struct {
			Tools []struct {
				Name string `json:"name"`
			} `json:"tools"`
		} `json:"result"`
	}
	helper.Must(json.Unmarshal(respBody, &rpcResp))

	toolNames := make(map[string]bool)
	for _, tool := range rpcResp.Result.Tools {
		toolNames[tool.Name] = true
	}

	// Should have: get_weather, read_file, read_secret, search_code
	// Should NOT have: delete_file, exec_command
	if !toolNames["get_weather"] {
		t.Error("expected get_weather")
	}
	if !toolNames["search_code"] {
		t.Error("expected search_code")
	}
	if toolNames["delete_file"] {
		t.Error("expected delete_file to be blocked")
	}
	if toolNames["exec_command"] {
		t.Error("expected exec_command to be blocked")
	}

	if len(rpcResp.Result.Tools) != 4 {
		t.Errorf("expected 4 tools after blocking delete_* and exec_*, got %d", len(rpcResp.Result.Tools))
	}

	// Verify log: Info entry with removed tools
	var foundFilteredLog bool
	for _, e := range hook.AllEntries() {
		if e.Message == "mcp: filtered tools/list response" {
			foundFilteredLog = true
			if total, ok := e.Data["total"].(int); !ok || total != 6 {
				t.Errorf("expected total=6, got %v", e.Data["total"])
			}
			if exposed, ok := e.Data["exposed"].(int); !ok || exposed != 4 {
				t.Errorf("expected exposed=4, got %v", e.Data["exposed"])
			}
			removed, _ := e.Data["removed"].(string)
			if !strings.Contains(removed, "delete_file") {
				t.Errorf("expected delete_file in removed, got %q", removed)
			}
			if !strings.Contains(removed, "exec_command") {
				t.Errorf("expected exec_command in removed, got %q", removed)
			}
			break
		}
	}
	if !foundFilteredLog {
		t.Error("expected 'mcp: filtered tools/list response' log entry")
	}
}

func TestMCPProxy_NonJSONRPC_Passthrough(t *testing.T) {
	helper := test.New(t)

	shutdown, _ := newCouper("testdata/integration/mcp/01_couper.hcl", helper)
	defer shutdown()

	// Non-JSON-RPC request should pass through to backend
	req, err := http.NewRequest(http.MethodGet, "http://back.end:8080/mcp", nil)
	helper.Must(err)

	client := newClient()
	res, err := client.Do(req)
	helper.Must(err)

	if res.StatusCode == 0 {
		t.Error("expected a response from backend")
	}
}
