# Jupyter-Bridge (Golang MCP Server & CLI for AI Agents)

A lightweight, zero-dependency, ultra-fast Golang tool and **Model Context Protocol (MCP)** server that connects local AI agents directly to a remote **Jupyter Notebook / JupyterLab Server** running on your VPS.

---

## Key Features

- **MCP Protocol Server**: Implements the Model Context Protocol (JSON-RPC 2.0 over stdio) for AI agents (Antigravity, Claude, Cursor, OpenCode).
- **Interactive Execution & Memory**: Runs Python/IPython code with persistent memory state across tool calls.
- **Multi-Session Support**: Manage multiple isolated kernels and terminal sessions simultaneously.
- **Shell & Terminal Execution**: Execute bash commands (`!pip install`, `df -h`, `nvidia-smi`, etc.) directly on the remote VPS.
- **Remote Filesystem**: Browse, read, write, and delete files/notebooks on the VPS.
- **Hardware & Resource Monitoring**: Inspect remote CPU, RAM, disk, OS, and GPU status.
- **Zero-Dependency & Dockerized**: Statically compiled single binary, minimal ~15MB Docker image.

---

## Exposed MCP Tools

| Tool Name | Parameters | Description |
| :--- | :--- | :--- |
| `execute_code` | `code`, `session_name`, `timeout_seconds` | Executes Python code in remote kernel; maintains RAM state. |
| `run_bash_command` | `command`, `session_name` | Runs a shell command inside the remote container. |
| `system_info` | _none_ | Returns CPU, RAM, disk space, and OS environment info. |
| `list_files` | `path` | Lists files and directories in remote workspace. |
| `read_file` | `path` | Reads contents of a file or `.ipynb` notebook. |
| `write_file` | `path`, `content` | Writes/updates a file or script on remote VPS. |
| `delete_file` | `path` | Deletes a remote file or folder. |
| `list_sessions` | _none_ | Lists active kernel sessions. |
| `create_session` | `session_name` | Creates a new named kernel session. |
| `restart_session` | `session_name` | Restarts kernel session to free RAM and reset state. |
| `close_session` | `session_name` | Shuts down a kernel session. |

---

## Configuration (`.env` or Environment Variables)

| Variable | Default | Description |
| :--- | :--- | :--- |
| `JUPYTER_SERVER_URL` | `https://jupyter-notebook.ufone-claim.site` | Base URL of your Jupyter Server |
| `JUPYTER_TOKEN` | `textdsfjsdfds` | Auth token for Jupyter API |
| `JUPYTER_KERNEL_NAME` | `python3` | Default kernel spec name |
| `USER_AGENT` | `Mozilla/5.0...` | Custom User-Agent header for proxy/Cloudflare bypass |

---

## CLI Usage (Standalone)

You can run the binary directly from your terminal:

```bash
# Check connection & status
./jupyter-bridge status

# Inspect remote hardware & RAM
./jupyter-bridge info

# Run Python code in kernel
./jupyter-bridge exec "import numpy as np; print(np.random.rand(3,3))"

# Run shell command on VPS
./jupyter-bridge bash "pip list"

# List files on remote VPS
./jupyter-bridge files work/

# List active sessions
./jupyter-bridge sessions
```

---

## Connecting to AI Agents (`mcp_config.json`)

### Option 1: Native Local Binary
```json
{
  "mcpServers": {
    "jupyter-vps": {
      "command": "/path/to/jupyter-bridge",
      "env": {
        "JUPYTER_SERVER_URL": "https://jupyter-notebook.ufone-claim.site",
        "JUPYTER_TOKEN": "textdsfjsdfds"
      }
    }
  }
}
```

### Option 2: Docker Container (Local or VPS)
```json
{
  "mcpServers": {
    "jupyter-vps": {
      "command": "docker",
      "args": [
        "run", "-i", "--rm",
        "-e", "JUPYTER_SERVER_URL=https://jupyter-notebook.ufone-claim.site",
        "-e", "JUPYTER_TOKEN=textdsfjsdfds",
        "jupyter-bridge"
      ]
    }
  }
}
```

---

## Building and Docker Deployment

### Local Build (Go):
```bash
# Build for current OS
go build -o jupyter-bridge .

# Cross-compile for Linux (amd64)
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o jupyter-bridge-linux .
```

### Docker Build & Run:
```bash
# Build Docker image
docker build -t jupyter-bridge .

# Run with Docker Compose
docker compose up -d
```
