# syntax=docker/dockerfile:1

# ==========================================
# Stage 1: Build Frontend (Vue 3 + Vite)
# Use Debian-based node:20-slim for robust glibc native binary support (esbuild/rollup)
# ==========================================
FROM node:20-slim AS frontend-builder
WORKDIR /app/frontend

RUN npm install -g pnpm@9

COPY frontend/package.json frontend/pnpm-lock.yaml* ./
RUN pnpm install --no-frozen-lockfile

COPY frontend/ ./
RUN pnpm run build

# ==========================================
# Stage 2: Build Backend (Go 1.26 + Embed)
# Use Debian-based golang:1.26-bookworm for reliable VCS & dependency resolution
# ==========================================
FROM golang:1.26-bookworm AS backend-builder
WORKDIR /app/backend

ENV CGO_ENABLED=0 \
    GOOS=linux \
    GOPROXY=https://proxy.golang.org,direct

COPY backend/go.mod backend/go.sum ./
RUN go mod download

COPY backend/ ./

# Copy built frontend assets to backend embed folder
COPY --from=frontend-builder /app/frontend/dist/ ./internal/webui/dist/

# Compile static binary
RUN go build -ldflags="-s -w" -o /app/sshtunnelhub ./cmd/server

# ==========================================
# Stage 3: Minimal Production Image
# Lightweight Alpine runner with static Go binary
# ==========================================
FROM alpine:3.20 AS runner

# Install runtime dependencies (certificates, timezone data, curl for healthcheck)
RUN apk add --no-cache ca-certificates tzdata curl

WORKDIR /app

# Create persistent data volume directory
RUN mkdir -p /data && chmod 777 /data
VOLUME ["/data"]

# Default environment configuration
ENV PORT=9090 \
    DATA_DIR=/data \
    GIN_MODE=release

# Copy integrated binary from backend-builder
COPY --from=backend-builder /app/sshtunnelhub /usr/local/bin/sshtunnelhub

EXPOSE 9090

HEALTHCHECK --interval=30s --timeout=5s --start-period=5s --retries=3 \
  CMD curl -f http://127.0.0.1:9090/api/dashboard/stats || exit 1

ENTRYPOINT ["/usr/local/bin/sshtunnelhub"]
CMD ["-port", "9090", "-data-dir", "/data"]
