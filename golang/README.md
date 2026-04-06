# Go Services

This directory contains the Go version of the deploy pipeline.

Current flow:

1. `cmd/1.deployer` receives a GitHub repo URL on `POST /deploy`
2. it clones the repo into `BASE_DIR/<project-id>`
3. it pushes the generated project id into Redis
4. `cmd/2.upload` blocks on the Redis queue
5. when a project id arrives, it starts a fresh Docker build container
6. the builder container runs `npm install && npm run build`
7. after build, the worker looks for `dist/` or `build/`

At the moment, the upload worker reaches the Docker build step and detects the artifact directory, but the final artifact upload/cleanup flow is not fully wired yet.

## Services

- `main.go`
  - local multi-service runner
  - starts the services under `cmd/*`
- `cmd/1.deployer`
  - HTTP service
  - accepts deploy requests
  - clones repositories locally
  - pushes job ids to Redis
- `cmd/2.upload`
  - background worker
  - consumes job ids from Redis
  - builds projects inside short-lived Docker containers
- `cmd/test`
  - local playground for trying integrations before moving them into the real services

## Dependencies

You need these installed locally:

- Go
- Docker
- Docker Compose
- Redis

Redis is already defined in the repo-level [docker-compose.yml](/home/zed/Desktop/code/vercel_clone/docker-compose.yml), so the usual way to run Redis here is through Docker Compose.

You also need a valid [golang/.env](/home/zed/Desktop/code/vercel_clone/golang/.env) file with:

- Supabase S3 credentials
- Redis connection settings
- builder image / builder command settings

## Important Env Vars

Configured through [config.go](/home/zed/Desktop/code/vercel_clone/golang/internal/config/config.go):

- `PORT`
- `BASE_DIR`
- `REDIS_ADDR`
- `REDIS_PASSWORD`
- `REDIS_DB`
- `REDIS_QUEUE`
- `BUILDER_IMAGE`
- `BUILDER_COMMAND`
- `SUPABASE_S3_BUCKET`
- `SUPABASE_S3_REGION`
- `SUPABASE_S3_ENDPOINT`
- `SUPABASE_S3_ACCESS_KEY_ID`
- `SUPABASE_S3_SECRET_ACCESS_KEY`

Current defaults from [golang/.env](/home/zed/Desktop/code/vercel_clone/golang/.env):

- `BASE_DIR=local`
- `REDIS_ADDR=localhost:6379`
- `REDIS_DB=0`
- `REDIS_QUEUE=deploy:jobs`
- `BUILDER_IMAGE=node:20-alpine`
- `BUILDER_COMMAND=npm install && npm run build`

## Start Redis

From the repository root:

```bash
docker compose up -d redis
```

This starts Redis on `localhost:6379`.

## Start The Go Services

From the `golang/` directory:

```bash
go run .
```

This starts the local multi-service runner in [main.go](/home/zed/Desktop/code/vercel_clone/golang/main.go), which then launches the services found under `cmd/*`.

If you want to run services individually:

```bash
go run ./cmd/1.deployer
go run ./cmd/2.upload
```

## Test A Deploy Request

With Redis running and the Go services started:

```bash
curl -X POST http://localhost:8020/deploy \
  -H "Content-Type: application/json" \
  -d '{
    "projectName": "demo",
    "repoURL": "https://github.com/Deadz0ner/algo-vizz"
  }'
```

Expected behavior:

1. deployer generates an id
2. repo is cloned into `golang/local/<id>`
3. project id is pushed into Redis
4. upload worker pops the id
5. upload worker starts a fresh Docker container
6. build runs with the configured builder image
7. worker checks for `dist/` or `build/`

## Docker Build Behavior

The upload worker uses [docker.go](/home/zed/Desktop/code/vercel_clone/golang/internal/builder/docker.go).

Current behavior:

- fresh container per build
- existing image reused across builds
- project directory mounted into the container
- network enabled for now
- build timeout is 2 minutes
- on timeout, the Docker command is killed and logs are returned

This is meant to isolate untrusted repo build steps from the host process.

## Local Project Layout

Cloned repositories are stored under:

```text
golang/local/<project-id>
```

The upload worker resolves that path using [makeId.go](/home/zed/Desktop/code/vercel_clone/golang/internal/utils/makeId.go).

## Playground / Experiments

Use [main.go](/home/zed/Desktop/code/vercel_clone/golang/cmd/test/main.go) for trying new tech integrations first.

Suggested pattern:

1. write a small isolated test helper in `cmd/test`
2. verify it works there
3. move reusable logic into `internal/...`
4. wire the real service entrypoints after that

This is a good place to try:

- Redis experiments
- DB connection checks
- Docker command experiments
- storage connection tests

## Current Gaps

These parts are not fully completed yet:

- final upload of built static artifacts from `cmd/2.upload`
- job status tracking
- cleanup of project directories after successful processing
- stronger Docker hardening such as memory / CPU limits

## Notes

- `cmd/2.upload` is a worker process, not a user-facing HTTP API right now
- Redis is used as a list queue, not pub/sub
- if Docker is unavailable, build jobs will fail at the worker step
- if a project is not a Node app that supports the configured build command, the build will fail
