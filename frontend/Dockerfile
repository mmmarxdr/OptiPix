# Base stage (para desarrollo, no hace el build)
FROM node:20-alpine AS base

WORKDIR /app

# Copiamos archivos de dependencias para aprovechar cache
COPY package*.json ./
RUN npm install

# Copiamos el resto del codigo
COPY . .

# Build stage (hereda de base y compila para produccion)
FROM base AS builder
RUN npm run build

# Server stage con Nginx
FROM nginx:alpine-slim

# Configuracion de Nginx (comentado si no existe nginx.conf, pero por defecto React necesita redirigir a index.html)
COPY nginx.conf /etc/nginx/conf.d/default.conf

# Carpeta de salida del build
COPY --from=builder /app/dist /usr/share/nginx/html

EXPOSE 80

CMD ["nginx", "-g", "daemon off;"]
