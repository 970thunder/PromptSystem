# AGENTS.md — AI 协作规约（PromptSystem / PromptOS）

> 任何 AI 助手在本仓库工作前必读。总规范见 `E:\Web\GOVERNANCE.md`。

## 项目是什么

PromptOS：AI Prompt & Skill 社区平台 MVP（feed、详情、登录注册、发布、搜索、点赞/收藏/评论）。前端 Vue3+TS+Vite+Tailwind（`src\frontend`），后端 Go net/http + MySQL + Redis（`src\backend`）。已部署服务器，未公开。

## 常用命令

| 用途 | 命令 |
|---|---|
| 依赖服务（MySQL/Redis） | `docker compose up -d mysql redis` |
| 本地一键启动 | `start-dev.bat`（端口 28301–28304，被占用即拒绝启动） |
| 后端静态检查 | `gofmt -l . && go vet ./...`（在 `src\backend`） |
| 后端测试 | `go test ./...`（集成测试需 `PROMPTOS_TEST_MYSQL_DSN` 等环境变量） |
| 前端构建 | `cd src\frontend && npm run build` |

## 已知问题（接到相关任务时优先处理）

- **前端零测试且不在 CI**——新增前端功能时必须至少补一条组件/冒烟测试。
- `docker-compose.yml` 硬编码 `MYSQL_ROOT_PASSWORD: root` 与默认 `JWT_SECRET`——任何触碰 compose 或安全相关的任务必须改为环境变量注入。
- `.grok\`、`.workbuddy\` 已加入 .gitignore，勿提交。

## 完成定义（DoD）

1. `gofmt` 无输出、`go vet` 通过、`go test ./...` 通过并贴输出；
2. 涉及前端的改动 `npm run build` 通过；
3. 数据库变更走 `sql\migrations\`，不直接改 `schema.sql`；
4. 新配置一律先进 `.env.docker.example` 占位，真实值不入库。

## 禁止事项

- 提交真实 `.env`、密钥、上传文件（`data\uploads\`）。
- 生产配置出现默认/弱密钥（`config_test.go` 已有 `TestValidateRejectsWeakProductionSecret`，不得删除）。
- 跳过/删除测试。

## 提交规范

conventional commits（近期历史均为 feat/fix/chore，保持一致）。

## 发布

生产为 docker compose 部署。发布前逐项勾选 `docs\RELEASE-CHECKLIST.md`（待补，模板在 `E:\Web\templates\`）。
