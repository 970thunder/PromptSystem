# PromptOS 部署说明

> 环境差异走配置注入（.env / compose 环境变量），不维护第二套源码目录。密钥来源见「配置与密钥」。

## 环境一览

| 环境 | 位置 | 地址 | 说明 |
|---|---|---|---|
| 生产 | `103.42.182.205`（SSH `2680`） | https://promptsystem.isoumao.top | PromptSystem 独立 Compose 项目 |
| 本地 | 开发机 `E:\Web\PromptSystem` | http://localhost:28301–28304 | `start-dev.bat`，端口占用即拒绝启动 |

## 服务拓扑（docker-compose.yml）

| 服务 | 镜像 | 端口 | 数据卷 |
|---|---|---|---|
| mysql | mysql:8.4 | 3306（`PROMPTOS_*_PORT` 可覆盖） | mysql_data；由 backend 启动入口初始化基线并执行迁移 |
| redis | redis:7-alpine | 6379 | redis_data |
| backend | 本地构建（Go） | 8080 | uploads_data |
| frontend | 本地构建（nginx） | 3000→80 | — |

各服务自带 healthcheck。

## 部署步骤

1. 本地 `docs\RELEASE-CHECKLIST.md` 全绿；
2. 服务器备份：mysqldump 全库 + uploads 卷，保留最近 3 版；
3. 本机构建并标记镜像后通过 `docker save`/`docker load` 传输；服务器不编译源码；远端必须校验本地计算的归档 SHA-256，镜像加载后立即删除压缩包；
4. 验证：`curl https://<域名>/` + 首页登录/发布冒烟；
5. 失败回滚：切回上一版镜像/目录 + 恢复数据库备份。

### 服务器发布布局（推荐目标）

```
/opt/promptos/
├── releases/vX.Y.Z/      # 每版一份 compose + 镜像 tag 固定
├── shared/               # uploads、日志等跨版本数据
└── current -> releases/vX.Y.Z
```

## 配置与密钥

- 仓库内只有 `.env.docker.example`（占位符）；真实值本地在 `E:\Web\secrets\promptsystem\`，服务器在 `/opt/secrets/promptsystem/`（chmod 600，不在任何 webroot 下）。
- 生产密钥位于服务器 `/opt/secrets/promptsystem/app.env`（权限 `600`），不进入仓库；Compose 通过环境变量注入；
- 生产入口由现有 nginx 反代，证书由 Certbot webroot 自动续期；
- 生产前端 nginx 先以 `Content-Security-Policy-Report-Only` 观察 Vue、图片和 API 来源；确认无误后再切换强制 `Content-Security-Policy`，不得直接放宽为 `*`。
- 首次新库不需要人工导入 SQL：backend 检测到当前数据库无表时自动应用 `src/backend/sql/schema.sql` 基线，随后通过 `schema_migrations` 执行全部 `sql/migrations`。已有卷和部分迁移库只执行未记录的迁移，不会覆盖数据；

## 当前生产发布

- 版本：`20260830-b584585`（Git `b584585`）
- Compose 项目：`promptsystem`
- 发布目录：`/srv/releases/promptsystem/20260830-b584585`
- 入口端口：前端 `127.0.0.1:3092`，后端 `127.0.0.1:5092`
- 数据卷：`promptsystem_promptsystem_mysql_data`、`promptsystem_promptsystem_redis_data`、`promptsystem_promptsystem_uploads`
- 上传存储：当前使用独立 Docker 本地卷；未复用其他站点的 RustFS 凭据
- 健康检查：`/api/v1/health/ready` 返回 `storageMode=mysql`

截至 2026-09-04 的线上只读核验：上述版本仍在运行，内存可用约 `3.3 GiB`、根盘使用率约 `48%`、无 Swap；公网入口仍仅为 `80/443`，PromptOS 端口仍为 loopback。仓库中的新版生产 Compose 已加入只读根文件系统、`cap_drop`、PID/内存上限等加固项，但线上当前 release 尚未重启应用该策略，必须在下一次低峰发布中完成 `config`、启动、健康和 HTTPS 验收后，才能把 `S-11` 标记为生产完成。服务器实际 RustFS bridge 地址以 `ss -lntp` 为准，目前为 `172.21.0.1:13902`；PromptOS 当前尚未接入 RustFS，上传仍在本地卷。

## 回滚

1. compose 切回上一版镜像 tag / release 目录（`current` 只在健康检查通过后更新）；
2. 必要时恢复 mysql_data 备份；
3. CHANGELOG 记录回滚事件。

## 依赖服务

- MySQL 8.4：`promptsystem_promptsystem_mysql_data` 卷；每次发布前由 `scripts\release.ps1` 串行执行 `mysqldump --single-transaction --routines --events`，压缩后写入 `/srv/backups/promptsystem/<version>/mysql.sql.gz` 并生成 `SHA256SUMS`。
- Redis 7：缓存，可丢失重建；生产 Compose 使用独立 `REDIS_PASSWORD` 启用 `requirepass` 和 protected mode，backend 通过同名环境变量认证。密码必须只存在于 `/opt/secrets/promptsystem/app.env`（权限 `600`），不得复用其他站点凭据；发布前用 `docker compose ... config --quiet` 和 ready 检查确认认证生效。

### 上传配额与低内存预算

生产 backend 通过 `UPLOAD_MAX_MB`、`UPLOAD_MAX_CONCURRENT`、`UPLOAD_DAILY_QUOTA_MB` 和
`UPLOAD_TOTAL_QUOTA_MB` 限制单文件大小、同时解码数、单用户 UTC 日用量和所有未回收对象的总量。
默认值分别为 `10`、`4`、`100` 和 `2048`；服务器约 7.7 GiB 内存且无 Swap，调整并发上限前必须
确认发布后的可用内存仍不少于 2 GiB。日配额使用 Redis 原子字节计数；总容量以 `uploads` 表未
`trashed` 记录加进程内上传预留量核算，避免并发请求超卖。Redis 不可用时生产请求保护保持
fail-closed，开发/测试仅使用进程内计数用于回归。

### 生产数据库权限

- `promptos_app` 仅用于运行时业务 DML（SELECT/INSERT/UPDATE/DELETE/EXECUTE），不得执行 DDL。
- `promptos_migrator` 仅由启动迁移阶段使用，拥有 PromptOS 数据库 DDL 权限；两者密码由 `/opt/secrets/promptsystem/app.env` 注入，权限 `600`。
- 生产 Compose 必须设置 `MYSQL_USER`、`MYSQL_PASSWORD`、`MYSQL_MIGRATION_USER`、`MYSQL_MIGRATION_PASSWORD`，禁止使用 MySQL root 连接应用。

生产 Compose 的 backend 使用非 root 镜像用户、`no-new-privileges`、`cap_drop: ALL`、只读根文件系统、`/tmp` tmpfs、PID 及内存上限；frontend nginx 仅保留绑定 80 端口所需的 `NET_BIND_SERVICE`，其余能力丢弃并使用只读根文件系统。更新时必须先执行 `docker compose -f deploy/promptsystem/docker-compose.yml config`，再在低峰串行重启；MySQL/Redis 不得套用应用容器的只读策略。

### 低内存数据完整性审计

`backend` 镜像同时提供一次性命令 `/usr/local/bin/promptos-integrity-audit`，用于扫描
`comments`、`likes`、`favorites` 和 `reports` 的未知 `target_type` 与孤儿目标。命令只读数据库，
输出一行 JSON；审计无异常时退出 `0`，发现异常或无法连接数据库时退出非 `0`。它不自动删除数据，
异常记录必须由人工核对后通过迁移或受审查的修复脚本处理。

服务器约 7.7 GiB 内存且无 Swap，使用 systemd timer 或 cron 每日低峰串行执行，不要与备份、镜像
load、迁移、恢复演练或其他高内存任务并行。示例（沿用生产 Compose 项目名和密钥）：

```bash
cd /srv/releases/promptsystem/<current>
flock -n /run/lock/promptsystem-integrity-audit.lock \
  docker compose -p promptsystem --env-file /opt/secrets/promptsystem/app.env \
  run --rm --no-deps backend /usr/local/bin/promptos-integrity-audit
```

将标准输出和标准错误接入现有 systemd journal；timer 失败时必须触发服务器现有告警接收端，
并在发布记录中保留执行时间、JSON 结果和退出码。若告警接收端尚未配置，该项保持未验收，
不得用本地运行或 CI 通过替代生产证据。

同一 backend 镜像还提供 `/usr/local/bin/promptos-maintenance`，用于按需串行执行计数审计和上传回收：

```bash
flock -n /run/lock/promptsystem-maintenance.lock \
  docker compose -p promptsystem --env-file /opt/secrets/promptsystem/app.env \
  run --rm --no-deps backend /usr/local/bin/promptos-maintenance --task=all --older-than=24h
```

回收只处理数据库中状态为 `pending` 且超过阈值的上传；对象删除成功后才标记为 `trashed`，失败项
保留原记录并以非 `0` 退出，供下一次任务重试。计数审计发现漂移同样以非 `0` 退出，不自动改写计数。

### 个人数据与注销

生产发布必须同时包含以下受保护接口：

- `GET /api/v1/user/data-export`：导出本人账户、Prompt、收藏、点赞和浏览历史；不含密码哈希或 OAuth 标识。
- `DELETE /api/v1/user/history`：只清理本人的 `view_histories`，不回退累计浏览数。
- `DELETE /api/v1/user/account`：事务化清理个人互动/历史明细，禁用并匿名化用户、递增 `session_version`；Prompt、评论和上传记录按保留策略留存，禁用作者内容不再公开。

注销后的上传对象不会在请求内同步删除，统一交给低峰期 `promptos-maintenance` 延迟回收；这样可以在无 Swap
服务器上避免删除事务、对象 I/O 和应用请求争用内存。发布验收应使用测试账号验证旧 JWT 返回 `401 AUTH_USER_DISABLED`，
并复核当前与上一版 release 的回滚路径。

### 本次发布记录（2026-08-30）

- 版本：`20260830-b584585`；Compose 项目：`promptsystem`。
- 当前 release：`/srv/releases/promptsystem/20260830-b584585`；上一回滚版本：`20260829-a9ba2cf`。
- 备份：`/srv/backups/promptsystem/20260830-b584585/`，MySQL 与 uploads 均通过 SHA-256、gzip/tar 完整性校验。
- 发布验证：`/api/v1/health/ready` 返回 `200`、`storageMode=mysql`、`degraded=false`；HTTPS 首页、详情、评论分页和匿名浏览正常。

### 生产演示数据处置记录（2026-08-30）

- 处置前备份：`/srv/backups/promptsystem/20260830-demo-removal/`。
- 校验：`mysql.sql.gz` SHA-256 `cb3ecee7a1d10e7a407d7f246c4419bb43c7b71a1eb5ee2cfe86e7a75bb23d48`；`uploads.tar.gz` SHA-256 `9a2609e8613823970486698dbca7c9f8c3e07456dcac88d0749e1f403f8fd3ca`；`sha256sum -c`、`gzip -t`、`tar -tzf` 通过。
- 用户 ID 1-6 已禁用、密码置空并递增 `session_version`；固定演示密码登录验证 HTTP 401。Prompt ID 101-106 已转移至无密码官方归属账号 `PromptOS Official`（ID 7）。
- 数据库复核：1-6 为 `status=0,password_is_null=1,session_version=1`；ID 7 为 `status=1,password_is_null=1`；Prompt 101-106 为 `user_id=7,status=1`。
- 该处置不等同于完整恢复演练；临时库/临时卷恢复仍是 P0-12 未完成项。

### 首次恢复演练记录（2026-08-30）

- 使用 `/srv/backups/promptsystem/20260830-demo-removal/` 在临时 MySQL 卷恢复，数据复核 `users=6,prompts=6,published=6`；uploads 归档可解包。
- 使用上一版本镜像 `promptsystem-backend:20260829-a9ba2cf` 加入临时网络和 Redis，`/api/v1/health/ready` 返回 `200`、`storageMode=mysql`、`degraded=false`。
- 约 50 秒完成备份恢复到 ready（演练 RTO）；RPO 为备份时间点，当前尚未设置每日定时备份。
- 演练结束删除临时容器、网络、卷和 `/srv/tmp/promptos-restore-uploads-20260830`，服务器复核无残留。演练不影响 `promptsystem` 正式 Compose 项目。
