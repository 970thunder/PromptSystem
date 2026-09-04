#!/usr/bin/env bash
# 文件作用：PromptOS 资源看门狗（完整迭代 TODO O-01）。检查磁盘 70/80/90 分级、
# 可用内存、容器状态、backend ready、证书剩余天数、RustFS 链路端口与每日备份
# 新鲜度；任一阈值越界通过告警通道发信。状态文件去重，仅在与上次级别不同时发信，
# 避免刷屏。设计为 systemd timer 每 15 分钟低频执行，内存开销可忽略。
set -uo pipefail

DOMAIN=${PROMPTOS_DOMAIN:-promptsystem.isoumao.top}
BACKEND_READY_URL=${PROMPTOS_READY_URL:-http://127.0.0.1:5092/api/v1/health/ready}
RUSTFS_PORTS=${PROMPTOS_RUSTFS_PORTS:-127.0.0.1:13900 172.21.0.1:13902}
BACKUP_ROOT=${PROMPTOS_BACKUP_ROOT:-/srv/backups/promptsystem/daily}
STATE_DIR=/var/lib/promptos-watchdog
ALERT_SCRIPT=/usr/local/bin/promptos-alert.sh
DISK_WARN=70
DISK_HIGH=80
DISK_CRITICAL=90
MEM_MIN_MB=500
CERT_MIN_DAYS=14
BACKUP_MAX_AGE_HOURS=26

mkdir -p "$STATE_DIR"

notify() { # notify <key> <subject> <body>
  local key="$1" subject="$2" body="$3"
  if [ "$(cat "$STATE_DIR/$key" 2>/dev/null || echo ok)" = "alerting" ]; then
    return 0
  fi
  echo alerting > "$STATE_DIR/$key"
  [ -x "$ALERT_SCRIPT" ] && "$ALERT_SCRIPT" "$subject" "$body"
}

resolve() { # resolve <key>：恢复后清除告警状态
  local key="$1"
  if [ "$(cat "$STATE_DIR/$key" 2>/dev/null || echo ok)" = "alerting" ]; then
    echo ok > "$STATE_DIR/$key"
  fi
}

FAIL=0

# 1) 磁盘分级（根盘）
DISK_PCT=$(df -P / | awk 'NR==2 {gsub("%","",$5); print $5}')
DISK_KEY="disk"
if [ "$DISK_PCT" -ge "$DISK_CRITICAL" ]; then
  notify "$DISK_KEY" "PromptOS 磁盘 ≥${DISK_CRITICAL}%（${DISK_PCT}%）" "根盘使用率 ${DISK_PCT}%，触发发布阻断线。请立即按 O-04 流程清理。"
  FAIL=1
elif [ "$DISK_PCT" -ge "$DISK_HIGH" ]; then
  notify "$DISK_KEY" "PromptOS 磁盘 ≥${DISK_HIGH}%（${DISK_PCT}%）" "根盘使用率 ${DISK_PCT}%，请安排清理悬空镜像与过期 build cache。"
  FAIL=1
elif [ "$DISK_PCT" -ge "$DISK_WARN" ]; then
  notify "$DISK_KEY" "PromptOS 磁盘 ≥${DISK_WARN}%（${DISK_PCT}%）" "根盘使用率 ${DISK_PCT}%，进入观察期。"
  FAIL=1
else
  resolve "$DISK_KEY"
fi

# 2) 可用内存
MEM_AVAIL_MB=$(free -m | awk '/^Mem:/ {print $7}')
if [ "$MEM_AVAIL_MB" -lt "$MEM_MIN_MB" ]; then
  notify "mem" "PromptOS 可用内存不足" "可用内存 ${MEM_AVAIL_MB}MB < ${MEM_MIN_MB}MB（无 Swap，注意串行约束）。"
  FAIL=1
else
  resolve "mem"
fi

# 3) 容器状态
for C in promptsystem-backend promptsystem-frontend promptsystem-mysql promptsystem-redis; do
  STATE=$(docker inspect -f '{{.State.Status}} {{.RestartCount}}' "$C" 2>/dev/null || echo "missing 0")
  STATUS=${STATE%% *}
  if [ "$STATUS" != "running" ]; then
    notify "container-$C" "PromptOS 容器异常：$C" "状态：$STATE。请 docker logs 排查。"
    FAIL=1
  else
    resolve "container-$C"
  fi
done

# 4) backend ready
READY=$(curl -s -o /dev/null -m 5 -w '%{http_code}' "$BACKEND_READY_URL" || echo 000)
if [ "$READY" != "200" ]; then
  notify "ready" "PromptOS backend ready 异常" "$BACKEND_READY_URL 返回 $READY。"
  FAIL=1
else
  resolve "ready"
fi

# 5) 证书剩余天数
EXPIRY_EPOCH=$(echo | timeout 10 openssl s_client -servername "$DOMAIN" -connect "$DOMAIN:443" 2>/dev/null | openssl x509 -noout -enddate 2>/dev/null | cut -d= -f2)
if [ -n "$EXPIRY_EPOCH" ]; then
  EXPIRY_DAYS=$(( ($(date -d "$EXPIRY_EPOCH" +%s) - $(date +%s)) / 86400 ))
  if [ "$EXPIRY_DAYS" -lt "$CERT_MIN_DAYS" ]; then
    notify "cert" "PromptOS 证书即将到期" "$DOMAIN 证书剩余 ${EXPIRY_DAYS} 天，请执行 certbot renew 并 reload nginx（O-06）。"
    FAIL=1
  else
    resolve "cert"
  fi
fi

# 6) RustFS 链路端口
for P in $RUSTFS_PORTS; do
  H=${P%%:*}; PORT=${P##*:}
  if ! timeout 3 bash -c "</dev/tcp/$H/$PORT" 2>/dev/null; then
    notify "rustfs-$PORT" "PromptOS RustFS 链路不可达" "$H:$PORT 连接失败（O-07）。"
    FAIL=1
  else
    resolve "rustfs-$PORT"
  fi
done

# 7) 每日备份新鲜度（D-11 的第二道保险）
LATEST_BACKUP=$(find "$BACKUP_ROOT" -mindepth 1 -maxdepth 1 -type d -name '2*' 2>/dev/null | sort | tail -1)
if [ -z "$LATEST_BACKUP" ]; then
  notify "backup" "PromptOS 尚无每日备份" "$BACKUP_ROOT 为空，请检查 promptos-backup.timer。"
  FAIL=1
else
  AGE_H=$(( ($(date +%s) - $(stat -c %Y "$LATEST_BACKUP")) / 3600 ))
  if [ "$AGE_H" -gt "$BACKUP_MAX_AGE_HOURS" ]; then
    notify "backup" "PromptOS 每日备份过期" "最新备份 $LATEST_BACKUP 已 ${AGE_H} 小时（>${BACKUP_MAX_AGE_HOURS}h），请检查 promptos-backup.timer 与告警。"
    FAIL=1
  else
    resolve "backup"
  fi
fi

echo "[$(date -Is)] watchdog done, fail=$FAIL (disk=${DISK_PCT}% mem_avail=${MEM_AVAIL_MB}MB ready=$READY)"
exit $FAIL
