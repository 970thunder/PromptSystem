# PromptOS

AI Prompt & Skill community platform. The current MVP focuses on the Prompt community flow: home feed, prompt detail, login/register, publish, search, and interactions.

## Project Structure

```text
src/
  frontend/   Vue 3 + TypeScript + Vite + TailwindCSS
  backend/    Go + net/http + MySQL + Redis
```

## Frontend

```bash
cd src/frontend
npm install
npm run dev
npm run build
npm run lint
```

The frontend dev server runs on `http://localhost:3000` and proxies `/api` to `http://localhost:8080`.

## Backend

```bash
cd src/backend
go run ./cmd/api
go test ./...
```

Backend defaults:

- Server: `http://localhost:8080`
- MySQL database: `promptos`
- Redis: `localhost:6379`

Available MVP APIs:

- `GET /api/v1/health`
- `GET /api/v1/categories`
- `GET /api/v1/prompts`
- `GET /api/v1/prompts/:id`

## Database

Initialize the local database with:

```bash
mysql -u root -p < src/backend/sql/schema.sql
```

The backend reads runtime configuration from environment variables such as `PORT`, `MYSQL_*`, `REDIS_*`, and `ALLOWED_ORIGIN`.

## Docker

```bash
docker compose up --build
```

If your local machine already has MySQL or Redis on the default ports, copy `.env.docker.example` to `.env` and adjust the published host ports before starting Compose.

Docker services:

- Frontend: `http://localhost:3000`
- Backend: `http://localhost:8080`
- MySQL: `localhost:3306`
- Redis: `localhost:6379`

## Development Order

All development should follow `TODO.md`. Update the checklist after each verified task, then commit the verified changes.

## Environment Naming

- Frontend env vars use the `VITE_` prefix, for example `VITE_API_BASE_URL` and `VITE_APP_TITLE`.
- The home feed prefers live prompt APIs and falls back to mock content if the backend is unavailable.
- Backend runtime config is environment-driven for local runs and Docker Compose.

## Verification Commands

```bash
cd src/frontend
npm run lint
npm run build

cd ../backend
go test ./...
go build ./cmd/api

cd ../..
docker compose config
```
