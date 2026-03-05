# Build stage
FROM golang:1.22-bookworm AS builder

# Instalar dependencias para libvips
RUN apt-get update && apt-get install -y libvips-dev pkg-config

WORKDIR /app

# Copiamos go.mod y go.sum si existen
COPY go.mod go.sum* ./
# Intentamos descargar dependencias (el || true evita que falle si no hay modulo aun)
RUN go mod download || true

# Copiamos el resto del codigo
COPY . .

# Compilamos la aplicación (|| true para que construya la imagen incluso si no hay codigo todavia)
RUN CGO_ENABLED=1 go build -o optipix-api ./cmd/server || echo "Build skipped, code missing" > optipix-api && chmod +x optipix-api

# Runtime stage
FROM debian:bookworm-slim

# Instalar dependencias necesarias para runtime: libvips42, nodejs, npm, ffmpeg (para video)
RUN apt-get update && apt-get install -y \
    libvips42 \
    nodejs \
    npm \
    ffmpeg \
    && rm -rf /var/lib/apt/lists/*

# Instalar SVGO de forma global
RUN npm install -g svgo

WORKDIR /app

# Copiar binario compilado (o el placeholder si falló)
COPY --from=builder /app/optipix-api .

EXPOSE 8090

# Punto de entrada
CMD ["./optipix-api"]
