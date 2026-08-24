# ---------- Build ----------
FROM golang:1.26-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build \
    -o guildchat \
    ./cmd/server

# shipped alongside the server so rollbacks and dirty-schema recovery can be
# run inside the container (`docker compose exec backend ./migrate version`)
RUN CGO_ENABLED=0 GOOS=linux go build \
    -o migrate \
    ./cmd/migrate

# ---------- Runtime ----------
FROM alpine:3.21

WORKDIR /app

RUN apk --no-cache add ca-certificates \
    && adduser -D -u 10001 appuser

COPY --from=builder /app/guildchat .
COPY --from=builder /app/migrate .

USER appuser

EXPOSE 8080

CMD ["./guildchat"]
