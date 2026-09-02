package jupyter

import "time"

// Kernel represents an active Jupyter kernel
type Kernel struct {
	ID             string    `json:"id"`
	Name           string    `json:"name"`
	LastActivity   time.Time `json:"last_activity"`
	ExecutionState string    `json:"execution_state"`
	Connections    int       `json:"connections"`
}

// KernelSpec represents a kernel specification
type KernelSpec struct {
	Name        string `json:"name"`
	DisplayName string `json:"display_name"`
	Language    string `json:"language"`
}

// ContentItem represents a file, folder, or notebook in the Jupyter workspace
type ContentItem struct {
	Name         string        `json:"name"`
	Path         string        `json:"path"`
	Type         string        `json:"type"` // "file", "directory", "notebook"
	Writable     bool          `json:"writable"`
	Created      string        `json:"created"`
	LastModified string        `json:"last_modified"`
	Size         int64         `json:"size,omitempty"`
	Mimetype     string        `json:"mimetype,omitempty"`
	Format       string        `json:"format,omitempty"`
	Content      interface{}   `json:"content,omitempty"`
}

// Terminal represents an active PTY terminal session
type Terminal struct {
	Name string `json:"name"`
}

// Jupyter WS Message structures (ZeroMQ over WebSocket)
type MsgHeader struct {
	MsgID    string `json:"msg_id"`
	MsgType  string `json:"msg_type"`
	Username string `json:"username"`
	Session  string `json:"session"`
	Version  string `json:"version"`
	Date     string `json:"date,omitempty"`
}

type WSMessage struct {
	Header       MsgHeader              `json:"header"`
	ParentHeader map[string]interface{} `json:"parent_header"`
	Metadata     map[string]interface{} `json:"metadata"`
	Content      map[string]interface{} `json:"content"`
	Channel      string                 `json:"channel"`
}

type ExecutionResult struct {
	Stdout     string   `json:"stdout"`
	Stderr     string   `json:"stderr"`
	Result     string   `json:"result"`
	Errors     []string `json:"errors"`
	Success    bool     `json:"success"`
	ExecutionN int      `json:"execution_count,omitempty"`
}
