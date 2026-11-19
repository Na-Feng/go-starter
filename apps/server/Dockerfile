# syntax=docker/dockerfile:1
FROM golang:1.22-alpine AS builder
WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /app/bin/server ./cmd/server

FROM alpine:3.20
RUN apk add --no-cache ca-certificates tzdata && adduser -D -u 10001 appuser
WORKDIR /app

COPY --from=builder /app/bin/server /app/server
USER appuser

EXPOSE 8080
ENTRYPOINT ["/app/server"]
