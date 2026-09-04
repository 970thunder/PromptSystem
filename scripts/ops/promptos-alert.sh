#!/usr/bin/env bash
# 文件作用：PromptOS 服务器告警通道。通过阿里云邮件推送 SMTP 把告警发送到
# 指定接收端（客服邮箱）。凭据从 /opt/secrets/promptsystem/alert.env 读取，
# 不落日志、不进仓库。供备份失败告警与资源看门狗共同调用。
#
# 用法：promptos-alert.sh "主题" "正文"
set -euo pipefail

ALERT_ENV_FILE=${ALERT_ENV_FILE:-/opt/secrets/promptsystem/alert.env}
[ -r "$ALERT_ENV_FILE" ] || { echo "alert env missing: $ALERT_ENV_FILE" >&2; exit 1; }
# shellcheck disable=SC1090
source "$ALERT_ENV_FILE"

SUBJECT=${1:?usage: promptos-alert.sh SUBJECT [BODY]}
BODY=${2:-}
: "${ALERT_SMTP_HOST:?}" "${ALERT_SMTP_PORT:?}" "${ALERT_SMTP_USER:?}" "${ALERT_SMTP_PASSWORD:?}" "${ALERT_SMTP_FROM:?}" "${ALERT_TO:?}"

MAIL_FILE=$(mktemp)
trap 'rm -f "$MAIL_FILE"' EXIT
{
  echo "From: PromptOS Alert <${ALERT_SMTP_FROM}>"
  echo "To: <${ALERT_TO}>"
  echo "Subject: ${SUBJECT}"
  echo "Content-Type: text/plain; charset=UTF-8"
  echo "Date: $(date -R)"
  echo
  echo "${BODY}"
  echo
  echo "--"
  echo "host=$(hostname) time=$(date -Is)"
} > "$MAIL_FILE"

curl -sS --max-time 30 --ssl-reqd \
  --url "smtp://${ALERT_SMTP_HOST}:${ALERT_SMTP_PORT}" \
  --mail-from "${ALERT_SMTP_FROM}" \
  --mail-rcpt "${ALERT_TO}" \
  --user "${ALERT_SMTP_USER}:${ALERT_SMTP_PASSWORD}" \
  -T "$MAIL_FILE"
