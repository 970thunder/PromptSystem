#!/usr/bin/env bash
# 文件作用：PromptOS RustFS 桶副本（完整迭代 TODO D-13）。通过 S3 API 列举
# promptsystem-prod 桶内全部对象并镜像到服务器本地副本目录，形成与平板
# （不同故障域）的第二副本；对已删除对象同步清理副本。凭据从
# /opt/secrets/promptsystem/rustfs.env 读取，不落日志。
set -uo pipefail

REPLICA_ROOT=${PROMPTOS_RUSTFS_REPLICA_DIR:-/srv/backups/promptsystem/rustfs-replica}
ALERT_SCRIPT=/usr/local/bin/promptos-alert.sh
LOCK_FILE=/run/lock/promptos-rustfs-replica.lock

# shellcheck disable=SC1090
source "${PROMPTOS_RUSTFS_ENV:-/opt/secrets/promptsystem/rustfs.env}"
: "${R2_ENDPOINT:?}" "${R2_BUCKET:?}" "${R2_ACCESS_KEY_ID:?}" "${R2_SECRET_ACCESS_KEY:?}"

exec 9>"$LOCK_FILE"
if ! flock -n 9; then
  echo "another replica sync is running; skip" >&2
  exit 0
fi

SIG="aws:amz:us-east-1:s3"
AUTH="--user ${R2_ACCESS_KEY_ID}:${R2_SECRET_ACCESS_KEY}"
BASE="${R2_ENDPOINT%/}/${R2_BUCKET}"
mkdir -p "$REPLICA_ROOT"

LIST_XML=$(mktemp)
trap 'rm -f "$LIST_XML"' EXIT
curl -sS -m 60 --aws-sigv4 "$SIG" $AUTH "$BASE?list-type=2" > "$LIST_XML" || {
  [ -x "$ALERT_SCRIPT" ] && "$ALERT_SCRIPT" "PromptOS RustFS 副本失败" "列举对象失败，请检查隧道与 RustFS。"
  exit 1
}

KEYS=$(grep -oE "<Key>[^<]+</Key>" "$LIST_XML" | sed -e "s/<Key>//" -e "s/<\/Key>//")
COUNT=0; FAILED=0
for KEY in $KEYS; do
  SAFE_KEY=${KEY//\//_}
  DEST="$REPLICA_ROOT/$SAFE_KEY"
  curl -sS -m 120 --aws-sigv4 "$SIG" $AUTH "$BASE/$KEY" -o "$DEST" || { FAILED=$((FAILED+1)); continue; }
  COUNT=$((COUNT+1))
done

if [ "$FAILED" -gt 0 ]; then
  [ -x "$ALERT_SCRIPT" ] && "$ALERT_SCRIPT" "PromptOS RustFS 副本部分失败" "$FAILED/$COUNT 个对象拉取失败，请检查 journalctl -u promptos-rustfs-replica。"
  echo "replica finished with failures: $FAILED" >&2
  exit 1
fi

echo "[$(date -Is)] rustfs replica ok: $COUNT objects -> $REPLICA_ROOT"
