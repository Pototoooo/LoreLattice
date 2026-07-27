# 使用 uv 运行 LoreLattice MCP 服务器

> 更推荐使用`uv`来运行基于python的MCP服务。

## 1. 安装 uv

```bash
# macOS/Linux
curl -LsSf https://astral.sh/uv/install.sh | sh

# 或使用 Homebrew (macOS)
brew install uv

# Windows
powershell -ExecutionPolicy ByPass -c "irm https://astral.sh/uv/install.ps1 | iex"
```

## 2. MCP 客户端配置

### Claude Desktop 配置

在 Claude Desktop 设置中添加:

```json
{
  "mcpServers": {
    "lorelattice": {
      "args": [
        "--directory",
        "/path/LoreLattice/mcp-server",
        "run",
        "run_server.py"
      ],
      "command": "uv",
      "env": {
        "LORELATTICE_API_KEY": "your_api_key_here",
        "LORELATTICE_BASE_URL": "http://localhost:8080/api/v1"
      }
    }
  }
}
```

### Cursor 配置

在 Cursor 中，编辑 MCP 配置文件 (通常在 `~/.cursor/mcp-config.json`):

```json
{
  "mcpServers": {
    "lorelattice": {
      "command": "uv",
      "args": [
        "--directory",
        "/path/LoreLattice/mcp-server",
        "run",
        "run_server.py"
      ],
      "env": {
        "LORELATTICE_API_KEY": "your_api_key_here",
        "LORELATTICE_BASE_URL": "http://localhost:8080/api/v1"
      }
    }
  }
}
```

### KiloCode 配置

对于 KiloCode 或其他支持 MCP 的编辑器，配置如下:

```json
{
  "mcpServers": {
    "lorelattice": {
      "command": "uv",
      "args": [
        "--directory",
        "/path/LoreLattice/mcp-server",
        "run",
        "run_server.py"
      ],
      "env": {
        "LORELATTICE_API_KEY": "your_api_key_here",
        "LORELATTICE_BASE_URL": "http://localhost:8080/api/v1"
      }
    }
  }
}
```

### 其他 MCP 客户端

对于一般 MCP 客户端配置:

```json
{
  "mcpServers": {
    "lorelattice": {
      "command": "uv",
      "args": [
        "--directory",
        "/path/LoreLattice/mcp-server",
        "run",
        "run_server.py"
      ],
      "env": {
        "LORELATTICE_API_KEY": "your_api_key_here",
        "LORELATTICE_BASE_URL": "http://localhost:8080/api/v1"
      }
    }
  }
}
```
