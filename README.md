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
- `POST /api/v1/prompts`
- `POST /api/v1/uploads/images`
- `POST /api/v1/user/login`
- `POST /api/v1/user/register`
- `GET /api/v1/user/info`
- `PUT /api/v1/user/info`

## Database

Initialize the local database with:

```bash
mysql -u root -p < src/backend/sql/schema.sql
```

If you already have an existing database created from an older schema, apply incremental SQL files from `src/backend/sql/migrations/` in filename order. For example:

```bash
mysql -u root -p promptos < src/backend/sql/migrations/0001_prompts_cover_and_params.sql
```

The backend reads runtime configuration from environment variables such as `PORT`, `JWT_SECRET`, `JWT_EXPIRE_HOURS`, `UPLOAD_*`, `R2_*`, `MYSQL_*`, `REDIS_*`, and `ALLOWED_ORIGIN`.

## Docker

```bash
docker compose up --build
```

If your local machine already has MySQL or Redis on the default ports, copy `.env.docker.example` to `.env` and adjust the published host ports before starting Compose.

For authentication, always set a real `PROMPTOS_JWT_SECRET` in `.env` before moving beyond local development.
For image uploads, local development defaults to filesystem storage and serves files under `/uploads`. Production can switch to `PROMPTOS_UPLOAD_PROVIDER=r2` with Cloudflare R2 credentials and a public custom domain.

Docker services:

- Frontend: `http://localhost:3000`
- Backend: `http://localhost:8080`
- MySQL: `localhost:3306`
- Redis: `localhost:6379`

## Development Order

All new frontend redesign work must follow [`docs/前端重设计开发执行手册.md`](docs/前端重设计开发执行手册.md). It is the current execution baseline: update each checkbox only with an evidence record, commit one task at a time, and push the task branch.

Contributor and AI coding conventions (stack, directories, API style) are defined in `CLAUDE.md`. The backend is **Go** (`src/backend/`), not Spring/Java.

## Environment Naming

- Frontend env vars use the `VITE_` prefix, for example `VITE_API_BASE_URL` and `VITE_APP_TITLE`.
- The home feed prefers live prompt APIs and falls back to mock content if the backend is unavailable.
- Backend runtime config is environment-driven for local runs and Docker Compose.
- Passwords are hashed with `bcrypt`, and protected API routes require a bearer JWT.
- Prompt cover uploads validate image MIME type and file size before storing the file.

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
