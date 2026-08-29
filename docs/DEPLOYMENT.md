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

- 版本：`20260829-a9ba2cf`（Git `a9ba2cf`）
- Compose 项目：`promptsystem`
- 发布目录：`/srv/releases/promptsystem/20260829-a9ba2cf`
- 入口端口：前端 `127.0.0.1:3092`，后端 `127.0.0.1:5092`
- 数据卷：`promptsystem_promptsystem_mysql_data`、`promptsystem_promptsystem_redis_data`、`promptsystem_promptsystem_uploads`
- 上传存储：当前使用独立 Docker 本地卷；未复用其他站点的 RustFS 凭据
- 健康检查：`/api/v1/health/ready` 返回 `storageMode=mysql`

## 回滚

1. compose 切回上一版镜像 tag / release 目录；
2. 必要时恢复 mysql_data 备份；
3. CHANGELOG 记录回滚事件。

## 依赖服务

- MySQL 8.4：`mysql_data` 卷，备份策略 <待补：定时 mysqldump 脚本>；
- Redis 7：缓存，可丢失重建。
