FROM golang:1.27.2-alpine AS builder

RUN apk add --no-cache ca-certificates

WORKDIR /build

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build \
    -trimpath \
    -ldflags="-s -w" \
    -o /build/server \
    ./cmd/server

FROM alpine:3.24

RUN apk upgrade --no-cache && apk add --no-cache ca-certificates tzdata

RUN addgroup -S appgroup && adduser -S appuser -G appgroup

WORKDIR /app

RUN mkdir -p /app/sounds /data/media && chown -R appuser:appgroup /app/sounds /data/media

COPY --from=builder --chown=appuser:appgroup /build/server .
COPY --from=builder --chown=appuser:appgroup /build/sounds /app/sounds-default

ENV SOUNDS_DIR=/app/sounds \
    SOUNDS_BUNDLED_DIR=/app/sounds-default

USER appuser

EXPOSE 8080

CMD ["./server"]
