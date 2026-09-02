package config

import (
	"os"
	"strings"

	"github.com/joho/godotenv"
)

type Config struct {
	JupyterURL      string
	JupyterToken    string
	KernelName      string
	UserAgent       string
	HTTPPort        string
	BridgeAPIKey    string
	Transport       string
	AutoSpawnKernel bool
}

func Load() *Config {
	_ = godotenv.Load(".env")

	url := os.Getenv("JUPYTER_SERVER_URL")
	if url == "" {
		url = os.Getenv("JUPYTER_URL")
	}
	if url == "" {
		url = "https://jupyter-notebook.ufone-claim.site"
	}
	url = strings.TrimRight(url, "/")

	token := os.Getenv("JUPYTER_TOKEN")
	if token == "" {
		token = os.Getenv("JUPYTER_AUTH_KEY")
	}
	if token == "" {
		token = "textdsfjsdfds"
	}

	kernel := os.Getenv("JUPYTER_KERNEL_NAME")
	if kernel == "" {
		kernel = "python3"
	}

	userAgent := os.Getenv("USER_AGENT")
	if userAgent == "" {
		userAgent = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36"
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = os.Getenv("BRIDGE_PORT")
	}
	if port == "" {
		port = "8080"
	}

	apiKey := os.Getenv("BRIDGE_API_KEY")

	transport := strings.ToLower(os.Getenv("MCP_TRANSPORT"))
	if transport == "" {
		transport = "stdio"
	}

	return &Config{
		JupyterURL:      url,
		JupyterToken:    token,
		KernelName:      kernel,
		UserAgent:       userAgent,
		HTTPPort:        port,
		BridgeAPIKey:    apiKey,
		Transport:       transport,
		AutoSpawnKernel: true,
	}
}
