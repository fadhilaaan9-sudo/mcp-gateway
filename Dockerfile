# Dockerfile gateway + bot Telegram.
# Polanya multi-stage: tahap pertama pakai image Go untuk compile,
# tahap akhir hanya membawa binary hasil compile + sertifikat.
# Image Go-nya dibuang, jadi image akhir sangat ringan (scratch).
# Ada 2 target: gateway (default) dan bot.
#   docker build -t ghcr.io/farhanfadhila/mcp-gateway:latest .
#   docker build --target bot -t ghcr.io/farhanfadhila/mcp-gateway-bot:latest .

# Tahap 1: bangun program (build).
FROM golang:1.25-alpine AS builder

WORKDIR /app

RUN apk add --no-cache ca-certificates

# Salin go.mod dan go.sum dulu supaya download library
# bisa di-cache kalau kode berubah tapi library tidak.
COPY go.mod go.sum ./
RUN go mod download

# Salin semua kode, lalu compile 2 program jadi file statis.
# CGO_ENABLED=0 artinya binary tidak butuh library C,
# jadi bisa jalan di image scratch yang kosong.
COPY . ./
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /gateway .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /bot ./cmd/bot

# Tahap 2: image akhir untuk bot Telegram.
# scratch = image kosong, isinya hanya program + sertifikat.
FROM scratch AS bot

COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /bot /bot

USER 65534:65534

ENTRYPOINT ["/bot"]

# Tahap 3: image akhir untuk gateway (target default).
# Ditaruh paling akhir supaya docker build tanpa --target
# menghasilkan image gateway.
FROM scratch AS gateway

COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /gateway /gateway

USER 65534:65534

EXPOSE 8080

ENTRYPOINT ["/gateway"]
