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
| mysql | mysql:8.4 | 3306（`PROMPTOS_*_PORT` 可覆盖） | mysql_data + `sql\schema.sql` 初始化 |
| redis | redis:7-alpine | 6379 | redis_data |
| backend | 本地构建（Go） | 8080 | uploads_data |
| frontend | 本地构建（nginx） | 3000→80 | — |

各服务自带 healthcheck。

## 部署步骤

1. 本地 `docs\RELEASE-CHECKLIST.md` 全绿；
2. 服务器备份：mysqldump 全库 + uploads 卷，保留最近 3 版；
3. 本机构建并标记镜像后通过 `docker save`/`docker load` 传输；服务器不编译源码；
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
- 首次新库需要先导入 `src/backend/sql/schema.sql`，之后后端启动时自动运行 `sql/migrations`；

## 当前生产发布

- 版本：`20260830-b584585`（Git `b584585`）
- Compose 项目：`promptsystem`
- 发布目录：`/srv/releases/promptsystem/20260830-b584585`
- 入口端口：前端 `127.0.0.1:3092`，后端 `127.0.0.1:5092`
- 数据卷：`promptsystem_promptsystem_mysql_data`、`promptsystem_promptsystem_redis_data`、`promptsystem_promptsystem_uploads`
- 上传存储：当前使用独立 Docker 本地卷；未复用其他站点的 RustFS 凭据
- 健康检查：`/api/v1/health/ready` 返回 `storageMode=mysql`

## 回滚

1. compose 切回上一版镜像 tag / release 目录；
2. 必要时恢复 mysql_data 备份；
3. CHANGELOG 记录回滚事件。

## 依赖服务

- MySQL 8.4：`promptsystem_promptsystem_mysql_data` 卷；每次发布前由 `scripts\release.ps1` 串行执行 `mysqldump --single-transaction --routines --events`，压缩后写入 `/srv/backups/promptsystem/<version>/mysql.sql.gz` 并生成 `SHA256SUMS`。
- Redis 7：缓存，可丢失重建。

### 生产数据库权限

- `promptos_app` 仅用于运行时业务 DML（SELECT/INSERT/UPDATE/DELETE/EXECUTE），不得执行 DDL。
- `promptos_migrator` 仅由启动迁移阶段使用，拥有 PromptOS 数据库 DDL 权限；两者密码由 `/opt/secrets/promptsystem/app.env` 注入，权限 `600`。
- 生产 Compose 必须设置 `MYSQL_USER`、`MYSQL_PASSWORD`、`MYSQL_MIGRATION_USER`、`MYSQL_MIGRATION_PASSWORD`，禁止使用 MySQL root 连接应用。

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
