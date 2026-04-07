# Go Services

The backend of the Vercel clone — three services that handle the full deploy pipeline.

## Services

### 1. Entry Service (`cmd/1.entry`)

HTTP + WebSocket server on port **8020**.

- `POST /deploy` — accepts `{ "repoURL": "..." }`, clones the repo, pushes job to Redis queue, returns `{ "id": "abc123" }`
- `GET /ws/{id}` — WebSocket endpoint that subscribes to Redis Pub/Sub channel `status:{id}` and forwards real-time status updates to the client
- `GET /health` — health check

```bash
go run ./cmd/1.entry
```

### 2. Build Worker (`cmd/2.build`)

Background worker process (no HTTP API).

- Blocks on Redis queue (`deploy:jobs`) waiting for project IDs
- Spins up a Docker container (`node:20-alpine`) to run `npm install && npm run build`
- Uploads build artifacts (`dist/` or `build/`) to Supabase S3
- Publishes status updates via Redis Pub/Sub at each stage: `building` → `uploading` → `deployed:<url>` (or `failed`)
- Cleans up local files after upload

```bash
go run ./cmd/2.build
```

### 3. Serve Service (`cmd/3.serve`)

Static file server on port **3001**.

- Routes `GET /<projectId>/<filepath>` to the corresponding S3 object
- Defaults to `index.html` when only the project ID is provided
- Sets correct MIME content types

```bash
go run ./cmd/3.serve
```

## Prerequisites

- Go 1.24+
- Docker (for build containers)
- Redis (via `docker compose up -d redis` from repo root)

## Environment Variables

Create a `.env` file in the `golang/` directory:

```env
# Server
PORT=8020

# Project storage
BASE_DIR=local

# Redis
REDIS_ADDR=localhost:6379
REDIS_PASSWORD=
REDIS_DB=0
REDIS_QUEUE=deploy:jobs

# Docker builder
BUILDER_IMAGE=node:20-alpine
BUILDER_COMMAND=npm install && npm run build

# Supabase S3
SUPABASE_S3_BUCKET=your_bucket
SUPABASE_S3_ENDPOINT=https://your-project.storage.supabase.co/storage/v1/s3
SUPABASE_S3_REGION=ap-southeast-1
SUPABASE_S3_ACCESS_KEY_ID=your_access_key
SUPABASE_S3_SECRET_ACCESS_KEY=your_secret_key

# Serve
SERVE_HOST=http://localhost:3001
```

## Running All Services

You need **3 terminals** (plus Redis):

```bash
# Terminal 0 — Redis (from repo root)
docker compose up -d redis

# Terminal 1 — Entry service
cd golang && go run ./cmd/1.entry

# Terminal 2 — Build worker
cd golang && go run ./cmd/2.build

# Terminal 3 — Serve service
cd golang && go run ./cmd/3.serve
```

## Test A Deploy Request

```bash
curl -X POST http://localhost:8020/deploy \
  -H "Content-Type: application/json" \
  -d '{"repoURL": "https://github.com/Deadz0ner/algo-vizz"}'
```

Then connect to the WebSocket for live updates:

```bash
websocat ws://localhost:8020/ws/<id-from-response>
```

## How Status Updates Flow

Redis Pub/Sub bridges the services:

```
Entry Service                    Redis                     Build Worker
     │                             │                            │
     │── RPush deploy:jobs ──────▶│                            │
     │── Publish "queued" ───────▶│                            │
     │                             │◀── BLPop ─────────────────│
     │                             │◀── Publish "building" ────│
     │◀── Subscribe status:{id} ──│                            │
     │                             │◀── Publish "uploading" ───│
     │                             │◀── Publish "deployed:url" │
     ▼                             │                            │
  WebSocket → Frontend             │                            │
```

## Docker Build Behavior

- Fresh container per build (`node:20-alpine` by default)
- Project directory mounted at `/workspace/project`
- Network enabled (for `npm install`)
- 2-minute timeout — container is killed if exceeded
- Live stdout/stderr streaming to worker logs
