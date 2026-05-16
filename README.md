# PromptOS

AI Prompt & Skill community platform. The current MVP focuses on the Prompt community flow: home feed, prompt detail, login/register, publish, search, and interactions.

## Project Structure

```text
src/
  frontend/   Vue 3 + TypeScript + Vite + TailwindCSS
  backend/    Spring Boot + MyBatis Plus + MySQL + Redis
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
mvn spring-boot:run
mvn test -DskipTests
```

Backend defaults:

- Server: `http://localhost:8080`
- MySQL database: `promptos`
- Redis: `localhost:6379`

## Database

Initialize the local database with:

```bash
mysql -u root -p < src/backend/sql/schema.sql
```

The development datasource is configured in `src/backend/src/main/resources/application.yml`.

## Development Order

All development should follow `TODO.md`. Update the checklist after each verified task, then commit the verified changes.

