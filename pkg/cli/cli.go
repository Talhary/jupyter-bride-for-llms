package cli

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"jupyter-bridge/pkg/jupyter"
	"jupyter-bridge/pkg/mcp"
)

func RunCLI(sm *jupyter.SessionManager, args []string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	if len(args) == 0 {
		printUsage()
		return nil
	}

	command := args[0]
	client := sm.Client()

	switch command {
	case "status":
		status, err := client.GetStatus(ctx)
		if err != nil {
			return fmt.Errorf("status check failed: %w", err)
		}
		fmt.Printf("Connected to Jupyter Server: %s\n", client.GetWSURL(""))
		for k, v := range status {
			fmt.Printf("  %s: %v\n", k, v)
		}

	case "info":
		out, _, err := mcp.ExecuteTool(ctx, sm, "system_info", nil)
		if err != nil {
			return err
		}
		fmt.Println(out)

	case "exec":
		if len(args) < 2 {
			return fmt.Errorf("usage: jupyter-bridge exec \"<python_code>\"")
		}
		code := strings.Join(args[1:], " ")
		out, isErr, err := mcp.ExecuteTool(ctx, sm, "execute_code", map[string]interface{}{
			"code": code,
		})
		if err != nil {
			return err
		}
		fmt.Println(out)
		if isErr {
			os.Exit(1)
		}

	case "bash", "sh":
		if len(args) < 2 {
			return fmt.Errorf("usage: jupyter-bridge bash \"<shell_command>\"")
		}
		cmd := strings.Join(args[1:], " ")
		out, isErr, err := mcp.ExecuteTool(ctx, sm, "run_bash_command", map[string]interface{}{
			"command": cmd,
		})
		if err != nil {
			return err
		}
		fmt.Println(out)
		if isErr {
			os.Exit(1)
		}

	case "files", "ls":
		path := ""
		if len(args) >= 2 {
			path = args[1]
		}
		out, _, err := mcp.ExecuteTool(ctx, sm, "list_files", map[string]interface{}{
			"path": path,
		})
		if err != nil {
			return err
		}
		fmt.Println(out)

	case "sessions":
		out, _, err := mcp.ExecuteTool(ctx, sm, "list_sessions", nil)
		if err != nil {
			return err
		}
		fmt.Println(out)

	default:
		printUsage()
	}

	return nil
}

func printUsage() {
	fmt.Println(`jupyter-bridge - High-Performance Golang Bridge for Jupyter & AI Agents

Usage:
  jupyter-bridge                 Run as MCP Protocol Server (stdio mode for AI agents)
  jupyter-bridge exec "<code>"   Execute Python code in remote kernel
  jupyter-bridge bash "<cmd>"    Run bash command in remote container
  jupyter-bridge info            Display remote system/hardware info
  jupyter-bridge files [path]    List remote workspace directory
  jupyter-bridge sessions        List active kernel sessions
  jupyter-bridge status          Check remote Jupyter connection status`)
}
