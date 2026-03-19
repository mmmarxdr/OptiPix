# ── Stage 1: Builder ──────────────────────────────────────────────────────────
# Compilamos el binario CLI con la misma imagen base que el backend para
# garantizar compatibilidad de glibc con la etapa de runtime.
FROM golang:1.22-bookworm AS builder

# Instalar cabeceras de desarrollo de libvips (necesarias para compilar govips con CGO)
RUN apt-get update && apt-get install -y \
    libvips-dev \
    pkg-config \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /build

# Construimos cache de modulos de Go primero (se invalida solo cuando cambian go.mod/go.sum)
COPY go.mod go.sum* ./
RUN go mod download

# Copiamos el codigo fuente y compilamos solo el binario CLI
# -ldflags="-s -w" elimina simbolos de debug para reducir el tamaño del binario
COPY . .
RUN CGO_ENABLED=1 GOOS=linux go build -ldflags="-s -w" -o /app/optipix ./cmd/cli

# ── Stage 2: Runtime ─────────────────────────────────────────────────────────
# Imagen minima con solo las dependencias de runtime necesarias.
# Alpine NO se usa porque govips requiere glibc (musl libc es incompatible).
FROM debian:bookworm-slim

# Instalar dependencias de runtime: libvips, Node.js/npm para SVGO, y ca-certificates.
# npm cache clean y rm -rf en la misma capa RUN para no inflar la imagen.
RUN apt-get update && apt-get install -y --no-install-recommends \
        libvips42 \
        nodejs \
        npm \
        ca-certificates \
    && npm install -g svgo@3 \
    && npm cache clean --force \
    && apt-get clean \
    && rm -rf /var/lib/apt/lists/*

# Crear usuario no-root con UID 1000 (compatibilidad con runners de CI donde el usuario suele ser UID 1000)
RUN groupadd -g 1000 optipix && \
    useradd -u 1000 -g optipix -m -s /bin/sh optipix

# Copiar el binario compilado desde el builder
COPY --from=builder /app/optipix /usr/local/bin/optipix

# Etiquetas OCI estandar para trazabilidad de la imagen
LABEL org.opencontainers.image.title="OptiPix CLI" \
      org.opencontainers.image.description="CLI de optimizacion de imagenes y SVGs sin necesidad de instalar Go, libvips ni SVGO localmente" \
      org.opencontainers.image.source="https://github.com/optipix/backend" \
      org.opencontainers.image.licenses="MIT"

# Directorio de trabajo donde se montan los assets del host via bind mount
WORKDIR /work

# Ejecutar como usuario no-root
USER optipix

# El binario optipix es el punto de entrada; CMD ofrece ayuda por defecto si no se pasan argumentos
ENTRYPOINT ["optipix"]
CMD ["--help"]
