package mcp

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"

	"jupyter-bridge/pkg/jupyter"
)

type Server struct {
	sessionManager *jupyter.SessionManager
	tools          []Tool
}

func NewServer(sm *jupyter.SessionManager) *Server {
	return &Server{
		sessionManager: sm,
		tools:          GetToolDefinitions(),
	}
}

// RunStdio starts the MCP JSON-RPC 2.0 loop over stdin and stdout
func (s *Server) RunStdio(ctx context.Context) error {
	reader := bufio.NewReader(os.Stdin)
	writer := bufio.NewWriter(os.Stdout)

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		line, err := reader.ReadBytes('\n')
		if err != nil {
			if err == io.EOF {
				return nil
			}
			return err
		}

		if len(line) == 0 || (len(line) == 1 && line[0] == '\n') {
			continue
		}

		var req JSONRPCRequest
		if err := json.Unmarshal(line, &req); err != nil {
			s.sendError(writer, nil, -32700, "Parse error", err.Error())
			continue
		}

		resp := s.handleRequest(ctx, &req)
		if resp != nil {
			resBytes, err := json.Marshal(resp)
			if err == nil {
				_, _ = writer.Write(resBytes)
				_, _ = writer.WriteString("\n")
				_ = writer.Flush()
			}
		}
	}
}

func (s *Server) handleRequest(ctx context.Context, req *JSONRPCRequest) *JSONRPCResponse {
	switch req.Method {
	case "initialize":
		var params InitializeParams
		_ = json.Unmarshal(req.Params, &params)

		return &JSONRPCResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Result: InitializeResult{
				ProtocolVersion: "2024-11-05",
				Capabilities: ServerCapabilities{
					Tools: &ToolsCapability{ListChanged: false},
				},
				ServerInfo: ServerInfo{
					Name:    "jupyter-bridge",
					Version: "1.0.0",
				},
			},
		}

	case "notifications/initialized":
		// Notification - no response needed
		return nil

	case "ping":
		return &JSONRPCResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Result:  map[string]interface{}{},
		}

	case "tools/list":
		return &JSONRPCResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Result: map[string]interface{}{
				"tools": s.tools,
			},
		}

	case "tools/call":
		var params CallToolParams
		if err := json.Unmarshal(req.Params, &params); err != nil {
			return &JSONRPCResponse{
				JSONRPC: "2.0",
				ID:      req.ID,
				Error: &JSONRPCError{
					Code:    -32602,
					Message: "Invalid params",
					Data:    err.Error(),
				},
			}
		}

		out, isErr, err := ExecuteTool(ctx, s.sessionManager, params.Name, params.Arguments)
		if err != nil {
			return &JSONRPCResponse{
				JSONRPC: "2.0",
				ID:      req.ID,
				Error: &JSONRPCError{
					Code:    -32000,
					Message: "Tool execution failed",
					Data:    err.Error(),
				},
			}
		}

		return &JSONRPCResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Result: CallToolResult{
				Content: []ContentItem{
					{
						Type: "text",
						Text: out,
					},
				},
				IsError: isErr,
			},
		}

	default:
		// If it's a notification without an ID, don't return error
		if req.ID == nil {
			return nil
		}
		return &JSONRPCResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Error: &JSONRPCError{
				Code:    -32601,
				Message: fmt.Sprintf("Method not found: %s", req.Method),
			},
		}
	}
}

func (s *Server) sendError(w *bufio.Writer, id interface{}, code int, message string, data interface{}) {
	resp := JSONRPCResponse{
		JSONRPC: "2.0",
		ID:      id,
		Error: &JSONRPCError{
			Code:    code,
			Message: message,
			Data:    data,
		},
	}
	bytes, err := json.Marshal(resp)
	if err == nil {
		_, _ = w.Write(bytes)
		_, _ = w.WriteString("\n")
		_ = w.Flush()
	} else {
		log.Printf("Error sending error response: %v", err)
	}
}
