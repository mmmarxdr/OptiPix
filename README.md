# OptiPix

OptiPix is an ultra-high performance, dual-purpose media optimization tool for raster images and vector graphics. It empowers you to effortlessly resize, compress, and convert photos into modern next-gen formats (like AVIF and WebP). 

OptiPix is designed to be used in two distinct ways:
1. **As a Self-Hosted Web App:** A beautifully clean drag-and-drop interface for end-users, powered by a robust backend API.
2. **As a CLI Utility:** A standalone command-line tool built for seamless integration into CI/CD pipelines (GitHub Actions, GitLab CI, etc.) for automated file batch processing.

## Key Features

- **Blazing Fast Performance:** Powered by Go under the hood, utilizing `libvips` (the fastest and most memory-efficient image processing library around) alongside `SVGO`.
- **Next-Gen Formats:** Full native support for decoding and encoding the latest **AVIF** and **WebP** formats.
- **Smart CLI Batch Processing:** The CLI tool hashes files to maintain a state file, skipping already-processed images to make CI/CD runs lightning fast.
- **Vector Graphics:** Intelligent path optimization for SVGs.
- **Absolute Privacy:** All processing in the Web App happens entirely "on the fly". The application is completely _stateless_ and NO files are ever persistently stored on the server disk.
- **Free-Tier Friendly (Anti-Abuse):** The API features built-in middleware for IP Rate Limiting and strict max upload sizes to protect your server from abuse and avoid unpredicted Cloud bills.

## Tech Stack

- **Core & API & CLI:** Go 1.22 + `govips` + `go-chi` router.
- **Frontend:** React 19 + Vite (A highly responsive Single Page Application built natively without heavy CSS frameworks).
- **Infrastructure:** Docker Multi-stage builds for isolated binaries (`api` vs `cli`).

---

## 🚀 1. Running the Web App (Server + UI)

The simplest way to run the full OptiPix platform—whether locally on your computer or deployed to a server—is via **Docker Compose**. You don't need to install Go, Node.js, or complex C-headers.

### Prerequisites
* [Docker](https://docs.docker.com/get-docker/) installed.
* Docker Compose V2.

### Steps
1. Clone this repository:
   ```bash
   git clone https://github.com/your-username/optipix.git
   cd optipix
   ```
2. Spin up the containers in the background:
   ```bash
   docker compose up --build -d
   ```
3. Access the platform:
   * **Frontend UI:** [http://localhost:5173](http://localhost:5173)
   * **Backend API (Health Check):** [http://localhost:8090/api/health](http://localhost:8090/api/health)

---

## 🛠 2. Using the CLI (For CI/CD & Batch Processing)

Under the hood, OptiPix utilizes a **Multi-Stage Dockerfile**. If you only want the CLI, you won't download or host the web server binary. 

To build the CLI-only Docker image:
```bash
docker build --target cli -t optipix-cli ./backend
```

### Running the CLI via Docker
Because the image includes an `ENTRYPOINT`, you can treat the Docker invocation natively as a command. You just need to mount your local directories.

**Example Usage:**
```bash
docker run --rm \
  -v $(pwd)/public/images:/images \
  -v $(pwd)/public/optimized:/optimized \
  optipix-cli \
  -input /images \
  -output /optimized \
  -format webp \
  -quality 80 \
  -state /images/.optipix-state.json
```

### CLI Configuration Flags

| Flag | Default | Description |
| :--- | :--- | :--- |
| `-input` | `./images` | Local directory to scan for original images (*.jpg, *.png, *.webp, *.avif). |
| `-output` | `./optimized` | Destination directory. **Tip:** Set this strictly identical to `-input` to magically overwrite your files "in-place", creating a perfect seamless CI/CD experience. |
| `-format` | `auto` | Final output format. **`auto`** detects your original extension (e.g., `.png`) and recompresses it efficiently as a modern `.png`. Accepted: `auto`, `webp`, `avif`, `jpeg`, `png`. |
| `-quality` | `80` | Compression quality (0 to 100). |
| `-state` | `.optipix-state.json`| Path to the local state file used for hashing. Must be kept between runs for the "Smart Skip" cache to work. |

### Preventing React / Vite Broken Imports in CI/CD

If your codebase relies on static imports like `import Hero from './hero.png'`, blindly converting `.png` files to `.webp` will break your Javascript build step!

**The solution:** Simply use `-format auto` and output to the exact same folder. OptiPix will heavily recompress `hero.png` keeping its `.png` extension and replacing the old heavy file. Your code imports will never break!

```yaml
- name: Optimize Images with OptiPix CLI
  run: |
    docker run --rm \
      -v ${{ github.workspace }}/public/images:/images \
      -v ${{ github.workspace }}/public/optimized:/optimized \
      optipix-cli:latest \
      -input /images \
      -output /optimized \
      -format avif \
      -quality 75 \
      -state /images/.optipix-state.json

- name: Commit state changes
  run: |
    git add public/images/.optipix-state.json
    git commit -m "chore: update image optimization cache [skip ci]" || echo "No changes to commit"
    git push
```

---

## ⚙️ API Configuration & Security Limits

If you plan to expose the Web API to the public internet, you can easily tune internal limits by editing environment variables inside the `docker-compose.yml`.

* `MAX_UPLOAD_SIZE`: Max upload size allowed in bytes. (*Defaults to 52428800 bytes / ~50MB*). It is recommended to reduce this to `10485760` (10MB) for public APIs.
* `RATE_LIMIT_PER_MINUTE`: How many images a single IP address can request within 60 seconds before getting blocked (*Defaults to 60*).
* `MAX_CONCURRENCY`: Prevents Out-Of-Memory (OOM) crashes by strictly limiting how many heavy images `libvips` processes in parallel via Go Channels (*Defaults to 4*).
* `PORT`: Server listening port (*Defaults to 8090*).
* `CORS_ORIGIN`: Allowed origins. Make sure to hardcode this to your exact frontend domain (e.g. `https://optipix.your-domain.com`) before production deployment to prevent 3rd-party abuse!

## 💻 Local Development (Without Docker)

If you wish to contribute to the code natively without containers:
1. You must install Go (1.22+), Node.js (18+), and the **`libvips` C library** (along with its dev headers and `pkg-config`) on your operating system.
2. Install SVGO globally: `npm i -g svgo`.
3. Boot the backend server: `cd backend && go run ./cmd/server`
4. Boot the frontend app: `cd frontend && npm install && npm run dev`
