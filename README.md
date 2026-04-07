# Vercel Clone

A self-hosted deployment platform that takes a GitHub repository URL, builds it, and serves the static output — similar to how Vercel works.

## Architecture

```
Frontend (React)  ──POST /deploy──▶  Entry Service (:8020)  ──RPush──▶  Redis Queue
       │                                    │                               │
       │ WebSocket /ws/{id}                 │ Subscribe status:{id}         │ BLPop
       │◀──── status updates ──────────────▶│                               ▼
       │                                    │◀── Pub/Sub ──────────  Build Worker
       │                                                              │
       │                                                              │ Upload artifacts
       │                                                              ▼
       │── GET /{id}/index.html ──────────────────────────────▶  Serve (:3001) ──▶ Supabase S3
```

**Three Go services + a React frontend:**

| Service | Port | Role |
|---------|------|------|
| Entry   | 8020 | Accepts deploy requests, clones repos, queues jobs, serves WebSocket |
| Build   | —    | Worker: builds projects in Docker, uploads artifacts to S3 |
| Serve   | 3001 | Serves deployed static files from S3 |
| Frontend| 5173 | React UI with real-time deploy status via WebSocket |

## Prerequisites

- Go 1.24+
- Docker
- Node.js 18+ (for the frontend)
- Redis (runs via Docker Compose)

## Quick Start

### 1. Start Redis

```bash
docker compose up -d redis
```

### 2. Configure environment

```bash
cp golang/.env.example golang/.env
# Fill in your Supabase S3 credentials
```

### 3. Start the Go services (3 terminals)

```bash
# Terminal 1 — Entry service
cd golang && go run ./cmd/1.entry

# Terminal 2 — Build worker
cd golang && go run ./cmd/2.build

# Terminal 3 — Serve service
cd golang && go run ./cmd/3.serve
```

### 4. Start the frontend

```bash
cd frontend
npm install
npm run dev
```

### 5. Deploy a repo

Open `http://localhost:5173`, paste a GitHub repo URL, and click Upload. You'll see real-time status updates as it goes through: queued → building → uploading → deployed.

## Directory Structure

```
├── frontend/               # React + Vite frontend
├── golang/
│   ├── cmd/
│   │   ├── 1.entry/        # HTTP + WebSocket entry service
│   │   ├── 2.build/        # Build worker (Docker + S3 upload)
│   │   └── 3.serve/        # Static file server (S3 proxy)
│   └── internal/
│       ├── builder/         # Docker container management
│       ├── config/          # Environment config
│       ├── deploy/          # Git clone logic
│       ├── queue/           # Redis queue + pub/sub helpers
│       └── utils/           # S3 upload, ID generation
├── node/                    # Node.js implementation (reference)
├── docker-compose.yml       # Redis
└── README.md
```

## Real-Time Status Flow

The frontend connects via WebSocket after submitting a deploy request. Status updates flow through Redis Pub/Sub:

1. **Entry service** publishes `queued` after pushing the job to Redis
2. **Build worker** publishes `building` when it picks up the job
3. **Build worker** publishes `uploading` after build completes, during S3 upload
4. **Build worker** publishes `deployed:<url>` on success, or `failed` on error
5. **Entry service** relays all messages from the Pub/Sub channel to the WebSocket client
