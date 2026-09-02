package jupyter

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"jupyter-bridge/pkg/config"
)

type Client struct {
	cfg        *config.Config
	httpClient *http.Client
}

func NewClient(cfg *config.Config) *Client {
	return &Client{
		cfg: cfg,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

func (c *Client) newRequest(ctx context.Context, method, path string, body interface{}) (*http.Request, error) {
	url := fmt.Sprintf("%s%s", c.cfg.JupyterURL, path)
	var bodyReader io.Reader
	if body != nil {
		jsonBytes, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		bodyReader = bytes.NewReader(jsonBytes)
	}

	req, err := http.NewRequestWithContext(ctx, method, url, bodyReader)
	if err != nil {
		return nil, err
	}

	req.Header.Set("User-Agent", c.cfg.UserAgent)
	req.Header.Set("Authorization", fmt.Sprintf("token %s", c.cfg.JupyterToken))
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	return req, nil
}

func (c *Client) do(req *http.Request, target interface{}) error {
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("jupyter API error [%d]: %s", resp.StatusCode, string(bodyBytes))
	}

	if target != nil && len(bodyBytes) > 0 {
		return json.Unmarshal(bodyBytes, target)
	}
	return nil
}

// REST API Methods

func (c *Client) GetStatus(ctx context.Context) (map[string]interface{}, error) {
	req, err := c.newRequest(ctx, "GET", "/api/status", nil)
	if err != nil {
		return nil, err
	}
	var res map[string]interface{}
	return res, c.do(req, &res)
}

func (c *Client) ListKernels(ctx context.Context) ([]Kernel, error) {
	req, err := c.newRequest(ctx, "GET", "/api/kernels", nil)
	if err != nil {
		return nil, err
	}
	var kernels []Kernel
	return kernels, c.do(req, &kernels)
}

func (c *Client) CreateKernel(ctx context.Context, name string) (*Kernel, error) {
	if name == "" {
		name = c.cfg.KernelName
	}
	payload := map[string]string{"name": name}
	req, err := c.newRequest(ctx, "POST", "/api/kernels", payload)
	if err != nil {
		return nil, err
	}
	var kernel Kernel
	return &kernel, c.do(req, &kernel)
}

func (c *Client) DeleteKernel(ctx context.Context, id string) error {
	req, err := c.newRequest(ctx, "DELETE", fmt.Sprintf("/api/kernels/%s", id), nil)
	if err != nil {
		return err
	}
	return c.do(req, nil)
}

func (c *Client) RestartKernel(ctx context.Context, id string) error {
	req, err := c.newRequest(ctx, "POST", fmt.Sprintf("/api/kernels/%s/restart", id), nil)
	if err != nil {
		return err
	}
	return c.do(req, nil)
}

func (c *Client) InterruptKernel(ctx context.Context, id string) error {
	req, err := c.newRequest(ctx, "POST", fmt.Sprintf("/api/kernels/%s/interrupt", id), nil)
	if err != nil {
		return err
	}
	return c.do(req, nil)
}

func (c *Client) ListContents(ctx context.Context, path string) (*ContentItem, error) {
	cleanPath := strings.Trim(path, "/")
	req, err := c.newRequest(ctx, "GET", fmt.Sprintf("/api/contents/%s", cleanPath), nil)
	if err != nil {
		return nil, err
	}
	var item ContentItem
	return &item, c.do(req, &item)
}

func (c *Client) SaveFile(ctx context.Context, path, content, fileType string) error {
	cleanPath := strings.Trim(path, "/")
	if fileType == "" {
		fileType = "file"
	}
	payload := map[string]interface{}{
		"type":    fileType,
		"format":  "text",
		"content": content,
	}
	req, err := c.newRequest(ctx, "PUT", fmt.Sprintf("/api/contents/%s", cleanPath), payload)
	if err != nil {
		return err
	}
	return c.do(req, nil)
}

func (c *Client) DeleteFile(ctx context.Context, path string) error {
	cleanPath := strings.Trim(path, "/")
	req, err := c.newRequest(ctx, "DELETE", fmt.Sprintf("/api/contents/%s", cleanPath), nil)
	if err != nil {
		return err
	}
	return c.do(req, nil)
}

func (c *Client) ListTerminals(ctx context.Context) ([]Terminal, error) {
	req, err := c.newRequest(ctx, "GET", "/api/terminals", nil)
	if err != nil {
		return nil, err
	}
	var terminals []Terminal
	return terminals, c.do(req, &terminals)
}

func (c *Client) CreateTerminal(ctx context.Context) (*Terminal, error) {
	req, err := c.newRequest(ctx, "POST", "/api/terminals", nil)
	if err != nil {
		return nil, err
	}
	var term Terminal
	return &term, c.do(req, &term)
}

func (c *Client) DeleteTerminal(ctx context.Context, name string) error {
	req, err := c.newRequest(ctx, "DELETE", fmt.Sprintf("/api/terminals/%s", name), nil)
	if err != nil {
		return err
	}
	return c.do(req, nil)
}
