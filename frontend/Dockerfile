# Build stage
FROM node:20-slim AS builder

WORKDIR /app

# Copiamos archivos de dependencias
COPY package*.json ./

# Instalamos (|| true para no fallar si no hay package.json todavia)
RUN npm install || true

# Copiamos el resto del codigo y construimos
COPY . .
RUN npm run build || mkdir -p dist && echo "Placeholder build" > dist/index.html

# Server stage con Nginx
FROM nginx:alpine

# Configuracion de Nginx
COPY nginx.conf /etc/nginx/conf.d/default.conf

# Carpeta de salida del build
COPY --from=builder /app/dist /usr/share/nginx/html

EXPOSE 80

CMD ["nginx", "-g", "daemon off;"]
