package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"sync"

	"github.com/google/uuid"
	"jupyter-bridge/pkg/jupyter"
)

type SSEServer struct {
	sessionManager *jupyter.SessionManager
	tools          []Tool
	apiKey         string
	clients        map[string]chan []byte
	mu             sync.RWMutex
}

func NewSSEServer(sm *jupyter.SessionManager, apiKey string) *SSEServer {
	return &SSEServer{
		sessionManager: sm,
		tools:          GetToolDefinitions(),
		apiKey:         apiKey,
		clients:        make(map[string]chan []byte),
	}
}

func (s *SSEServer) ServeHTTP(addr string) error {
	mux := http.NewServeMux()

	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	mux.HandleFunc("/sse", s.handleSSE)
	mux.HandleFunc("/message", s.handleMessage)

	log.Printf("Starting MCP SSE / HTTP Server on %s ...", addr)
	return http.ListenAndServe(addr, mux)
}

func (s *SSEServer) checkAuth(r *http.Request) bool {
	if s.apiKey == "" {
		return true
	}
	authHeader := r.Header.Get("Authorization")
	if authHeader == fmt.Sprintf("Bearer %s", s.apiKey) || authHeader == s.apiKey {
		return true
	}
	if r.URL.Query().Get("key") == s.apiKey {
		return true
	}
	return false
}

func (s *SSEServer) handleSSE(w http.ResponseWriter, r *http.Request) {
	if !s.checkAuth(r) {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming unsupported", http.StatusInternalServerError)
		return
	}

	sessionID := uuid.New().String()
	msgChan := make(chan []byte, 64)

	s.mu.Lock()
	s.clients[sessionID] = msgChan
	s.mu.Unlock()

	defer func() {
		s.mu.Lock()
		delete(s.clients, sessionID)
		close(msgChan)
		s.mu.Unlock()
	}()

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	// 1. Send endpoint event telling the agent where to send POST messages
	endpointURL := fmt.Sprintf("/message?sessionId=%s", sessionID)
	if s.apiKey != "" {
		endpointURL += fmt.Sprintf("&key=%s", s.apiKey)
	}
	fmt.Fprintf(w, "event: endpoint\ndata: %s\n\n", endpointURL)
	flusher.Flush()

	notify := r.Context().Done()
	for {
		select {
		case <-notify:
			return
		case msg, ok := <-msgChan:
			if !ok {
				return
			}
			fmt.Fprintf(w, "event: message\ndata: %s\n\n", string(msg))
			flusher.Flush()
		}
	}
}

func (s *SSEServer) handleMessage(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if !s.checkAuth(r) {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	sessionID := r.URL.Query().Get("sessionId")
	if sessionID == "" {
		http.Error(w, "Missing sessionId query parameter", http.StatusBadRequest)
		return
	}

	s.mu.RLock()
	_, exists := s.clients[sessionID]
	s.mu.RUnlock()

	if !exists {
		http.Error(w, "Session expired or invalid", http.StatusNotFound)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Failed to read body", http.StatusBadRequest)
		return
	}

	var req JSONRPCRequest
	if err := json.Unmarshal(body, &req); err != nil {
		http.Error(w, "Invalid JSON-RPC", http.StatusBadRequest)
		return
	}

	w.WriteHeader(http.StatusAccepted)
	w.Write([]byte("Accepted"))

	// Dispatch processing asynchronously
	go func() {
		server := NewServer(s.sessionManager)
		resp := server.handleRequest(context.Background(), &req)
		if resp != nil {
			respBytes, err := json.Marshal(resp)
			if err == nil {
				s.mu.RLock()
				ch, ok := s.clients[sessionID]
				if ok {
					ch <- respBytes
				}
				s.mu.RUnlock()
			}
		}
	}()
}
