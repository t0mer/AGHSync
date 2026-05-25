# Stage 1: build frontend
FROM node:20-alpine AS frontend
WORKDIR /app/web
COPY web/package*.json ./
RUN npm ci --silent
COPY web/ ./
RUN npm run build

# Stage 2: build Go binary (with embedded frontend)
FROM golang:1.25-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
COPY --from=frontend /app/web/dist ./internal/webui/dist
ARG VERSION=docker
RUN CGO_ENABLED=0 go build \
    -ldflags="-s -w -X github.com/t0mer/aghsync/cmd/aghsync.version=${VERSION}" \
    -o aghsync ./cmd/aghsync/

# Stage 3: minimal runtime image
FROM alpine:3.19
RUN apk add --no-cache ca-certificates tzdata
WORKDIR /app
COPY --from=builder /app/aghsync ./
EXPOSE 8080
ENTRYPOINT ["./aghsync"]
