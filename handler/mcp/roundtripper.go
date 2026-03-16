package mcp

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/sirupsen/logrus"

	"github.com/coupergateway/couper/errors"
	"github.com/coupergateway/couper/eval"
	"github.com/coupergateway/couper/internal/seetie"
)

// MCPRoundTripper wraps a backend transport and applies MCP tool filtering.
type MCPRoundTripper struct {
	backend http.RoundTripper
	context *hclsyntax.Body
	logger  *logrus.Entry
}

// NewMCPRoundTripper creates an MCP-aware round tripper that filters tools.
func NewMCPRoundTripper(backend http.RoundTripper, ctx *hclsyntax.Body, logger *logrus.Entry) *MCPRoundTripper {
	return &MCPRoundTripper{
		backend: backend,
		context: ctx,
		logger:  logger,
	}
}

func (m *MCPRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	// Read the request body; preserve nil for empty/missing bodies
	var reqBody []byte
	if req.Body != nil {
		var err error
		reqBody, err = io.ReadAll(req.Body)
		if err != nil {
			return nil, err
		}
		req.Body.Close()
	}

	// Try to parse as JSON-RPC
	rpcReq := ParseRequest(reqBody)
	if rpcReq == nil {
		// Not a valid JSON-RPC request — pass through transparently
		m.logger.Debug("mcp: non-JSON-RPC request, passing through")
		m.restoreBody(req, reqBody)
		return m.backend.RoundTrip(req)
	}

	// Build tool filter from HCL context (evaluated per-request for JWT claims etc.)
	filter := m.buildFilter(req)

	// Handle tools/call — fail closed if tool name cannot be determined
	if rpcReq.Method == "tools/call" && filter.HasRules() {
		var params ToolCallParams
		if err := json.Unmarshal(rpcReq.Params, &params); err != nil || params.Name == "" {
			m.logger.Info("mcp: tool call denied, unable to determine tool name")
			body := NewMethodNotFoundError(rpcReq.ID, "")
			return newJSONResponse(http.StatusOK, body),
				errors.BetaMcpToolBlocked.Messagef("tool name could not be determined")
		}

		if !filter.IsAllowed(params.Name) {
			m.logger.WithField("tool", params.Name).Info("mcp: tool call denied")
			body := NewMethodNotFoundError(rpcReq.ID, params.Name)
			return newJSONResponse(http.StatusOK, body),
				errors.BetaMcpToolBlocked.Messagef("tool %q not allowed by gateway policy", params.Name)
		}

		m.logger.WithField("tool", params.Name).Debug("mcp: tool call allowed")
	}

	// Forward to backend
	m.restoreBody(req, reqBody)
	resp, err := m.backend.RoundTrip(req)
	if err != nil {
		return nil, err
	}

	// Handle tools/list response — filter the tools array
	if rpcReq.Method == "tools/list" && resp.StatusCode == http.StatusOK {
		return m.filterToolsListResponse(resp, filter)
	}

	return resp, nil
}

// restoreBody sets req.Body back from the buffered bytes, preserving nil for empty bodies.
func (m *MCPRoundTripper) restoreBody(req *http.Request, body []byte) {
	if len(body) == 0 {
		req.Body = nil
		return
	}
	req.Body = io.NopCloser(bytes.NewReader(body))
}

func (m *MCPRoundTripper) buildFilter(req *http.Request) *ToolFilter {
	filter := &ToolFilter{}

	evalCtx := eval.ContextFromRequest(req)
	if evalCtx == nil {
		return filter
	}

	hclCtx := evalCtx.HCLContextSync()

	if val, err := eval.ValueFromBodyAttribute(hclCtx, m.context, "allowed_tools"); err == nil {
		filter.Allowed = seetie.ValueToStringSlice(val)
	}

	if val, err := eval.ValueFromBodyAttribute(hclCtx, m.context, "blocked_tools"); err == nil {
		filter.Blocked = seetie.ValueToStringSlice(val)
	}

	return filter
}

func (m *MCPRoundTripper) filterToolsListResponse(resp *http.Response, filter *ToolFilter) (*http.Response, error) {
	if !filter.HasRules() {
		return resp, nil
	}

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	resp.Body.Close()

	var rpcResp Response
	if err := json.Unmarshal(respBody, &rpcResp); err != nil {
		resp.Body = io.NopCloser(bytes.NewReader(respBody))
		return resp, nil
	}

	if rpcResp.Error != nil || rpcResp.Result == nil {
		resp.Body = io.NopCloser(bytes.NewReader(respBody))
		return resp, nil
	}

	var result ToolsListResult
	if err := json.Unmarshal(rpcResp.Result, &result); err != nil {
		resp.Body = io.NopCloser(bytes.NewReader(respBody))
		return resp, nil
	}

	totalTools := len(result.Tools)

	var removedTools []string
	for _, tool := range result.Tools {
		if !filter.IsAllowed(tool.Name) {
			removedTools = append(removedTools, tool.Name)
		}
	}

	result.Tools = filter.FilterTools(result.Tools)

	if len(removedTools) > 0 {
		m.logger.WithFields(logrus.Fields{
			"total":   totalTools,
			"exposed": len(result.Tools),
			"removed": strings.Join(removedTools, ", "),
		}).Info("mcp: filtered tools/list response")
	}

	m.logger.WithFields(logrus.Fields{
		"total":   totalTools,
		"exposed": len(result.Tools),
	}).Debug("mcp: tools/list filtered")

	newResult, err := json.Marshal(result)
	if err != nil {
		resp.Body = io.NopCloser(bytes.NewReader(respBody))
		return resp, nil
	}

	rpcResp.Result = newResult
	newBody, err := json.Marshal(rpcResp)
	if err != nil {
		resp.Body = io.NopCloser(bytes.NewReader(respBody))
		return resp, nil
	}

	resp.Body = io.NopCloser(bytes.NewReader(newBody))
	resp.ContentLength = int64(len(newBody))
	resp.Header.Set("Content-Length", strconv.Itoa(len(newBody)))

	return resp, nil
}

func newJSONResponse(statusCode int, body []byte) *http.Response {
	return &http.Response{
		StatusCode: statusCode,
		Header: http.Header{
			"Content-Type":   {"application/json"},
			"Content-Length": {strconv.Itoa(len(body))},
		},
		Body:          io.NopCloser(bytes.NewReader(body)),
		ContentLength: int64(len(body)),
	}
}
