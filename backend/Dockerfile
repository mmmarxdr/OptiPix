FROM golang:1.22-alpine AS builder

RUN apk add --no-cache build-base pkgconfig vips-dev vips-heif libheif

WORKDIR /app

COPY go.mod go.sum* ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=1 GOOS=linux go build -o optipix-api ./cmd/server
RUN CGO_ENABLED=1 GOOS=linux go build -o optipix ./cmd/cli

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

# Copiar los binarios compilados desde el builder
COPY --from=builder /app/optipix-api .
COPY --from=builder /app/optipix /usr/local/bin/optipix

EXPOSE 8090
CMD ["./optipix-api"]

FROM base-runtime AS cli
COPY --from=builder /app/optipix-cli .

ENTRYPOINT ["./optipix-cli"]
