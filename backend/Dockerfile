# Stage 1: Build
# Pinned patch for the stdlib vulnerability fixes in the 1.26 line.
FROM golang:1.26.5-alpine AS builder

RUN apk add --no-cache git ca-certificates

WORKDIR /build

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -o /build/server ./cmd/server

# Stage 2: Runtime
# Alpine 3.22 + apk upgrade keeps OS packages patched (musl CVE-2026-40200 etc.).
FROM alpine:3.22

RUN apk upgrade --no-cache && apk add --no-cache ca-certificates tzdata

WORKDIR /app

COPY --from=builder /build/server .

EXPOSE 8080

CMD ["./server"]
