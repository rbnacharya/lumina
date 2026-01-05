# Lumina

**A high-performance LLM Gateway and Observability platform**

Lumina is a unified proxy that sits between your applications and LLM providers, providing a single API interface, cost tracking, and full observability.

## Features

- **Unified API** - Single endpoint for OpenAI and Anthropic models
- **Virtual Keys** - Create access-controlled keys with budget limits and model restrictions
- **Cost Tracking** - Monitor spend per key with configurable budgets
- **Request Logging** - Full request/response logging with searchable history
- **Web Dashboard** - Manage keys, view logs, and monitor usage

## Quick Start

```bash
docker run -d \
  -p 8080:8080 \
  -p 3000:3000 \
  -e DATABASE_URL=postgres://user:pass@host:5432/lumina \
  -e REDIS_URL=redis://host:6379 \
  -e OPENSEARCH_URL=http://host:9200 \
  -e JWT_SECRET=your-secret-key \
  -e ENCRYPTION_KEY=your-32-byte-key \
  rbnacharya/lumina:latest
```

## Ports

| Port | Service |
|------|---------|
| 8080 | Gateway API |
| 3000 | Web Dashboard |

## Environment Variables

| Variable | Description | Required |
|----------|-------------|----------|
| `DATABASE_URL` | PostgreSQL connection string | Yes |
| `REDIS_URL` | Redis connection string | Yes |
| `OPENSEARCH_URL` | OpenSearch connection string | Yes |
| `JWT_SECRET` | Secret for JWT signing | Yes |
| `ENCRYPTION_KEY` | 32-byte key for encrypting API keys | Yes |
| `LOG_LEVEL` | Logging level (debug, info, warn, error) | No |

## Usage

1. Access the dashboard at `http://localhost:3000`
2. Register an account and add your provider API keys (OpenAI/Anthropic)
3. Create virtual keys with optional budget limits and model restrictions
4. Use virtual keys in your applications:

```bash
curl -X POST http://localhost:8080/v1/chat/completions \
  -H "Authorization: Bearer lum_your_virtual_key" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "openai/gpt-4o",
    "messages": [{"role": "user", "content": "Hello!"}]
  }'
```

## Infrastructure Requirements

- PostgreSQL 15+
- Redis 7+
- OpenSearch 2.x

## Links

- [GitHub Repository](https://github.com/rbnacharya/lumina)
- [Documentation](https://github.com/rbnacharya/lumina#readme)

## Support

For support, please:
- Create an issue on [GitHub](https://github.com/rbnacharya/lumina/issues)
- Contact me at rbnacharya@gmail.com

## License

MIT
