# PromptOS 部署说明

> 环境差异走配置注入（.env / compose 环境变量），不维护第二套源码目录。密钥来源见「配置与密钥」。

## 环境一览

| 环境 | 位置 | 地址 | 说明 |
|---|---|---|---|
| 生产 | 与其余项目同一台服务器（见 `secrets\shared\` 服务器清单） | https://<域名，待登记> | 已部署，未公开 |
| 本地 | 开发机 `E:\Web\PromptSystem` | http://localhost:28301–28304 | `start-dev.bat`，端口占用即拒绝启动 |

## 服务拓扑（docker-compose.yml）

| 服务 | 镜像 | 端口 | 数据卷 |
|---|---|---|---|
| mysql | mysql:8.4 | 3306（`PROMPTOS_*_PORT` 可覆盖） | mysql_data + `sql\schema.sql` 初始化 |
| redis | redis:7-alpine | 6379 | redis_data |
| backend | 本地构建（Go） | 8080 | uploads_data |
| frontend | 本地构建（nginx） | 3000→80 | — |

各服务自带 healthcheck。

## 部署步骤（实际上服务器方式待登记后补全）

1. 本地 `docs\RELEASE-CHECKLIST.md` 全绿；
2. 服务器备份：mysqldump 全库 + uploads 卷，保留最近 3 版；
3. <待登记：服务器 `git pull` + `docker compose build && up -d`，还是本地构建镜像推送>；
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
- ⚠️ 待办（P1）：compose 中 `MYSQL_ROOT_PASSWORD: root` 与默认 `JWT_SECRET` 必须改为 `${VAR}` 注入；当前 compose 无 TLS/反代，生产入口由服务器侧反代承担（方式待登记）。

## 回滚

1. compose 切回上一版镜像 tag / release 目录；
2. 必要时恢复 mysql_data 备份；
3. CHANGELOG 记录回滚事件。

## 依赖服务

- MySQL 8.4：`mysql_data` 卷，备份策略 <待补：定时 mysqldump 脚本>；
- Redis 7：缓存，可丢失重建。
