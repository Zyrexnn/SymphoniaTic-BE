# Stage 1: Build binary
FROM golang:alpine AS builder

WORKDIR /app

# Install build dependencies
RUN apk add --no-cache git

# Copy dependency files and download
COPY go.mod go.sum ./
RUN go mod download

# Copy application source code
COPY . .

# Build native binary
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o server ./cmd/server

# Stage 2: Production runtime image
FROM alpine:latest

WORKDIR /app

# Install runtime dependencies (certificates & timezone data)
RUN apk add --no-cache ca-certificates tzdata

# Create directory for file uploads
RUN mkdir -p /app/uploads

# Copy binary and setup SQL script
COPY --from=builder /app/server /app/server
COPY --from=builder /app/setup.sql /app/setup.sql

EXPOSE 8082

CMD ["/app/server"]
