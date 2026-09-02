package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"jupyter-bridge/pkg/jupyter"
)

func GetToolDefinitions() []Tool {
	return []Tool{
		{
			Name:        "execute_code",
			Description: "Executes Python code in the remote Jupyter kernel on your VPS. Keeps memory, variables, imports, and functions alive across calls. Supports IPython magics (%time, !shell, etc.)",
			InputSchema: ToolSchema{
				Type: "object",
				Properties: map[string]Property{
					"code": {
						Type:        "string",
						Description: "The Python code snippet to execute.",
					},
					"session_name": {
						Type:        "string",
						Description: "Optional session name for multi-kernel multiplexing (default: 'default').",
					},
					"timeout_seconds": {
						Type:        "integer",
						Description: "Max execution timeout in seconds (default: 120).",
					},
				},
				Required: []string{"code"},
			},
		},
		{
			Name:        "run_bash_command",
			Description: "Executes a shell/bash command directly in the remote container/VPS environment.",
			InputSchema: ToolSchema{
				Type: "object",
				Properties: map[string]Property{
					"command": {
						Type:        "string",
						Description: "The bash command to run (e.g., 'pip install package', 'df -h', 'nvidia-smi', 'curl ...').",
					},
					"session_name": {
						Type:        "string",
						Description: "Optional session name (default: 'default').",
					},
				},
				Required: []string{"command"},
			},
		},
		{
			Name:        "system_info",
			Description: "Inspects the remote VPS hardware, CPU, RAM, disk space, OS, and GPU status.",
			InputSchema: ToolSchema{
				Type:       "object",
				Properties: map[string]Property{},
			},
		},
		{
			Name:        "list_files",
			Description: "Lists files and directories in the remote Jupyter workspace.",
			InputSchema: ToolSchema{
				Type: "object",
				Properties: map[string]Property{
					"path": {
						Type:        "string",
						Description: "Relative directory path to list (e.g., '', 'work', 'data').",
					},
				},
			},
		},
		{
			Name:        "read_file",
			Description: "Reads the content of a file or notebook from the remote Jupyter workspace.",
			InputSchema: ToolSchema{
				Type: "object",
				Properties: map[string]Property{
					"path": {
						Type:        "string",
						Description: "Path to the file to read (e.g. 'script.py', 'notebook.ipynb').",
					},
				},
				Required: []string{"path"},
			},
		},
		{
			Name:        "write_file",
			Description: "Writes, updates, or creates a file or script on the remote VPS workspace.",
			InputSchema: ToolSchema{
				Type: "object",
				Properties: map[string]Property{
					"path": {
						Type:        "string",
						Description: "Target file path (e.g. 'train.py', 'data.json').",
					},
					"content": {
						Type:        "string",
						Description: "Full text content to write.",
					},
				},
				Required: []string{"path", "content"},
			},
		},
		{
			Name:        "delete_file",
			Description: "Deletes a file or directory on the remote VPS.",
			InputSchema: ToolSchema{
				Type: "object",
				Properties: map[string]Property{
					"path": {
						Type:        "string",
						Description: "Target file or folder path to delete.",
					},
				},
				Required: []string{"path"},
			},
		},
		{
			Name:        "list_sessions",
			Description: "Lists all active kernel sessions, their status, and kernel IDs.",
			InputSchema: ToolSchema{
				Type:       "object",
				Properties: map[string]Property{},
			},
		},
		{
			Name:        "create_session",
			Description: "Creates a new named kernel session for isolated code execution.",
			InputSchema: ToolSchema{
				Type: "object",
				Properties: map[string]Property{
					"session_name": {
						Type:        "string",
						Description: "Unique name for the session (e.g. 'training', 'data-pipeline').",
					},
				},
				Required: []string{"session_name"},
			},
		},
		{
			Name:        "restart_session",
			Description: "Restarts a kernel session, clearing RAM memory and resetting Python state.",
			InputSchema: ToolSchema{
				Type: "object",
				Properties: map[string]Property{
					"session_name": {
						Type:        "string",
						Description: "Session name to restart (default: 'default').",
					},
				},
			},
		},
		{
			Name:        "close_session",
			Description: "Terminates and destroys a kernel session on the remote server.",
			InputSchema: ToolSchema{
				Type: "object",
				Properties: map[string]Property{
					"session_name": {
						Type:        "string",
						Description: "Session name to terminate.",
					},
				},
				Required: []string{"session_name"},
			},
		},
	}
}

func ExecuteTool(ctx context.Context, sm *jupyter.SessionManager, name string, args map[string]interface{}) (string, bool, error) {
	client := sm.Client()

	switch name {
	case "execute_code":
		code, _ := args["code"].(string)
		if code == "" {
			return "Error: 'code' argument is required", true, nil
		}
		sessionName, _ := args["session_name"].(string)
		if sessionName == "" {
			sessionName = "default"
		}
		timeoutSec := 120
		if t, ok := args["timeout_seconds"].(float64); ok && t > 0 {
			timeoutSec = int(t)
		}

		kernelID, err := sm.GetOrCreateSession(ctx, sessionName)
		if err != nil {
			return fmt.Sprintf("Failed to get/create session: %v", err), true, nil
		}

		res, err := client.ExecuteCode(ctx, kernelID, code, time.Duration(timeoutSec)*time.Second)
		if err != nil {
			return fmt.Sprintf("Execution failed: %v", err), true, nil
		}

		var sb strings.Builder
		if res.Stdout != "" {
			sb.WriteString("=== STDOUT ===\n")
			sb.WriteString(res.Stdout)
			sb.WriteString("\n")
		}
		if res.Stderr != "" {
			sb.WriteString("=== STDERR ===\n")
			sb.WriteString(res.Stderr)
			sb.WriteString("\n")
		}
		if res.Result != "" {
			sb.WriteString("=== RESULT ===\n")
			sb.WriteString(res.Result)
			sb.WriteString("\n")
		}
		if len(res.Errors) > 0 {
			sb.WriteString("=== ERRORS ===\n")
			for _, e := range res.Errors {
				sb.WriteString(e)
				sb.WriteString("\n")
			}
		}

		output := sb.String()
		if output == "" {
			output = "(Code executed successfully with no output)"
		}
		return output, !res.Success, nil

	case "run_bash_command":
		cmd, _ := args["command"].(string)
		if cmd == "" {
			return "Error: 'command' argument is required", true, nil
		}
		sessionName, _ := args["session_name"].(string)
		if sessionName == "" {
			sessionName = "default"
		}

		code := fmt.Sprintf("!%s", cmd)
		kernelID, err := sm.GetOrCreateSession(ctx, sessionName)
		if err != nil {
			return fmt.Sprintf("Failed to get/create session: %v", err), true, nil
		}

		res, err := client.ExecuteCode(ctx, kernelID, code, 180*time.Second)
		if err != nil {
			return fmt.Sprintf("Command failed: %v", err), true, nil
		}

		output := res.Stdout
		if res.Stderr != "" {
			output += "\n" + res.Stderr
		}
		if len(res.Errors) > 0 {
			output += "\nErrors: " + strings.Join(res.Errors, "\n")
		}
		if output == "" {
			output = "(Command completed with no output)"
		}
		return output, !res.Success, nil

	case "system_info":
		kernelID, err := sm.GetOrCreateSession(ctx, "default")
		if err != nil {
			return fmt.Sprintf("Failed to get session: %v", err), true, nil
		}
		code := `import os, subprocess, platform, sys
print('=== SYSTEM & OS ===')
print(f'Platform: {platform.platform()}')
print(f'Python: {platform.python_version()}')
print(f'CWD: {os.getcwd()}')
print('\n=== MEMORY & SWAP ===')
print(subprocess.getoutput('free -h 2>/dev/null || cat /proc/meminfo | head -n 5'))
print('\n=== DISK SPACE ===')
print(subprocess.getoutput('df -h /home/jovyan 2>/dev/null || df -h .'))
print('\n=== GPU (IF ANY) ===')
print(subprocess.getoutput('nvidia-smi 2>/dev/null || echo "No NVIDIA GPU detected"'))
`
		res, err := client.ExecuteCode(ctx, kernelID, code, 30*time.Second)
		if err != nil {
			return fmt.Sprintf("System info failed: %v", err), true, nil
		}
		return res.Stdout, false, nil

	case "list_files":
		path, _ := args["path"].(string)
		item, err := client.ListContents(ctx, path)
		if err != nil {
			return fmt.Sprintf("Failed to list files: %v", err), true, nil
		}

		items, ok := item.Content.([]interface{})
		if !ok || len(items) == 0 {
			return fmt.Sprintf("Directory '%s' is empty.", path), false, nil
		}

		var sb strings.Builder
		sb.WriteString(fmt.Sprintf("Directory: /%s\n", strings.Trim(path, "/")))
		for _, raw := range items {
			m, ok := raw.(map[string]interface{})
			if !ok {
				continue
			}
			t, _ := m["type"].(string)
			n, _ := m["name"].(string)
			p, _ := m["path"].(string)
			size, _ := m["size"].(float64)
			sb.WriteString(fmt.Sprintf(" - [%s] %s (path: %s, size: %.0f bytes)\n", t, n, p, size))
		}
		return sb.String(), false, nil

	case "read_file":
		path, _ := args["path"].(string)
		if path == "" {
			return "Error: 'path' argument is required", true, nil
		}
		item, err := client.ListContents(ctx, path)
		if err != nil {
			return fmt.Sprintf("Failed to read file: %v", err), true, nil
		}
		if item.Type == "notebook" {
			b, _ := json.MarshalIndent(item.Content, "", "  ")
			return string(b), false, nil
		}
		if str, ok := item.Content.(string); ok {
			return str, false, nil
		}
		b, _ := json.MarshalIndent(item.Content, "", "  ")
		return string(b), false, nil

	case "write_file":
		path, _ := args["path"].(string)
		content, _ := args["content"].(string)
		if path == "" {
			return "Error: 'path' is required", true, nil
		}
		err := client.SaveFile(ctx, path, content, "file")
		if err != nil {
			return fmt.Sprintf("Failed to write file: %v", err), true, nil
		}
		return fmt.Sprintf("Successfully saved '%s'", path), false, nil

	case "delete_file":
		path, _ := args["path"].(string)
		if path == "" {
			return "Error: 'path' is required", true, nil
		}
		err := client.DeleteFile(ctx, path)
		if err != nil {
			return fmt.Sprintf("Failed to delete '%s': %v", path, err), true, nil
		}
		return fmt.Sprintf("Deleted '%s'", path), false, nil

	case "list_sessions":
		sessions, err := sm.ListSessions(ctx)
		if err != nil {
			return fmt.Sprintf("Failed to list sessions: %v", err), true, nil
		}
		if len(sessions) == 0 {
			return "No active named sessions (will spawn on demand).", false, nil
		}
		var sb strings.Builder
		sb.WriteString("Active Sessions:\n")
		for _, s := range sessions {
			sb.WriteString(fmt.Sprintf(" - Session '%s' -> Kernel ID: %s (started: %s)\n", s.Name, s.KernelID, s.CreatedAt.Format(time.RFC3339)))
		}
		return sb.String(), false, nil

	case "create_session":
		name, _ := args["session_name"].(string)
		if name == "" {
			return "Error: 'session_name' is required", true, nil
		}
		kID, err := sm.GetOrCreateSession(ctx, name)
		if err != nil {
			return fmt.Sprintf("Failed to create session: %v", err), true, nil
		}
		return fmt.Sprintf("Session '%s' created with Kernel ID: %s", name, kID), false, nil

	case "restart_session":
		name, _ := args["session_name"].(string)
		if name == "" {
			name = "default"
		}
		err := sm.RestartSession(ctx, name)
		if err != nil {
			return fmt.Sprintf("Failed to restart session '%s': %v", name, err), true, nil
		}
		return fmt.Sprintf("Session '%s' restarted successfully.", name), false, nil

	case "close_session":
		name, _ := args["session_name"].(string)
		if name == "" {
			return "Error: 'session_name' is required", true, nil
		}
		err := sm.CloseSession(ctx, name)
		if err != nil {
			return fmt.Sprintf("Failed to close session: %v", err), true, nil
		}
		return fmt.Sprintf("Session '%s' closed.", name), false, nil

	default:
		return fmt.Sprintf("Unknown tool: %s", name), true, nil
	}
}
