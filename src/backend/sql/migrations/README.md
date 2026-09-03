# SQL Migrations

Migration files in this directory use a simple ascending numeric prefix:

- `0001_*.sql`
- `0002_*.sql`
- `0003_*.sql`

Rules:

1. Apply files in filename order.
2. Never edit an old migration after it has been used in a shared environment.
3. Put new schema changes in a new numbered file.
4. Keep `schema.sql` aligned with the latest schema. It is applied automatically
   by the backend only when the configured database has no tables; operators do
   not manually import it during a normal deployment.

Example:

```bash
mysql -u root -p promptos < src/backend/sql/migrations/0001_prompts_cover_and_params.sql
```

Current files:

- `0001_prompts_cover_and_params.sql`
- `0002_fix_seed_text_encoding.sql`
- `0003_users_github_oauth.sql`
- `0004_users_oauth_profile.sql`
- `0005_image_categories_zh.sql`
- `0006_comment_reports.sql`
- `0007_view_histories.sql`
- `0008_prompt_images.sql`
- `0009_users_session_version.sql`
- `0010_prompt_tags_unique.sql`
- `0011_prompt_search_indexes.sql`
- `0012_recalibrate_counters.sql`
- `0013_uploads_table.sql`
- `0014_upload_owner_fk.sql`
- `0015_normalize_prompt_tags.sql`
- `0016_comment_sort_indexes.sql`

The Go backend applies the baseline automatically for a truly empty database,
then runs pending migrations on startup using the `schema_migrations` table.
Docker images include both `sql/schema.sql` and `sql/migrations` at `/app/sql`.
