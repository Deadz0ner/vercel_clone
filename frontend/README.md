# Frontend

React + TypeScript + Vite frontend for the Vercel clone.

## Setup

```bash
npm install
npm run dev
```

The dev server starts on `http://localhost:5173`.

## Environment Variables

Create a `.env` file in the `frontend/` directory (optional — defaults work for local dev):

```env
VITE_BACKEND_URL=http://localhost:8020
VITE_WS_URL=ws://localhost:8020
```

## How It Works

1. User pastes a GitHub repo URL and clicks **Upload**
2. Frontend sends `POST /deploy` to the entry service, receives a project ID
3. Frontend opens a WebSocket connection to `ws://localhost:8020/ws/{projectId}`
4. Real-time status updates appear as the backend progresses:
   - **Queued** — job is in the Redis queue
   - **Building** — Docker container is running `npm install && npm run build`
   - **Deploying** — build artifacts are being uploaded to S3
   - **Deployed** — done, the deployed URL is shown
   - **Failed** — something went wrong, check server logs
5. On success, the deployed URL is displayed with a link to visit the site

## Tech Stack

- React 18
- TypeScript
- Vite
- Tailwind CSS
- Radix UI components
- Native WebSocket API (no polling)
