# Combined Dockerfile for Lumina (Frontend + Backend)
# Multi-stage build for minimal final image

# Stage 1: Build Go backend
FROM golang:1.24-alpine AS backend-builder

WORKDIR /app

# Install build dependencies
RUN apk add --no-cache git

# Copy Go workspace files
COPY go.work go.work.sum ./
COPY apps/gateway/go.mod apps/gateway/go.sum ./apps/gateway/

# Download dependencies
WORKDIR /app/apps/gateway
RUN go mod download

# Copy source code
COPY apps/gateway/ ./

# Build binary
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o /gateway ./cmd/gateway

# Stage 2: Build Next.js frontend
FROM node:20-alpine AS frontend-builder

WORKDIR /app

# Copy package files
COPY apps/web/package.json apps/web/package-lock.json ./

# Install dependencies
RUN npm ci

# Copy source code
COPY apps/web/ ./

# Build Next.js app
ENV NEXT_TELEMETRY_DISABLED=1
RUN npm run build

# Stage 3: Final runtime image
FROM alpine:3.19

WORKDIR /app

# Install runtime dependencies
RUN apk add --no-cache ca-certificates nodejs npm supervisor

# Copy Go binary
COPY --from=backend-builder /gateway /app/gateway

# Copy Next.js standalone build
COPY --from=frontend-builder /app/.next/standalone /app/web
COPY --from=frontend-builder /app/.next/static /app/web/.next/static
COPY --from=frontend-builder /app/public /app/web/public

# Copy supervisor config
COPY <<EOF /etc/supervisor/conf.d/supervisord.conf
[supervisord]
nodaemon=true
user=root

[program:gateway]
command=/app/gateway
autostart=true
autorestart=true
stderr_logfile=/dev/stderr
stderr_logfile_maxbytes=0
stdout_logfile=/dev/stdout
stdout_logfile_maxbytes=0

[program:web]
command=node /app/web/server.js
directory=/app/web
autostart=true
autorestart=true
stderr_logfile=/dev/stderr
stderr_logfile_maxbytes=0
stdout_logfile=/dev/stdout
stdout_logfile_maxbytes=0
environment=PORT="3000",HOSTNAME="0.0.0.0"
EOF

# Expose ports
EXPOSE 8080 3000

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
  CMD wget -q --spider http://localhost:8080/health || exit 1

# Run supervisor
CMD ["supervisord", "-c", "/etc/supervisor/conf.d/supervisord.conf"]
