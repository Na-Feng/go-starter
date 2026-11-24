# syntax=docker/dockerfile:1
FROM golang:1.22-alpine AS builder
WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /app/bin/server ./cmd/server

FROM alpine:3.20
RUN apk add --no-cache ca-certificates tzdata \
    && adduser -D -u 10001 appuser \
    && mkdir -p /app/uploads /app/configs \
    && chown -R appuser:appuser /app
WORKDIR /app

COPY --from=builder --chown=appuser:appuser /app/bin/server /app/server
COPY --from=builder --chown=appuser:appuser /app/configs/config.yaml /app/configs/config.yaml
USER appuser

ENV APP_ENV=prod LOG_FORMAT=json AUTO_MIGRATE=false
VOLUME ["/app/uploads"]
EXPOSE 8080
ENTRYPOINT ["/app/server"]
