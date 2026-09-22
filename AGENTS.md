# AGENTS.md — AI 协作规约（PromptSystem / PromptOS）

> 任何 AI 助手在本仓库工作前必读。总规范见 `E:\Web\GOVERNANCE.md`。

## 项目是什么

PromptOS：AI Prompt & Skill 社区平台 MVP（feed、详情、登录注册、发布、搜索、点赞/收藏/评论）。前端 Vue3+TS+Vite+Tailwind（`src\frontend`），后端 Go net/http + MySQL + Redis（`src\backend`）。已部署服务器，未公开。

| 项目快照 | 当前值 |
|---|---|
| 当前 Phase | PromptOS 1.0 内容社区上线重构 |
| 当前聚焦模块 | MVP 全页面与功能质量修复 |
| 任务清单 | `TODO.md`（以代码和测试结果为准） |

## 常用命令

| 用途 | 命令 |
|---|---|
| 依赖服务（MySQL/Redis） | `docker compose up -d mysql redis` |
| 本地一键启动 | `start-dev.bat`（端口 28301–28304，被占用即拒绝启动） |
| 后端静态检查 | `gofmt -l . && go vet ./...`（在 `src\backend`） |
| 后端测试 | `go test ./...`（集成测试需 `PROMPTOS_TEST_MYSQL_DSN` 等环境变量） |
| 前端构建 | `cd src\frontend && npm run build` |

## 已知问题（接到相关任务时优先处理）

- 前端测试已接入本地 Vitest；新增前端功能时必须至少补一条组件/冒烟测试，并在 CI 中执行构建与测试。
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

## 开发与交付节奏

详细流程见 [`docs/DEVELOPMENT-WORKFLOW.md`](docs/DEVELOPMENT-WORKFLOW.md)。以下规则是强制门禁：

- 每个可独立交付的功能都必须同时完成代码、测试和必要文档；前端页面/交互至少补测试，后端安全、上传、权限和数据变更至少补失败路径测试。
- 修改完成后先运行对应的 lint、单元/集成测试、构建和契约检查；涉及页面或联调时追加 Playwright 与真实依赖验证。
- 测试全部通过后立即使用 Conventional Commits 提交，并马上推送当前分支；不得把已验证改动长期留在本地。
- 测试失败、依赖不可用或验收范围不完整时，不得报告为完成；允许提交的部分必须在提交说明和交付记录中写明限制。
- 每次提交后确认 `git status --short --branch` 和 `git log -1 --oneline`，并在交付说明中记录提交号、推送分支和测试结果。
