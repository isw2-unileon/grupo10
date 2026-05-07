# Stage 1: Build
FROM golang:1.23-alpine AS builder

RUN apk add --no-cache git build-base

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN go build -o /app/main ./src/main.go

# Stage 2: Final lightweight image
FROM alpine:latest

RUN adduser -D appuser
WORKDIR /home/appuser

RUN apk add --no-cache ca-certificates

COPY --from=builder /app/main .

# Copy migrations so the server can run them on startup
COPY --from=builder /app/migrations ./migrations

USER appuser

EXPOSE 8080

CMD ["./main"]
