package jupyter

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

func (c *Client) GetWSURL(kernelID string) string {
	base := c.cfg.JupyterURL
	if strings.HasPrefix(base, "https://") {
		base = "wss://" + strings.TrimPrefix(base, "https://")
	} else if strings.HasPrefix(base, "http://") {
		base = "ws://" + strings.TrimPrefix(base, "http://")
	}
	return fmt.Sprintf("%s/api/kernels/%s/channels?token=%s", base, kernelID, c.cfg.JupyterToken)
}

// ExecuteCode runs Python/IPython code on the specified kernel via WebSocket channels
func (c *Client) ExecuteCode(ctx context.Context, kernelID, code string, timeout time.Duration) (*ExecutionResult, error) {
	if timeout <= 0 {
		timeout = 120 * time.Second
	}

	wsURL := c.GetWSURL(kernelID)
	headers := http.Header{}
	headers.Set("User-Agent", c.cfg.UserAgent)

	dialer := websocket.DefaultDialer
	dialer.HandshakeTimeout = 15 * time.Second

	conn, resp, err := dialer.DialContext(ctx, wsURL, headers)
	if err != nil {
		if resp != nil {
			return nil, fmt.Errorf("websocket dial failed (status %d): %w", resp.StatusCode, err)
		}
		return nil, fmt.Errorf("websocket dial failed: %w", err)
	}
	defer conn.Close()

	msgID := uuid.New().String()
	sessionID := uuid.New().String()

	execMsg := WSMessage{
		Header: MsgHeader{
			MsgID:    msgID,
			MsgType:  "execute_request",
			Username: "jupyter-bridge-agent",
			Session:  sessionID,
			Version:  "5.3",
			Date:     time.Now().UTC().Format(time.RFC3339),
		},
		ParentHeader: map[string]interface{}{},
		Metadata:     map[string]interface{}{},
		Content: map[string]interface{}{
			"code":             code,
			"silent":           false,
			"store_history":    true,
			"user_expressions": map[string]interface{}{},
			"allow_stdin":      false,
			"stop_on_error":    true,
		},
		Channel: "shell",
	}

	if err := conn.WriteJSON(execMsg); err != nil {
		return nil, fmt.Errorf("failed to send execute_request: %w", err)
	}

	res := &ExecutionResult{
		Success: true,
	}

	done := make(chan error, 1)

	go func() {
		for {
			var raw WSMessage
			err := conn.ReadJSON(&raw)
			if err != nil {
				done <- err
				return
			}

			// Check if message belongs to our execution request
			parentMsgID, _ := raw.ParentHeader["msg_id"].(string)
			if parentMsgID != msgID {
				continue
			}

			switch raw.Header.MsgType {
			case "stream":
				streamName, _ := raw.Content["name"].(string)
				text, _ := raw.Content["text"].(string)
				if streamName == "stderr" {
					res.Stderr += text
				} else {
					res.Stdout += text
				}

			case "execute_result", "display_data":
				data, ok := raw.Content["data"].(map[string]interface{})
				if ok {
					if plain, ok := data["text/plain"].(string); ok {
						if res.Result != "" {
							res.Result += "\n" + plain
						} else {
							res.Result = plain
						}
					}
				}
				if count, ok := raw.Content["execution_count"].(float64); ok {
					res.ExecutionN = int(count)
				}

			case "error":
				res.Success = false
				if tb, ok := raw.Content["traceback"].([]interface{}); ok {
					for _, line := range tb {
						if s, ok := line.(string); ok {
							res.Errors = append(res.Errors, s)
						}
					}
				} else {
					ename, _ := raw.Content["ename"].(string)
					evalue, _ := raw.Content["evalue"].(string)
					res.Errors = append(res.Errors, fmt.Sprintf("%s: %s", ename, evalue))
				}

			case "status":
				state, _ := raw.Content["execution_state"].(string)
				if state == "idle" {
					done <- nil
					return
				}
			}
		}
	}()

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-time.After(timeout):
		_ = c.InterruptKernel(context.Background(), kernelID)
		return nil, fmt.Errorf("execution timed out after %v", timeout)
	case err := <-done:
		if err != nil {
			return nil, fmt.Errorf("websocket error during execution: %w", err)
		}
		return res, nil
	}
}
