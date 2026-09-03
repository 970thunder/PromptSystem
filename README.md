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

The backend initializes a truly empty configured database from
`src/backend/sql/schema.sql` and then applies pending files from
`src/backend/sql/migrations/` through the `schema_migrations` ledger. No manual
SQL import is needed for a normal local or production startup. Existing or
partially migrated databases only receive migrations that are not already
recorded.

For an existing database where an operator must run one migration manually,
apply incremental SQL files from `src/backend/sql/migrations/` in filename
order. For example:

```bash
mysql -u root -p promptos < src/backend/sql/migrations/0001_prompts_cover_and_params.sql
```

The backend reads runtime configuration from environment variables such as `PORT`, `JWT_SECRET`, `JWT_EXPIRE_HOURS`, `UPLOAD_*`, `R2_*`, `MYSQL_*`, `REDIS_*`, and `ALLOWED_ORIGIN`. Set `UPLOAD_PROVIDER=rustfs` (or `s3`), `R2_ENDPOINT` (or `S3_ENDPOINT`), `R2_ACCESS_KEY_ID`, `R2_SECRET_ACCESS_KEY`, `R2_BUCKET`, and `R2_PUBLIC_URL` for RustFS. The S3-compatible endpoint uses path-style requests and supports HTTP for an internal RustFS network.

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

## One-Click Development Startup

Use the bundled launcher to avoid port conflicts with other local apps. Ports are fixed in the `28301-28399` range:

| Service   | Port  |
|-----------|-------|
| Frontend  | 28301 |
| Backend   | 28302 |
| MySQL     | 28303 |
| Redis     | 28304 |

```bash
# Windows: double-click start-dev.bat, or from Git Bash:
bash scripts/start-dev.sh          # start everything (MySQL/Redis via Docker Compose)
bash scripts/start-dev.sh --no-db  # start without MySQL/Redis (backend memory fallback)
bash scripts/start-dev.sh stop     # stop everything (containers down, data volumes kept)
```

The launcher refuses to start if any fixed application port is already in use, and never silently picks another port. Existing `promptos-mysql` and `promptos-redis` containers are reused. During development the launcher stays in the foreground and follows `logs/frontend.log` and `logs/backend.log`; press `Ctrl+C` or close the launcher window to stop the child services and containers. `stop-dev.bat` remains available for an explicit cleanup.

## Development Order

All new frontend redesign work must follow [`docs/前端重设计开发执行手册.md`](docs/前端重设计开发执行手册.md). It is the current execution baseline: update each checkbox only with an evidence record, commit one task at a time, and push the task branch.

Contributor and AI coding conventions (stack, directories, API style) are defined in `CLAUDE.md`. The backend is **Go** (`src/backend/`), not Spring/Java.

## Environment Naming

- Frontend env vars use the `VITE_` prefix, for example `VITE_API_BASE_URL` and `VITE_APP_TITLE`.
- The home feed uses live prompt APIs; API failures are shown as an actionable error state.
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
