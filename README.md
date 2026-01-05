# Lumina

A high-performance LLM Gateway and Observability platform.

Lumina is a proxy that sits between your client applications and LLM providers (OpenAI, Anthropic), providing:
- **Single API Interface** - Manage multiple models through one endpoint
- **Cost Tracking** - Monitor spend per virtual key with budget limits
- **Observability** - Full request/response logging with searchable history
- **Reliability** - Rate limiting and request throttling

## Architecture

```
┌─────────────┐     ┌─────────────┐     ┌─────────────────┐
│   Client    │────▶│   Lumina    │────▶│  LLM Provider   │
│ Application │◀────│   Gateway   │◀────│ (OpenAI/Claude) │
└─────────────┘     └─────────────┘     └─────────────────┘
                           │
              ┌────────────┼────────────┐
              ▼            ▼            ▼
        ┌──────────┐ ┌──────────┐ ┌────────────┐
        │ Postgres │ │  Redis   │ │ OpenSearch │
        └──────────┘ └──────────┘ └────────────┘
```

## Quick Start

### Prerequisites

- [Docker](https://docs.docker.com/get-docker/) & Docker Compose
- [Tilt](https://docs.tilt.dev/install.html) (for development)
- [Task](https://taskfile.dev/installation/) (task runner)
- [Go 1.22+](https://golang.org/dl/) (for local development)
- [Node.js 20+](https://nodejs.org/) (for frontend development)

### Development

1. **Start all services with Tilt:**
   ```bash
   task start-dev
   ```

2. **Or use Docker Compose directly:**
   ```bash
   task dev
   ```

3. **Access the services:**
   - Gateway API: http://localhost:8080
   - Web Dashboard: http://localhost:3000
   - OpenSearch Dashboards: http://localhost:5601

### Available Commands

```bash
# Development
task start-dev      # Start with Tilt (recommended)
task stop-dev       # Stop Tilt
task dev            # Start with Docker Compose
task dev-down       # Stop Docker Compose
task logs           # View service logs

# Database
task migrate        # Run migrations
task migrate-down   # Rollback migrations
task migrate-create -- <name>  # Create new migration

# Testing
task test           # Run all tests
task test-backend   # Run Go tests
task test-frontend  # Run frontend tests

# Building
task build          # Build all production images
task build-gateway  # Build Go binary
task build-web      # Build Next.js

# Production
task prod-up        # Start production services
task prod-down      # Stop production services

# Utilities
task lint           # Run linters
task fmt            # Format code
task clean          # Clean build artifacts
task health         # Check service health
```

## Project Structure

```
lumina/
├── apps/
│   ├── gateway/          # Go backend (API proxy)
│   └── web/              # Next.js frontend (Dashboard)
├── deployments/
│   ├── docker/           # Docker Compose files
│   └── k8s/              # Kubernetes manifests
├── scripts/              # Utility scripts
├── Taskfile.yml          # Task runner configuration
├── Tiltfile              # Tilt development configuration
└── go.work               # Go workspace
```

## Configuration

Environment variables for the gateway:

| Variable | Description | Default |
|----------|-------------|---------|
| `PORT` | Gateway HTTP port | `8080` |
| `DATABASE_URL` | PostgreSQL connection string | - |
| `REDIS_URL` | Redis connection string | - |
| `OPENSEARCH_URL` | OpenSearch connection string | - |
| `JWT_SECRET` | Secret for JWT signing | - |
| `ENCRYPTION_KEY` | Key for encrypting API keys | - |
| `LOG_LEVEL` | Logging level | `info` |

## API Usage

### Using Virtual Keys

1. Create a virtual key in the dashboard
2. Use it in your LLM client:

```bash
curl -X POST http://localhost:8080/v1/chat/completions \
  -H "Authorization: Bearer lum_your_virtual_key" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "gpt-4",
    "messages": [{"role": "user", "content": "Hello!"}]
  }'
```

The gateway will:
1. Validate your virtual key
2. Look up the associated provider and real API key
3. Forward the request to the provider
4. Log the request/response to OpenSearch
5. Track token usage and costs

## MVP Scope

- **Supported Providers:** OpenAI (Chat Completions), Anthropic (Messages API)
- **Features:** Key management, cost tracking, request logging, rate limiting
- **Infrastructure:** PostgreSQL, Redis, OpenSearch

## Upcoming Features

- [ ] **Model Pricing Configuration** - Admin interface to upload and manage per-model pricing (initially via JSON config files, later via dashboard)
- [ ] **Google Gemini Support** - Add Gemini models as a supported provider
- [ ] **Image Generation** - Support for DALL-E, Stable Diffusion, and other image generation APIs
- [ ] **Accurate Cost Calculation** - Real-time cost tracking based on configurable model pricing

## Support

For support, please:
- Create an issue on [GitHub](https://github.com/rbnacharya/lumina/issues)
- Contact me at rbnacharya@gmail.com

## License

MIT
