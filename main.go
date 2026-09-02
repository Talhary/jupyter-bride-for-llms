package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"jupyter-bridge/pkg/cli"
	"jupyter-bridge/pkg/config"
	"jupyter-bridge/pkg/jupyter"
	"jupyter-bridge/pkg/mcp"
)

func main() {
	mcpFlag := flag.Bool("mcp", false, "Run as MCP stdio server")
	httpFlag := flag.Bool("http", false, "Run as remote MCP SSE / HTTP server")
	portFlag := flag.String("port", "", "Port for HTTP SSE server (overrides env)")
	flag.Parse()

	cfg := config.Load()
	if *portFlag != "" {
		cfg.HTTPPort = *portFlag
	}

	client := jupyter.NewClient(cfg)
	sessionManager := jupyter.NewSessionManager(client)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle graceful shutdown signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigChan
		cancel()
	}()

	args := flag.Args()

	// 1. If explicit CLI command provided, run CLI
	if len(args) > 0 && !*mcpFlag && !*httpFlag {
		if err := cli.RunCLI(sessionManager, args); err != nil {
			log.Fatalf("Error: %v\n", err)
		}
		return
	}

	// 2. If HTTP/SSE mode requested via flag or env
	if *httpFlag || cfg.Transport == "sse" || cfg.Transport == "http" {
		addr := fmt.Sprintf("0.0.0.0:%s", cfg.HTTPPort)
		sseServer := mcp.NewSSEServer(sessionManager, cfg.BridgeAPIKey)
		if err := sseServer.ServeHTTP(addr); err != nil {
			log.Fatalf("HTTP server error: %v\n", err)
		}
		return
	}

	// 3. Default: MCP JSON-RPC Server over stdio
	server := mcp.NewServer(sessionManager)
	if err := server.RunStdio(ctx); err != nil && err != context.Canceled {
		fmt.Fprintf(os.Stderr, "MCP server error: %v\n", err)
		os.Exit(1)
	}
}
