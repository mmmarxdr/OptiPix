# Build stage
FROM golang:1.22-bookworm AS builder

# Instalar dependencias para libvips
RUN apt-get update && apt-get install -y libvips-dev pkg-config

WORKDIR /app

# Construimos cache de modulos de Go primero
COPY go.mod go.sum* ./
RUN go mod download

# Copiamos el codigo y compilamos
COPY . .
RUN CGO_ENABLED=1 GOOS=linux go build -o optipix-api ./cmd/server

# Runtime stage
FROM debian:bookworm-slim

# Instalar dependencias necesarias para runtime y limpiar cache de apt para una imagen mas ligera
RUN apt-get update && apt-get install -y --no-install-recommends \
    libvips42 \
    nodejs \
    npm \
    ffmpeg \
    ca-certificates \
    && npm install -g svgo \
    && apt-get clean \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /app

# Copiar el binario compilado desde el builder
COPY --from=builder /app/optipix-api .

EXPOSE 8090

# Ejecutar el servicio
CMD ["./optipix-api"]
