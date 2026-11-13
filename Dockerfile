# Stage 1: Build Go application
FROM golang:1.25-alpine AS builder

WORKDIR /app

# Install certificates and timezone data
RUN apk add --no-cache ca-certificates tzdata

# Copy dependency files
COPY go.mod go.sum ./

# Download Go dependencies
RUN go mod download

# Copy project source
COPY . .

# Compile Go application
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -trimpath -ldflags="-s -w" \
    -o /server ./cmd/server

# Stage 2: Runtime
FROM alpine:3.22

WORKDIR /app

RUN apk add --no-cache ca-certificates tzdata \
    && adduser -D -u 10001 appuser \
    && mkdir -p /app/uploads \
    && chown -R appuser:appuser /app

# Copy compiled binary
COPY --from=builder --chown=appuser:appuser /server ./server

USER appuser

EXPOSE 8080

CMD ["./server"]
