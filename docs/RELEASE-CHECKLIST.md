# 发布检查清单 — PromptOS v<版本>（<日期>）

逐项勾选，全部通过才允许部署。

## 代码

- [ ] `docs\CHANGELOG.md` 已写本次版本条目
- [ ] `git status` 干净（无未提交改动、无未跟踪的临时文件）
- [ ] 已打 tag `v<版本>`

## 质量

- [ ] `gofmt -l .` 无输出（src\backend）
- [ ] `go vet ./...` 通过
- [ ] `go test ./...` 通过（集成测试带 `PROMPTOS_TEST_MYSQL_DSN` 等）
- [ ] 前端 `npm run build` 通过
- [ ] 冒烟：`start-dev.bat` 启动 + 首页 + 登录 + 发布一条 prompt

## 安全

- [ ] compose 无硬编码密码：`MYSQL_ROOT_PASSWORD`/`JWT_SECRET` 均为 `${VAR}` 注入
- [ ] 无真实密钥入库（`git diff` 检查；新配置先进 `.env.docker.example` 占位）
- [ ] 依赖无高危漏洞（`npm audit` / Go 依赖检查）

## 部署

- [ ] 服务器当前版本已备份（数据库 dump + uploads，位置：____，保留 3 版）
- [ ] 发布脚本执行成功（`scripts\release.ps1`）
- [ ] 健康检查通过：`curl https://<域名>/`
- [ ] 人工验证：首页 / 登录 / 发布 / 详情页
- [ ] 回滚步骤确认可用（`docs\DEPLOYMENT.md` 回滚章节）
