# Build stage (compilamos el código en un entorno liviano)
FROM golang:1.22-alpine AS builder

# Alpine usa apk en vez de apt. Instalamos dependencias para libvips
RUN apk add --no-cache build-base pkgconfig vips-dev vips-heif libheif

WORKDIR /app

# Maximizamos el cache de Go
COPY go.mod go.sum* ./
RUN go mod download

# Compilamos ambos puntos de entrada
COPY . .
RUN CGO_ENABLED=1 GOOS=linux go build -o optipix-api ./cmd/server
RUN CGO_ENABLED=1 GOOS=linux go build -o optipix-cli ./cmd/cli

# Base runtime para no repetir dependencias
FROM alpine:3.19 AS base-runtime

RUN apk add --no-cache \
    vips \
    vips-heif \
    libheif \
    nodejs \
    npm \
    ffmpeg \
    ca-certificates \
    && npm install -g svgo

WORKDIR /app

# ---- TARGET: API (Servidor HTTP) ----
FROM base-runtime AS api
# Aqui SOLO traemos el binario de la API
COPY --from=builder /app/optipix-api .

EXPOSE 8090
CMD ["./optipix-api"]

# ---- TARGET: CLI (CI/CD / Terminal) ----
FROM base-runtime AS cli
# Aqui SOLO traemos el binario del CLI
COPY --from=builder /app/optipix-cli .

ENTRYPOINT ["./optipix-cli"]
