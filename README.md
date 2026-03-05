# OptiPix

OptiPix es una herramienta *self-hosted* de altísimo rendimiento para optimizar imágenes y gráficos vectoriales. Permite redimensionar, comprimir y convertir fotos a formatos modernos (como AVIF o WebP) a través de una interfaz limpia de arrastrar y soltar.

## Características Principales

- **Altísimo Rendimiento:** Construido en Go utilizando `libvips` (el motor de procesamiento de imágenes más rápido y eficiente en memoria que existe) y `SVGO`.
- **Formatos Modernos:** Soporte nativo para lectura y escritura de **WebP** y **AVIF**.
- **Vectores:** Optimización inteligente de SVGs.
- **Privacidad Total:** Toda la optimización ocurre al vuelo. La aplicación es completamente _stateless_ y ningún archivo es guardado de forma permanente en el servidor.
- **Fácil Despliegue:** Todo el stack está dockerizado para levantarse con un solo comando.

## Stack Tecnológico

- **Frontend:** React 19 + Vite (Single Page Application, diseñado sin frameworks CSS pesados).
- **Backend:** Go 1.22 + `go-chi` router + `govips`.
- **Infraestructura:** Docker + Nginx.

## Cómo ejecutar el proyecto (Recomendado)

La forma más sencilla de correr OptiPix en tu máquina o en tu servidor es usando **Docker**. No necesitás tener instalado ni Go, ni Node.js, ni libvips en tu entorno local.

### Prerrequisitos

* [Docker](https://docs.docker.com/get-docker/) instalado.
* Docker Compose.

### Pasos

1. Cloná este repositorio:
   ```bash
   git clone https://github.com/tu-usuario/optipix.git
   cd optipix
   ```

2. Levantá los contenedores:
   ```bash
   docker compose up --build -d
   ```

3. Listo! Ingresá a la plataforma desde tu navegador:
   * **Frontend UI:** [http://localhost:5173](http://localhost:5173)
   * **Backend API (Health Check):** [http://localhost:8090/api/health](http://localhost:8090/api/health)

## Configuración Avanzada

Si necesitás ajustar parámetros internos (como los MB máximos permitidos por imagen), podés editar de forma sencilla el archivo `docker-compose.yml`. Las siguientes variables de entorno están soportadas por el Backend:

* `MAX_UPLOAD_SIZE`: Tamaño máximo de subida en bytes (Por defecto ~50MB).
* `PORT`: Puerto en el que escucha la API (Por defecto 8090).
* `CORS_ORIGIN`: Orígenes permitidos (Por defecto `*`).

## Desarrollo local (Sin Docker)

Si deseás desarrollar o contribuir al código de forma nativa:
1. Necesitarás tener instalados Go (1.22+), Node.js (18+) y **la librería C `libvips`** (con sus dev headers y `pkg-config`) en tu sistema.
2. Instalación de SVGO de forma global: `npm i -g svgo`.
3. Levantar backend: `cd backend && go run ./cmd/server`
4. Levantar frontend: `cd frontend && npm install && npm run dev`
