# =============================================================================
# AySub Multi-Stage Dockerfile
# =============================================================================
# Stage 1: Build frontend
# Stage 2: Build Go backend with embedded frontend
# Stage 3: Final minimal image
# =============================================================================

ARG NODE_IMAGE=node:24-alpine
ARG GOLANG_IMAGE=golang:1.26.5-alpine
ARG ALPINE_IMAGE=alpine:3.20
ARG PNPM_VERSION=9.15.9
ARG GOPROXY=https://goproxy.cn,direct
ARG GOSUMDB=sum.golang.google.cn

# -----------------------------------------------------------------------------
# Stage 1: Frontend Builder
# -----------------------------------------------------------------------------
FROM ${NODE_IMAGE} AS frontend-builder

ARG PNPM_VERSION

WORKDIR /app/frontend

# Install the exact pnpm version used by this lockfile.
RUN corepack enable && corepack prepare pnpm@${PNPM_VERSION} --activate

# Install dependencies first (better caching)
COPY frontend/package.json frontend/pnpm-lock.yaml frontend/pnpm-workspace.yaml frontend/.npmrc ./
RUN pnpm install --frozen-lockfile

# Copy frontend source and build
COPY frontend/ ./
RUN pnpm run build

# -----------------------------------------------------------------------------
# Stage 2: Backend Builder
# -----------------------------------------------------------------------------
FROM ${GOLANG_IMAGE} AS backend-builder

# Build arguments for version info (set by CI)
ARG VERSION=docker
ARG COMMIT=docker
ARG DATE
ARG GOPROXY
ARG GOSUMDB

ENV GOPROXY=${GOPROXY}
ENV GOSUMDB=${GOSUMDB}

# Install build dependencies
RUN apk add --no-cache git ca-certificates tzdata

WORKDIR /app/backend

# Copy go mod files first (better caching)
COPY backend/go.mod backend/go.sum ./
RUN go mod download

# Copy backend source first
COPY backend/ ./

# Copy frontend dist from previous stage (must be after backend copy to avoid being overwritten)
COPY --from=frontend-builder /app/backend/internal/web/dist ./internal/web/dist

# Build the binary (BuildType=release for CI builds, embed frontend)
RUN CGO_ENABLED=0 GOOS=linux go build \
    -tags embed \
    -ldflags="-s -w -X main.Commit=${COMMIT} -X main.Date=${DATE:-$(date -u +%Y-%m-%dT%H:%M:%SZ)} -X main.BuildType=release" \
    -o /app/aysub \
    ./cmd/server

# -----------------------------------------------------------------------------
# Stage 3: Final Runtime Image
# -----------------------------------------------------------------------------
FROM ${ALPINE_IMAGE}

# Labels
LABEL maintainer="AIAllABOUTYOU <https://github.com/AIAllABOUTYOU>"
LABEL description="AySub - three-in-one AI API gateway"
LABEL org.opencontainers.image.source="https://github.com/AIAllABOUTYOU/AySub"

# Install runtime dependencies
RUN apk add --no-cache \
    ca-certificates \
    tzdata \
    curl \
    su-exec \
    && rm -rf /var/cache/apk/*

# Create non-root user
RUN addgroup -g 1000 aysub && \
    adduser -u 1000 -G aysub -s /bin/sh -D aysub

# Set working directory
WORKDIR /app

# Copy binary from builder
COPY --from=backend-builder /app/aysub /app/aysub

# Create data directory
RUN mkdir -p /app/data && chown -R aysub:aysub /app

# Copy entrypoint script (fixes volume permissions then drops to aysub)
COPY deploy/docker-entrypoint.sh /app/docker-entrypoint.sh
RUN chmod +x /app/docker-entrypoint.sh

# Expose port (can be overridden by SERVER_PORT env var)
EXPOSE 8080

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=10s --retries=3 \
    CMD wget -q -T 5 -O /dev/null http://localhost:${SERVER_PORT:-8080}/health || exit 1

# Run the application (entrypoint fixes /app/data ownership then execs as aysub)
ENTRYPOINT ["/app/docker-entrypoint.sh"]
CMD ["/app/aysub"]
