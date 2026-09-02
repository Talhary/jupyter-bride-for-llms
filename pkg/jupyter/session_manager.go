package jupyter

import (
	"context"
	"fmt"
	"sync"
	"time"
)

type SessionInfo struct {
	Name      string    `json:"name"`
	KernelID  string    `json:"kernel_id"`
	CreatedAt time.Time `json:"created_at"`
}

type SessionManager struct {
	client   *Client
	mu       sync.RWMutex
	sessions map[string]*SessionInfo
}

func NewSessionManager(client *Client) *SessionManager {
	return &SessionManager{
		client:   client,
		sessions: make(map[string]*SessionInfo),
	}
}

// GetOrCreateSession returns a kernel ID for a session name (or default if empty)
func (sm *SessionManager) GetOrCreateSession(ctx context.Context, name string) (string, error) {
	if name == "" {
		name = "default"
	}

	sm.mu.RLock()
	s, exists := sm.sessions[name]
	sm.mu.RUnlock()

	if exists {
		// Verify kernel is still alive on remote server
		kernels, err := sm.client.ListKernels(ctx)
		if err == nil {
			for _, k := range kernels {
				if k.ID == s.KernelID {
					return s.KernelID, nil
				}
			}
		}
	}

	// Lock for creating/attaching
	sm.mu.Lock()
	defer sm.mu.Unlock()

	// Double check
	if s, exists := sm.sessions[name]; exists {
		kernels, err := sm.client.ListKernels(ctx)
		if err == nil {
			for _, k := range kernels {
				if k.ID == s.KernelID {
					return s.KernelID, nil
				}
			}
		}
	}

	// Check if there are active kernels on the server we can adopt
	kernels, err := sm.client.ListKernels(ctx)
	if err == nil && len(kernels) > 0 {
		// Check if any kernel is unassigned
		assigned := make(map[string]bool)
		for _, v := range sm.sessions {
			assigned[v.KernelID] = true
		}
		for _, k := range kernels {
			if !assigned[k.ID] {
				sm.sessions[name] = &SessionInfo{
					Name:      name,
					KernelID:  k.ID,
					CreatedAt: time.Now(),
				}
				return k.ID, nil
			}
		}
	}

	// Spawn new kernel
	newKernel, err := sm.client.CreateKernel(ctx, sm.client.cfg.KernelName)
	if err != nil {
		return "", fmt.Errorf("failed to create new Jupyter kernel: %w", err)
	}

	sm.sessions[name] = &SessionInfo{
		Name:      name,
		KernelID:  newKernel.ID,
		CreatedAt: time.Now(),
	}

	return newKernel.ID, nil
}

func (sm *SessionManager) ListSessions(ctx context.Context) ([]SessionInfo, error) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	kernels, _ := sm.client.ListKernels(ctx)
	aliveIDs := make(map[string]bool)
	for _, k := range kernels {
		aliveIDs[k.ID] = true
	}

	var res []SessionInfo
	for _, s := range sm.sessions {
		if aliveIDs[s.KernelID] {
			res = append(res, *s)
		}
	}
	return res, nil
}

func (sm *SessionManager) CloseSession(ctx context.Context, name string) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	s, exists := sm.sessions[name]
	if !exists {
		return fmt.Errorf("session '%s' not found", name)
	}

	err := sm.client.DeleteKernel(ctx, s.KernelID)
	delete(sm.sessions, name)
	return err
}

func (sm *SessionManager) RestartSession(ctx context.Context, name string) error {
	sm.mu.RLock()
	s, exists := sm.sessions[name]
	sm.mu.RUnlock()

	if !exists {
		return fmt.Errorf("session '%s' not found", name)
	}

	return sm.client.RestartKernel(ctx, s.KernelID)
}

func (sm *SessionManager) Client() *Client {
	return sm.client
}
