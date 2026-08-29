# 发布检查清单 — PromptOS（版本和日期由 `scripts/release.ps1` 注入）

逐项勾选，全部通过才允许部署。

## 代码

- [ ] `docs\CHANGELOG.md` 已写本次版本条目并记录镜像 digest
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
- [ ] 发布脚本执行成功（`scripts\release.ps1 -Version <版本>`）
- [ ] 健康检查通过：`https://promptsystem.isoumao.top/` 与 `/api/v1/health/ready`
- [ ] 人工验证：首页 / 登录 / 发布 / 详情页
- [ ] 回滚步骤确认可用：同一 Compose 项目名切回上一 release，必要时从备份恢复
