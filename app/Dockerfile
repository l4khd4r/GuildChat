# ---------- Build ----------
FROM golang:1.26-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build \
    -o guildchat \
    ./cmd/server

# ---------- Runtime ----------
FROM alpine:3.21

WORKDIR /app

RUN apk --no-cache add ca-certificates \
    && adduser -D -u 10001 appuser

COPY --from=builder /app/guildchat .

USER appuser

EXPOSE 8080

CMD ["./guildchat"]
