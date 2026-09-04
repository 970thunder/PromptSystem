#!/usr/bin/env bash
# 文件作用：PromptOS MySQL 每日逻辑备份。串行（flock）执行 mysqldump，
# gzip 压缩、SHA-256 校验、按天保留，失败或校验不过即调用告警通道。
# 对应完整迭代 TODO D-11；恢复演练流程见 docs/DEPLOYMENT.md（D-14）。
set -uo pipefail

RELEASE_DIR=${PROMPTOS_RELEASE_DIR:-/srv/releases/promptsystem/current}
COMPOSE_ENV=${PROMPTOS_COMPOSE_ENV:-/opt/secrets/promptsystem/app.env}
BACKUP_ROOT=${PROMPTOS_BACKUP_ROOT:-/srv/backups/promptsystem/daily}
RETAIN_DAYS=${PROMPTOS_BACKUP_RETAIN_DAYS:-14}
LOCK_FILE=/run/lock/promptos-backup.lock
ALERT_SCRIPT=/usr/local/bin/promptos-alert.sh

DATE=$(date +%F)
DEST="$BACKUP_ROOT/$DATE"

exec 9>"$LOCK_FILE"
if ! flock -n 9; then
  echo "another promptos backup is running; skip" >&2
  exit 0
fi

mkdir -p "$DEST"
cd "$RELEASE_DIR" || exit 1

echo "[$(date -Is)] dumping promptos database..."
docker compose -p promptsystem --env-file "$COMPOSE_ENV" \
  exec -T mysql sh -c 'exec mysqldump -uroot -p"$MYSQL_ROOT_PASSWORD" --single-transaction --routines --events --databases promptos' 2>/dev/null \
  | gzip -9 > "$DEST/mysql-promptos.sql.gz"

DUMP_OK=$?
SIZE=$(stat -c %s "$DEST/mysql-promptos.sql.gz" 2>/dev/null || echo 0)
if [ "$DUMP_OK" -ne 0 ] || [ "$SIZE" -lt 1024 ]; then
  [ -x "$ALERT_SCRIPT" ] && "$ALERT_SCRIPT" \
    "PromptOS 每日备份失败：$DATE" \
    "mysqldump 失败或产物为空（size=$SIZE）。请检查 journalctl -u promptos-backup 与 MySQL 状态。"
  echo "backup failed (size=$SIZE)" >&2
  exit 1
fi

sha256sum "$DEST/mysql-promptos.sql.gz" > "$DEST/SHA256SUMS"
if ! (cd "$DEST" && sha256sum -c SHA256SUMS >/dev/null 2>&1) || ! gzip -t "$DEST/mysql-promptos.sql.gz" 2>/dev/null; then
  [ -x "$ALERT_SCRIPT" ] && "$ALERT_SCRIPT" \
    "PromptOS 备份校验失败：$DATE" \
    "SHA-256 或 gzip 校验未通过：$DEST。备份可能损坏，请勿用于恢复并立即排查。"
  echo "verify failed" >&2
  exit 1
fi

find "$BACKUP_ROOT" -mindepth 1 -maxdepth 1 -type d -mtime +"$RETAIN_DAYS" -exec rm -rf {} \; 2>/dev/null

echo "[$(date -Is)] backup ok: $DEST/mysql-promptos.sql.gz ($SIZE bytes)"
echo "[$(date -Is)] retained dirs: $(find "$BACKUP_ROOT" -mindepth 1 -maxdepth 1 -type d | wc -l)"
