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

	// tools/call for a blocked tool — should return 403 with beta_mcp_tool_blocked error
	body := `{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"delete_file","arguments":{}}}`
	req, err := http.NewRequest(http.MethodPost, "http://back.end:8080/mcp", strings.NewReader(body))
	helper.Must(err)
	req.Header.Set("Content-Type", "application/json")

	client := newClient()
	res, err := client.Do(req)
	helper.Must(err)

	if res.StatusCode != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", res.StatusCode)
	}

	// Verify log: error entry with beta_mcp_tool_blocked error type
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

	// Verify error_type in access log
	var foundErrorType bool
	for _, e := range hook.AllEntries() {
		if et, ok := e.Data["error_type"].(string); ok && et == "beta_mcp_tool_blocked" {
			foundErrorType = true
			break
		}
	}
	if !foundErrorType {
		t.Error("expected error_type=beta_mcp_tool_blocked in log")
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

	shutdown, hook := newCouper("testdata/integration/mcp/01_couper.hcl", helper)
	defer shutdown()

	// Non-JSON-RPC POST with plain text body should pass through to backend
	req, err := http.NewRequest(http.MethodPost, "http://back.end:8080/mcp", strings.NewReader("not json"))
	helper.Must(err)
	req.Header.Set("Content-Type", "text/plain")

	client := newClient()
	res, err := client.Do(req)
	helper.Must(err)

	// Backend /mcp handler returns 400 for non-JSON — confirms the request reached the backend
	if res.StatusCode != http.StatusBadRequest {
		t.Errorf("expected 400 from backend for non-JSON body, got %d", res.StatusCode)
	}

	// Verify no MCP filtering or denial logs
	for _, e := range hook.AllEntries() {
		if e.Message == "mcp: tool call denied" || e.Message == "mcp: filtered tools/list response" {
			t.Errorf("unexpected MCP log entry for non-JSON-RPC request: %q", e.Message)
		}
	}
}

// ── MCP OAuth proxy tests ───────────────────────────────────────────────────

func TestMCPProxy_OAuthProtectedResource(t *testing.T) {
	helper := test.New(t)

	shutdown, _ := newCouper("testdata/integration/mcp/02_couper.hcl", helper)
	defer shutdown()

	client := newClient()

	// The auto-registered OAuth endpoint should rewrite the resource field
	// to match the proxy URL and authorization_servers to the proxy origin.
	req, err := http.NewRequest(http.MethodGet, "http://back.end:8080/.well-known/oauth-protected-resource", nil)
	helper.Must(err)

	res, err := client.Do(req)
	helper.Must(err)

	if res.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", res.StatusCode)
	}

	respBody, err := io.ReadAll(res.Body)
	helper.Must(err)
	res.Body.Close()

	var metadata map[string]interface{}
	helper.Must(json.Unmarshal(respBody, &metadata))

	// resource should be rewritten to the proxy URL (scheme://host + mcpEndpoint)
	resource, ok := metadata["resource"].(string)
	if !ok {
		t.Fatal("missing resource field")
	}
	if resource != "http://back.end:8080/mcp" {
		t.Errorf("resource = %q, want %q", resource, "http://back.end:8080/mcp")
	}

	// authorization_servers should point to proxy origin
	authServers, ok := metadata["authorization_servers"].([]interface{})
	if !ok || len(authServers) == 0 {
		t.Fatal("missing authorization_servers")
	}
	if authServers[0] != "http://back.end:8080/mcp" {
		t.Errorf("authorization_servers[0] = %q, want %q", authServers[0], "http://back.end:8080/mcp")
	}
}

func TestMCPProxy_OAuthProtectedResourceWithMCPSuffix(t *testing.T) {
	helper := test.New(t)

	shutdown, _ := newCouper("testdata/integration/mcp/02_couper.hcl", helper)
	defer shutdown()

	client := newClient()

	// MCP clients also try /.well-known/oauth-protected-resource/mcp
	req, err := http.NewRequest(http.MethodGet, "http://back.end:8080/.well-known/oauth-protected-resource/mcp", nil)
	helper.Must(err)

	res, err := client.Do(req)
	helper.Must(err)

	if res.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", res.StatusCode)
	}

	respBody, err := io.ReadAll(res.Body)
	helper.Must(err)
	res.Body.Close()

	var metadata map[string]interface{}
	helper.Must(json.Unmarshal(respBody, &metadata))

	resource, _ := metadata["resource"].(string)
	if resource != "http://back.end:8080/mcp" {
		t.Errorf("resource = %q, want %q", resource, "http://back.end:8080/mcp")
	}
}

func TestMCPProxy_OAuthAuthorizationServer(t *testing.T) {
	helper := test.New(t)

	shutdown, _ := newCouper("testdata/integration/mcp/02_couper.hcl", helper)
	defer shutdown()

	client := newClient()

	req, err := http.NewRequest(http.MethodGet, "http://back.end:8080/.well-known/oauth-authorization-server", nil)
	helper.Must(err)

	res, err := client.Do(req)
	helper.Must(err)

	if res.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", res.StatusCode)
	}

	respBody, err := io.ReadAll(res.Body)
	helper.Must(err)
	res.Body.Close()

	var metadata map[string]interface{}
	helper.Must(json.Unmarshal(respBody, &metadata))

	// issuer, token_endpoint, registration_endpoint should be rewritten to proxy
	if issuer, _ := metadata["issuer"].(string); issuer != "http://back.end:8080/mcp" {
		t.Errorf("issuer = %q, want %q", issuer, "http://back.end:8080/mcp")
	}
	if tokenEP, _ := metadata["token_endpoint"].(string); !strings.HasSuffix(tokenEP, "/mcp/token") {
		t.Errorf("token_endpoint should end with /mcp/token, got %q", tokenEP)
	}
	if regEP, _ := metadata["registration_endpoint"].(string); !strings.HasSuffix(regEP, "/mcp/register") {
		t.Errorf("registration_endpoint should end with /mcp/register, got %q", regEP)
	}

	// authorization_endpoint MUST stay pointing at upstream (browser redirect)
	authEP, _ := metadata["authorization_endpoint"].(string)
	if !strings.Contains(authEP, "/authorize") {
		t.Errorf("authorization_endpoint should point to upstream, got %q", authEP)
	}
}

func TestMCPProxy_OAuthTokenResourceRewrite(t *testing.T) {
	helper := test.New(t)

	shutdown, _ := newCouper("testdata/integration/mcp/02_couper.hcl", helper)
	defer shutdown()

	client := newClient()

	// POST /mcp/token with resource=proxy should be rewritten to resource=upstream
	body := "grant_type=authorization_code&code=test&resource=http%3A%2F%2Fback.end%3A8080%2Fmcp"
	req, err := http.NewRequest(http.MethodPost, "http://back.end:8080/mcp/token", strings.NewReader(body))
	helper.Must(err)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	res, err := client.Do(req)
	helper.Must(err)

	respBody, err := io.ReadAll(res.Body)
	helper.Must(err)
	res.Body.Close()

	if res.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d; body: %s", res.StatusCode, string(respBody))
	}

	// The test backend echoes the resource param it received
	var tokenResp map[string]interface{}
	helper.Must(json.Unmarshal(respBody, &tokenResp))

	resource, _ := tokenResp["resource"].(string)
	if !strings.Contains(resource, testBackend.Addr()) {
		t.Errorf("resource should be rewritten to upstream origin, got %q (expected to contain %q)", resource, testBackend.Addr())
	}
}

func TestMCPProxy_OAuthRegister(t *testing.T) {
	helper := test.New(t)

	shutdown, _ := newCouper("testdata/integration/mcp/02_couper.hcl", helper)
	defer shutdown()

	client := newClient()

	body := `{"client_name":"test","redirect_uris":["http://localhost/callback"]}`
	req, err := http.NewRequest(http.MethodPost, "http://back.end:8080/mcp/register", strings.NewReader(body))
	helper.Must(err)
	req.Header.Set("Content-Type", "application/json")

	res, err := client.Do(req)
	helper.Must(err)

	if res.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201, got %d", res.StatusCode)
	}

	respBody, err := io.ReadAll(res.Body)
	helper.Must(err)
	res.Body.Close()

	var regResp map[string]interface{}
	helper.Must(json.Unmarshal(respBody, &regResp))

	if _, ok := regResp["client_id"]; !ok {
		t.Error("expected client_id in response")
	}
}
