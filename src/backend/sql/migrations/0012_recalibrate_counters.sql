-- Purpose: reconcile the denormalized prompts.likes / prompts.favorites /
-- prompts.views counters with their detail tables (likes, favorites,
-- view_histories) after any historical double-write gap, partial write, or
-- manual edit, so the public counters exactly match the number of unique detail
-- rows. Normally the store write path performs these updates in the same
-- transaction as the detail-row insert; this migration repairs any legacy drift.
--
-- Semantics:
--   * likes     = number of distinct 'prompt' rows in `likes`
--   * favorites = number of distinct 'prompt' rows in `favorites`
--   * views     = number of distinct (user_id, prompt_id) rows in
--                 view_histories, i.e. distinct logged-in viewers.
--
--   Anonymous views are NOT persisted to any detail table (see the store,
--   which keeps only the total-view counter and no attribution row), so they
--   cannot be reconstructed from history. This recalibration normalizes views
--   to the logged-in baseline; after it runs, every anonymous view re-bumps the
--   prompts.views counter going forward, and every logged-in user's first view
--   of a prompt also bumps it once (repeat views only refresh viewed_at).
--
-- Compatibility / idempotency: each UPDATE derives its value from a pure LEFT
-- JOIN aggregate over the detail tables, so re-running the migration converges
-- to the same result and never compounds the counters. Only the three counter
-- columns are touched; no detail data is altered or dropped. It is safe to run
-- on a fresh database (the aggregate simply yields 0) and on any existing
-- database that already applied earlier baselines.
--
-- Irreversibility: re-running is the only "revert" path and is non-destructive.
USE promptos;

UPDATE prompts p
LEFT JOIN (
    SELECT target_id, COUNT(*) AS cnt
    FROM likes
    WHERE target_type = 'prompt'
    GROUP BY target_id
) d ON d.target_id = p.id
SET p.likes = COALESCE(d.cnt, 0);

UPDATE prompts p
LEFT JOIN (
    SELECT target_id, COUNT(*) AS cnt
    FROM favorites
    WHERE target_type = 'prompt'
    GROUP BY target_id
) d ON d.target_id = p.id
SET p.favorites = COALESCE(d.cnt, 0);

UPDATE prompts p
LEFT JOIN (
    SELECT prompt_id, COUNT(*) AS cnt
    FROM view_histories
    GROUP BY prompt_id
) d ON d.prompt_id = p.id
SET p.views = COALESCE(d.cnt, 0);
