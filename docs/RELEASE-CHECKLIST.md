# 发布检查清单 — PromptOS（生产域名：promptsystem.isoumao.top）

逐项勾选，全部通过才允许部署。

## 代码

- [ ] `docs\CHANGELOG.md` 已写本次版本条目并记录镜像 digest（版本目录：`/srv/releases/promptsystem/<version>`）
- [ ] `git status` 干净（无未提交改动、无未跟踪的临时文件）
- [ ] 已打与发布版本完全一致的 tag

## 质量

- [ ] `gofmt -l .` 无输出（src\backend）
- [ ] `go vet ./...` 通过
- [ ] `go test ./...` 通过（集成测试带 `PROMPTOS_TEST_MYSQL_DSN` 等）
- [ ] 前端 `npm run build` 通过
- [ ] 冒烟：`start-dev.bat` 启动 + 首页 + 登录 + 发布一条 prompt

## 安全

- [ ] 生产 compose 无硬编码密码：数据库、JWT、Redis、OAuth、SMTP 均从 `/opt/secrets/promptsystem/app.env` 注入
- [ ] 无真实密钥入库（`git diff` 检查；新配置先进 `.env.docker.example` 占位）
- [ ] 依赖无高危漏洞（`npm audit --audit-level=high` / `govulncheck`；无法联网时记录原因）

## 部署

- [ ] 服务器当前版本已备份（数据库 dump + uploads，脚本输出位置和 SHA-256 已记录，保留 3 版）
- [ ] 发布脚本执行成功（`pwsh -File scripts/release.ps1 -Version <version>`；SSH `root@103.42.182.205:2680`）
- [ ] 健康检查通过：`https://promptsystem.isoumao.top/` 与 `https://promptsystem.isoumao.top/api/v1/health/ready`；后端 `127.0.0.1:5092`、前端 `127.0.0.1:3092`
- [ ] 人工验证：首页 / 登录 / 发布 / 详情页
- [ ] 回滚步骤确认可用：在 `/srv/releases/promptsystem/<previous-version>` 使用 Compose 项目名 `promptsystem` 执行 `docker compose -p promptsystem -f docker-compose.yml up -d`；必要时从 `/srv/backups/promptsystem/<version>/` 恢复 MySQL 与 uploads，禁止 `down -v`

## 固定生产参数

- 发布目录：`/srv/releases/promptsystem/<version>`；上一版本目录必须保留用于回滚。
- 备份目录：`/srv/backups/promptsystem/<version>/`，包含 `mysql.sql.gz`、`uploads.tar.gz` 和 `SHA256SUMS`；备份、镜像 load、迁移和重启在无 Swap 服务器上串行执行。
- 发布前记录 `free -h`、`df -h /`、`docker compose -p promptsystem ps`；发布后检查 ready、HTTPS 首页、登录、详情、评论、上传和 `docker compose -p promptsystem ps`。
- 服务器不编译源码，不执行全局 `docker prune`、`docker volume prune`、`down -v`，不停止其他 Compose 项目。
