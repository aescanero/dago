# DAGO Deployment Guide

This directory contains deployment configurations for DAGO (DA Orchestrator).

## Quick Start with Docker Compose

### Prerequisites

- Docker 20.10+
- Docker Compose 1.29+
- **Ollama installed on your host machine** (default LLM provider)

### Basic Setup

1. **Install and start Ollama on host:**
   ```bash
   # Linux
   curl -fsSL https://ollama.com/install.sh | sh
   ollama serve

   # macOS
   brew install ollama
   # Ollama runs as background service

   # Windows - Download from ollama.com
   ```

2. **Pull an Ollama model:**
   ```bash
   # In a new terminal
   ollama pull llama3.1
   # Or use a smaller/faster model
   ollama pull granite4:micro
   ```

3. **Verify Ollama is running:**
   ```bash
   curl http://localhost:11434/api/tags
   ollama list
   ```

4. **Navigate to deployments directory:**
   ```bash
   cd deployments
   ```

5. **Start the DAGO stack:**
   ```bash
   docker-compose up -d
   ```

6. **Verify everything is running:**
   ```bash
   # Check service health
   docker-compose ps

   # Test DAGO API
   curl http://localhost:8080/health

   # Check workers can connect to Ollama
   docker logs dago-executor-worker
   ```

## Default Configuration

The docker-compose setup uses **Ollama running on the host** as the default LLM provider:

- **LLM Provider:** Ollama (local on host, no API key required)
- **Default Model:** llama3.1
- **Ollama Endpoint:** http://host.docker.internal:11434 (from containers)
- **Ollama Endpoint:** http://localhost:11434 (from host)

### Services

| Service | Port | Description | Location |
|---------|------|-------------|----------|
| **ollama** | **11434** | **Local LLM service** | **Host machine** |
| dago | 8080 | Main orchestrator HTTP API | Container |
| dago | 9090 | WebSocket API | Container |
| dago | 2112 | Prometheus metrics | Container |
| redis | 6379 | State storage and event bus | Container |
| executor-worker | 8081 | Executor worker (health check) | Container |
| router-worker | 8082 | Router worker (health check) | Container |

## Switching LLM Providers

### Using Anthropic (Claude)

1. **Create/edit `.env` file:**
   ```bash
   LLM_PROVIDER=anthropic
   LLM_API_KEY=sk-ant-your-api-key-here
   LLM_MODEL=claude-sonnet-4-20250514
   ```

2. **Restart workers:**
   ```bash
   docker-compose restart executor-worker router-worker dago
   ```

### Using OpenAI (GPT)

1. **Create/edit `.env` file:**
   ```bash
   LLM_PROVIDER=openai
   LLM_API_KEY=sk-your-openai-api-key-here
   LLM_MODEL=gpt-4o
   ```

2. **Restart workers:**
   ```bash
   docker-compose restart executor-worker router-worker dago
   ```

### Using Google Gemini

1. **Create/edit `.env` file:**
   ```bash
   LLM_PROVIDER=gemini
   LLM_API_KEY=your-gemini-api-key-here
   LLM_MODEL=gemini-2.0-flash-exp
   ```

2. **Restart workers:**
   ```bash
   docker-compose restart executor-worker router-worker dago
   ```

## Detailed Ollama Setup

For detailed instructions on using Ollama, including GPU setup, model management, and troubleshooting, see:

**📖 [OLLAMA_SETUP.md](./OLLAMA_SETUP.md)**

Topics covered:
- GPU setup (NVIDIA)
- Model management
- Performance tuning
- Troubleshooting
- Data persistence

## Docker Compose Files

### `docker-compose.yml`
Default configuration. Connects to Ollama running on the host machine via `host.docker.internal:11434`.

**Note:** Ollama runs on the host, not in a container. GPU acceleration is handled by Ollama on the host.

## Environment Variables

All services support the following LLM-related environment variables:

| Variable | Description | Default | Required |
|----------|-------------|---------|----------|
| `LLM_PROVIDER` | LLM provider (anthropic, openai, gemini, ollama) | `ollama` | Yes |
| `LLM_API_KEY` | API key for cloud providers | - | No (for Ollama) |
| `LLM_BASE_URL` | Ollama endpoint or OpenAI-compatible API URL | `http://ollama:11434` | No |
| `LLM_MODEL` | Model name to use | `llama3.1` | Yes |
| `LOG_LEVEL` | Logging level (debug, info, warn, error) | `info` | No |

### Additional Environment Variables

**Redis:**
- `REDIS_ADDR` - Redis address (default: `redis:6379`)
- `REDIS_PASS` - Redis password (optional)
- `REDIS_DB` - Redis database number (default: `0`)

**Workers:**
- `WORKER_ID` - Unique worker identifier
- `MAX_ITERATIONS` - Max agent iterations (executor only)
- `WORKER_POOL_SIZE` - Worker pool size (dago core)

## Kubernetes Deployment

For production Kubernetes deployments, see the Helm charts:

```bash
cd helm/dago
helm install dago . -f values.yaml
```

See [helm/dago/README.md](./helm/dago/README.md) for details.

## Monitoring

### Prometheus Metrics

DAGO exposes Prometheus metrics on port 2112:

```bash
curl http://localhost:2112/metrics
```

### Health Checks

Each service has a health endpoint:

```bash
# Core orchestrator
curl http://localhost:8080/health

# Executor worker
curl http://localhost:8081/health

# Router worker
curl http://localhost:8082/health
```

### Logs

View service logs:

```bash
# All services
docker-compose logs -f

# Specific service
docker-compose logs -f dago
docker-compose logs -f executor-worker
docker-compose logs -f router-worker
docker-compose logs -f ollama
```

## Troubleshooting

### Services won't start

```bash
# Check service status
docker-compose ps

# Check logs for errors
docker-compose logs

# Restart services
docker-compose restart
```

### Ollama issues

See [OLLAMA_SETUP.md](./OLLAMA_SETUP.md#troubleshooting) for detailed Ollama troubleshooting.

### Worker connection issues

**Check Redis connectivity:**
```bash
docker exec dago-redis redis-cli ping
```

**Verify network:**
```bash
docker network inspect dago_dago-network
```

### Performance issues

**With Ollama:**
- Use GPU acceleration (requires nvidia-docker)
- Use smaller models (e.g., `llama3.1:8b`)
- Increase worker resources

**With cloud providers:**
- Check API rate limits
- Monitor API latency
- Consider using faster models

## Data Persistence

### Docker Volumes

Docker volumes are used for containerized data persistence:

- `redis-data` - Redis data (state, events)

**Backup Redis volume:**
```bash
docker run --rm -v dago_redis-data:/data -v $(pwd):/backup alpine \
  tar czf /backup/redis-backup.tar.gz -C /data .
```

### Ollama Models (Host)

Ollama models are stored on the **host machine**:

- **Linux:** `/usr/share/ollama/.ollama/models`
- **macOS:** `~/.ollama/models`
- **Windows:** `C:\Users\<user>\.ollama\models`

**Backup Ollama models:**
```bash
# Linux/macOS
tar czf ollama-models-backup.tar.gz -C ~/.ollama/models .

# Or use Ollama's built-in commands
ollama list  # See all models
ollama pull <model>  # Re-download if needed
```

## Cleaning Up

**Stop services:**
```bash
docker-compose down
```

**Remove volumes (WARNING: deletes all data):**
```bash
docker-compose down -v
```

**Remove images:**
```bash
docker-compose down --rmi all
```

## Development Mode

For local development with live reload:

```bash
# In each service directory (dago, dago-node-executor, dago-node-router)
make dev
```

This requires [air](https://github.com/cosmtrek/air) for live reloading:
```bash
go install github.com/cosmtrek/air@latest
```

## Support

- **Documentation:** [/docs](../../docs/)
- **Issues:** https://github.com/aescanero/dago/issues
- **Architecture:** [CLAUDE.md](../../CLAUDE.md)
