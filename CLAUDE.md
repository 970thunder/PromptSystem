# CLAUDE.md — PromptOS 项目指南

<!--
 读取顺序：每次新对话请先读完本文件全部内容，再开始响应。
 输出语言：所有自然语言回复用中文，代码注释用英文。
 token 节省原则：遇到模糊需求，优先按本文件约定推断执行，
 而非反问。只有本文件无法覆盖的决策才确认。
-->

---

## 1. 项目快照（每次对话必读）

| 字段 | 值 |
|------|-----|
| 项目 | PromptOS — AI 提示词 & 技能社区平台 |
| 当前 Phase | **Phase 1 MVP** |
| 当前聚焦模块 | **Phase 1 — TODO 闭环：工程准入、设计系统、安全审核、详情展示** |
| 设计文档 | `prompt_platform_full_ai_dev_prd_and_prompt_pack.md` §二十（MVP 优先级） |
| 源码状态 | 前端: `src/frontend/` · 后端: `src/backend/`（Go） |
| 任务清单 | `TODO.md`（以代码为准，完成项须同步勾选） |

> ⚠️ **每次开始新功能前，先更新「当前聚焦模块」字段。**
> 这是最高优先级的上下文，AI 据此决定生成什么文件、用什么路径。

---

## 2. 技术栈（生成代码时的强制约定）

### 前端
- **框架**：Vue 3 + TypeScript + Vite
- **样式**：TailwindCSS（禁止裸 CSS，除非 Tailwind 无法实现）
- **状态**：Pinia（禁止 Vuex）
- **路由**：Vue Router 4
- **组件库**：Naive UI（用户端 MVP）；Element Plus 仅 Phase 3 管理端预留
- **组件规范**：Composition API + `<script setup>` 语法，禁止 Options API

### 后端（当前实现）
- **语言 / 运行时**：Go 1.25+，`net/http` + `http.ServeMux`
- **模块名**：`promptos-backend`（见 `src/backend/go.mod`）
- **数据库**：MySQL（`database/sql` + `go-sql-driver/mysql`），schema 在 `src/backend/sql/`
- **持久化抽象**：`internal/store` 接口；MySQL 不可用时降级内存 store
- **认证**：JWT（`internal/auth`），密码 `bcrypt`（`golang.org/x/crypto/bcrypt`）
- **文件存储**：本地目录或 Cloudflare R2（`internal/storage`）
- **缓存**：Redis 已在 Docker Compose 中提供，**业务层尚未接入**（Phase 1 不要求）
- **搜索**：MySQL `LIKE` + 标签联表（`/api/v1/prompts/search`）；ElasticSearch 为 Phase 2+

### Phase 2+ 预留（当前不要生成）
- MinIO、RabbitMQ、ElasticSearch 集群、Spring/Java 栈 — **不在本仓库 MVP 路径**

### AI 集成（Phase 2+）
- OpenAI · Claude · Gemini · OpenRouter · Dify · LangChain

---

## 3. 项目结构约定（生成文件时必须遵守路径）

```text
src/
├── frontend/
│   ├── components/        # 通用组件（PascalCase 命名）
│   ├── views/             # 页面级组件（与路由 1:1 对应）
│   ├── stores/            # Pinia stores（camelCase 命名）
│   ├── api/               # API 调用层（按业务域分文件）
│   ├── types/             # TypeScript 类型定义
│   ├── mock/              # 前端 mock 降级数据
│   └── utils/             # 工具函数（如 request.ts）
└── backend/
    ├── cmd/api/           # 入口 main.go
    ├── internal/
    │   ├── api/           # HTTP 路由与 handler（server.go, auth.go, upload.go）
    │   ├── auth/          # JWT
    │   ├── config/        # 环境变量配置
    │   ├── database/      # MySQL 连接
    │   ├── storage/       # 图片上传（本地 / R2）
    │   └── store/         # 领域模型、store 接口、MySQL/内存实现
    └── sql/               # schema.sql 与 migrations/
```

**命名规则速查：**
- Vue 组件文件：`PascalCase.vue`（如 `PromptCard.vue`）
- Pinia store 文件：`camelCase.ts`（如 `user.ts`，导出 `useUserStore`）
- API 文件：`{domain}Api.ts`（如 `promptApi.ts`）
- Go 包：按目录分包；handler 方法挂在 `internal/api` 的 `server` 上
- Store 接口：`internal/store/interfaces.go`（如 `PromptManager`、`UserManager`）
- 统一 API 前缀：`/api/v1`
- JSON 响应信封：`{ code, message, data }`

**已有 MVP 路由（新增接口须与此风格一致）：**
- `GET /api/v1/health`
- `GET /api/v1/categories`
- `GET|POST /api/v1/prompts`
- `GET /api/v1/prompts/search`
- `GET|PUT|DELETE /api/v1/prompts/:id`
- `POST /api/v1/prompts/:id/like`、`POST .../favorite`（需 JWT）
- `POST /api/v1/uploads/images`（需 JWT）
- `GET /api/v1/auth/github`、`GET /api/v1/auth/github/callback`（GitHub OAuth）
- `POST /api/v1/user/login`、`POST /api/v1/user/register`
- `GET|PUT /api/v1/user/info`、`POST /api/v1/user/logout`（除 login/register 外需 JWT）

---

## 4. 架构设计约定

**四层架构（生成功能时明确所属层）：**
1. **内容社区层** — 发布/搜索/评论/点赞/收藏/分类
2. **学习成长层** — 教程/技能体系/创作者等级 Lv1–Lv5
3. **工具运行层** — Playground/变量/Workflow/Agent
4. **商业化层** — 变现/企业版/API 平台

**Go 后端分层（MVP 约定）：**
1. **Handler**（`internal/api`）— 解析请求、鉴权、校验、写 JSON 响应
2. **Store**（`internal/store`）— 业务与数据访问；MySQL 实现优先，保持接口可测
3. **禁止**在 handler 内写大段 SQL；SQL 放在 `mysql_*.go` 或迁移脚本中

**Skill 定义**（Phase 2 再实现，类型可先放在 `types/index.ts`）：
```typescript
interface Skill {
  prompt: string       // 核心提示词
  structure: object    // 结构化参数
  workflow: Step[]     // 执行步骤
  constraints: Rule[]  // 约束规则
}
```

**UI 风格**：暗色科技感 + 玻璃态（glassmorphism），参考 Civitai / Liblib AI 风格。
生成 CSS 时优先使用 `backdrop-filter`、半透明背景、霓虹色高亮。

---

## 5. 分阶段开发进度

| Phase | 核心功能 | 状态 |
|-------|---------|------|
| **1 MVP** | 首页·详情·登录注册·发布·搜索·分类·点赞收藏；**评论待做** | 🚧 进行中 |
| **2** | 技能系统·教学·Prompt 变量·Playground | ⬜ 未开始 |
| **3** | 竞赛·企业版·Workflow·Agent·API 平台·管理端 | ⬜ 未开始 |

> 生成功能前先确认它属于哪个 Phase。**当前只实现 Phase 1 的内容**，不得提前为 Phase 2/3 生成代码。

---

## 6. 内联决策规则（AI 遇到以下场景直接执行，无需确认）

| 场景 | 行为 |
|------|------|
| 新建 Vue 组件 | Composition API + `<script setup>` + TypeScript |
| 新建 API 调用 | 放入 `src/frontend/api/{domain}Api.ts`，用 axios 封装 |
| 新建后端接口 | 在 `internal/api/server.go` 注册路由；业务进 `internal/store` |
| 新建持久化逻辑 | 扩展 `store` 接口，实现 `mysql_*.go`，必要时补 `memory_*.go` |
| 数据库变更 | 新增 `src/backend/sql/migrations/NNNN_description.sql`，并更新 `schema.sql` |
| 需要缓存 | Phase 1 可不接；若接入 Redis，key 格式 `业务域:实体ID` |
| UI 组件选型 | 用户端 Naive UI；`useMessage` / `useDialog` 做反馈 |
| 样式冲突 | Tailwind 优先，覆盖用 `:deep()` 而非 `!important` |
| 错误处理 | 前端 try/catch + Naive UI Message；后端 handler 内返回统一 `apiResponse` |

**环境变量：**
- 前端：`VITE_*`（如 `VITE_API_BASE_URL`）
- 后端：`PORT`、`JWT_SECRET`、`JWT_EXPIRE_HOURS`、`MYSQL_*`、`REDIS_*`、`UPLOAD_*`、`R2_*`、`ALLOWED_ORIGIN` 等（见 `internal/config`）

---

## 7. 防错护栏（禁止项）

- ❌ 禁止在 Vue 文件中混用 Options API 和 Composition API
- ❌ 禁止在 handler 内堆叠复杂业务（应下沉到 `store`）
- ❌ 禁止硬编码密钥与环境相关路径（用环境变量 / config）
- ❌ 禁止跨 Phase 提前实现功能（Phase 1 期间不写 Playground / Skill 运行代码）
- ❌ 禁止在无类型定义的情况下生成前端实现（先补 `src/frontend/src/types`）
- ❌ 禁止用 `any` 类型（必须明确 TypeScript 类型）
- ❌ 禁止新建 Spring Boot / Java / MyBatis 目录或依赖

---

## 8. 标准输出格式（减少重复说明）

每次生成代码时，按此顺序输出：

1. **简述**：一句话说明本次生成的内容和所属层
2. **代码**：完整文件内容（带文件路径注释）
3. **关联**：列出需要同步修改的其他文件（如路由注册、store 引用、migration）
4. **下一步**：建议的下一个子任务（不超过 3 条），并提醒更新 `TODO.md`

---
