# Ollama Setup Guide

This guide explains how to run DAGO with Ollama running on your host machine.

## Overview

DAGO is configured to use **Ollama running on the host** by default. The Docker containers connect to Ollama via `host.docker.internal:11434`.

```
┌─────────────────────────────────────┐
│           HOST MACHINE              │
│                                     │
│  ┌───────────────┐                 │
│  │    Ollama     │  :11434         │
│  │  (ollama serve)                 │
│  └───────┬───────┘                 │
│          │                          │
│  ┌───────┴────────────────────┐    │
│  │    Docker Containers       │    │
│  │  ┌──────┐  ┌──────┐  ┌────┐│   │
│  │  │ dago │  │ exec │  │rout││   │
│  │  └──────┘  └──────┘  └────┘│   │
│  │      Connect via             │   │
│  │  host.docker.internal:11434 │   │
│  └─────────────────────────────┘   │
└─────────────────────────────────────┘
```

## Prerequisites

- Docker 20.10+
- Docker Compose 1.29+
- Ollama installed on your host machine

## Quick Start

### 1. Install Ollama on Host

**Linux:**
```bash
curl -fsSL https://ollama.com/install.sh | sh
```

**macOS:**
```bash
brew install ollama
```

**Windows:**
Download from [ollama.com](https://ollama.com/download)

### 2. Start Ollama Service

```bash
# Start Ollama server
ollama serve
```

**Note:** On macOS and Windows, Ollama usually runs as a background service after installation.

### 3. Pull a Model

In a new terminal:

```bash
# Pull default model
ollama pull llama3.1

# Or pull a smaller/faster model
ollama pull llama3.1:8b

# Or pull granite4 (very small, fast)
ollama pull granite4:micro
```

### 4. Verify Ollama is Running

```bash
# Test Ollama API
curl http://localhost:11434/api/tags

# List available models
ollama list
```

### 5. Start DAGO Stack

```bash
cd deployments
docker-compose up -d
```

### 6. Verify Connection

```bash
# Check DAGO health
curl http://localhost:8080/health

# Check worker logs to verify Ollama connection
docker logs dago-executor-worker
docker logs dago-router-worker
```

## Configuration

### Docker Compose Configuration

The docker-compose.yml is configured to connect to Ollama on the host:

```yaml
environment:
  - LLM_PROVIDER=ollama
  - LLM_BASE_URL=http://host.docker.internal:11434
  - LLM_MODEL=llama3.1
```

### Using a Different Model

**Option 1: Environment variable**
```bash
LLM_MODEL=granite4:micro docker-compose up -d
```

**Option 2: Edit .env file**
```bash
# In dago/.env
LLM_MODEL=mistral
```

**Option 3: Edit docker-compose.yml**
Change the default value in the environment section.

## Available Models

### Recommended Models

| Model | Size | Speed | Use Case |
|-------|------|-------|----------|
| `llama3.1:8b` | ~4.7GB | Fast | General purpose, good balance |
| `granite4:micro` | ~1.6GB | Very Fast | Lightweight, quick responses |
| `mistral` | ~4.1GB | Fast | Code and reasoning |
| `codellama` | ~3.8GB | Medium | Code-focused tasks |
| `llama3.1:70b` | ~40GB | Slow | High quality, requires powerful hardware |

### Pull a Model

```bash
ollama pull <model-name>

# Examples
ollama pull llama3.1:8b
ollama pull granite4:micro
ollama pull mistral
ollama pull codellama
```

## Troubleshooting

### Containers Can't Connect to Ollama

**Symptoms:**
- Workers fail with "connection refused" errors
- Logs show "failed to connect to http://host.docker.internal:11434"

**Solutions:**

1. **Verify Ollama is running:**
   ```bash
   curl http://localhost:11434/api/tags
   ```

2. **Check Ollama is listening on all interfaces:**
   ```bash
   # Linux - bind to all interfaces
   OLLAMA_HOST=0.0.0.0:11434 ollama serve
   ```

3. **On Linux, use host network mode (alternative):**
   ```yaml
   # In docker-compose.yml, add to services:
   network_mode: "host"
   ```
   Then use `LLM_BASE_URL=http://localhost:11434`

4. **On older Docker versions, use host IP:**
   ```bash
   # Find host IP
   ip addr show docker0 | grep inet

   # Use that IP in .env
   LLM_BASE_URL=http://172.17.0.1:11434
   ```

### Model Not Found

**Pull the model first:**
```bash
ollama pull llama3.1
```

**Verify model is available:**
```bash
ollama list
```

### Slow Performance

**Options:**

1. **Use a smaller model:**
   ```bash
   ollama pull llama3.1:8b
   # Update .env
   LLM_MODEL=llama3.1:8b
   ```

2. **Use GPU acceleration (if available):**
   - NVIDIA GPU: Ollama automatically uses CUDA
   - Apple Silicon: Ollama uses Metal
   - AMD GPU: Check [Ollama docs](https://github.com/ollama/ollama)

3. **Increase Ollama resources:**
   ```bash
   # Set context size (default 2048)
   OLLAMA_NUM_CTX=4096 ollama serve
   ```

### Out of Memory Errors

1. **Use a smaller model:**
   ```bash
   ollama pull granite4:micro
   ```

2. **Reduce concurrent requests:**
   ```bash
   # In docker-compose.yml
   environment:
     - WORKER_POOL_SIZE=1  # Reduce from default 5
   ```

3. **Set Ollama memory limit:**
   ```bash
   OLLAMA_MAX_LOADED_MODELS=1 ollama serve
   ```

## Advanced Configuration

### Custom Ollama Port

If Ollama runs on a different port:

```bash
# Start Ollama on custom port
OLLAMA_HOST=0.0.0.0:8080 ollama serve

# Update docker-compose.yml
environment:
  - LLM_BASE_URL=http://host.docker.internal:8080
```

### Ollama on Remote Host

To use Ollama running on a different machine:

```bash
# In .env or docker-compose.yml
LLM_BASE_URL=http://192.168.1.100:11434
```

### Multiple Models

Ollama can keep multiple models loaded. Configure in Ollama:

```bash
OLLAMA_MAX_LOADED_MODELS=2 ollama serve
```

Then switch models by changing `LLM_MODEL` environment variable.

### Ollama with GPU (NVIDIA)

Ollama automatically detects and uses NVIDIA GPUs:

```bash
# Verify GPU is detected
ollama run llama3.1 "test"
# Watch GPU usage
nvidia-smi -l 1
```

### Ollama as Systemd Service (Linux)

Create `/etc/systemd/system/ollama.service`:

```ini
[Unit]
Description=Ollama Service
After=network-online.target

[Service]
ExecStart=/usr/local/bin/ollama serve
User=ollama
Group=ollama
Restart=always
RestartSec=3
Environment="OLLAMA_HOST=0.0.0.0:11434"
Environment="OLLAMA_MODELS=/var/lib/ollama/models"

[Install]
WantedBy=multi-user.target
```

Enable and start:
```bash
sudo systemctl enable ollama
sudo systemctl start ollama
sudo systemctl status ollama
```

## Managing Models

### List Models
```bash
ollama list
```

### Remove a Model
```bash
ollama rm llama3.1
```

### Update a Model
```bash
ollama pull llama3.1  # Re-pull to update
```

### Model Information
```bash
ollama show llama3.1
```

### Model Storage Location

**Linux:** `/usr/share/ollama/.ollama/models`
**macOS:** `~/.ollama/models`
**Windows:** `C:\Users\<user>\.ollama\models`

## Switching to Cloud Providers

To switch from Ollama to Anthropic, OpenAI, or Gemini:

**Create/edit `.env`:**
```bash
# For Anthropic
LLM_PROVIDER=anthropic
LLM_API_KEY=sk-ant-your-api-key-here
LLM_MODEL=claude-sonnet-4-20250514

# For OpenAI
LLM_PROVIDER=openai
LLM_API_KEY=sk-your-openai-api-key-here
LLM_MODEL=gpt-4o

# For Gemini
LLM_PROVIDER=gemini
LLM_API_KEY=your-gemini-api-key-here
LLM_MODEL=gemini-2.0-flash-exp
```

**Restart services:**
```bash
docker-compose restart
```

## Performance Benchmarks

Approximate inference speed (tokens/second) on different hardware:

| Model | CPU (8-core) | GPU (RTX 3060) | Apple M2 |
|-------|--------------|----------------|----------|
| granite4:micro | 40-60 | 120-150 | 80-100 |
| llama3.1:8b | 15-25 | 80-120 | 50-70 |
| llama3.1:70b | 2-5 | 25-40 | 15-25 |

## References

- [Ollama Documentation](https://github.com/ollama/ollama)
- [Ollama Model Library](https://ollama.com/library)
- [Ollama API Reference](https://github.com/ollama/ollama/blob/main/docs/api.md)
