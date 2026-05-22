# SQL Migrations

Migration files in this directory use a simple ascending numeric prefix:

- `0001_*.sql`
- `0002_*.sql`
- `0003_*.sql`

Rules:

1. Apply files in filename order.
2. Never edit an old migration after it has been used in a shared environment.
3. Put new schema changes in a new numbered file.
4. Keep `schema.sql` aligned with the latest schema for fresh installs.

Example:

```bash
mysql -u root -p promptos < src/backend/sql/migrations/0001_prompts_cover_and_params.sql
```

Current files:

- `0001_prompts_cover_and_params.sql`
- `0002_fix_seed_text_encoding.sql`
- `0003_users_github_oauth.sql`
- `0004_users_oauth_profile.sql`

The Go backend runs pending migrations automatically on startup when MySQL is available (`schema_migrations` table). Docker images include `sql/migrations` at `/app/sql/migrations`.
