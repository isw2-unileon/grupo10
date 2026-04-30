# Stage 1: Build stage
FROM golang:1.23-alpine AS builder

# Install necessary build tools
RUN apk add --no-cache git build-base

WORKDIR /app

# Copy go mod and sum files
COPY go.mod go.sum ./
RUN go mod download

# Copy the rest of the source code
COPY . .

# Build the application
# We assume your entry point is src/main.go
RUN go build -o /app/main ./src/main.go

# Stage 2: Final lightweight image
FROM alpine:latest

# Security: Add a non-root user
RUN adduser -D appuser
WORKDIR /home/appuser

# Install CA certificates for external API calls (like AI integration)
RUN apk add --no-cache ca-certificates

# Copy only the binary from the builder
COPY --from=builder /app/main .

# Use the non-root user
USER appuser

EXPOSE 8080

CMD ["./main"]