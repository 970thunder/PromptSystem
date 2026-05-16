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
| 当前聚焦模块 | **项目初始化 - 已完成** |
| 设计文档 | `prompt_platform_full_ai_dev_prd_and_prompt_pack.md` §21 |
| 源码状态 | 前端: `src/frontend/` · 后端: `src/backend/` |

> ⚠️ **每次开始新功能前，先更新"当前聚焦模块"字段。**
> 这是最高优先级的上下文，AI 据此决定生成什么文件、用什么路径。

---

## 2. 技术栈（生成代码时的强制约定）

### 前端
- **框架**：Vue 3 + TypeScript + Vite
- **样式**：TailwindCSS（禁止裸 CSS，除非 Tailwind 无法实现）
- **状态**：Pinia（禁止 Vuex）
- **路由**：Vue Router 4
- **组件库**：Naive UI（用户端） / Element Plus（管理端）
- **组件规范**：Composition API + `<script setup>` 语法，禁止 Options API

### 后端
- **框架**：Spring Boot 3
- **ORM**：MyBatis Plus（禁止裸 JDBC）
- **缓存**：Redis（用 `RedisTemplate`，禁止直接操作连接）
- **搜索**：ElasticSearch
- **存储**：MinIO
- **消息**：RabbitMQ

### AI 集成
- OpenAI · Claude · Gemini · OpenRouter · Dify · LangChain

---

## 3. 项目结构约定（生成文件时必须遵守路径）
src/
├── frontend/
│   ├── components/        # 通用组件（PascalCase 命名）
│   ├── views/             # 页面级组件（与路由 1:1 对应）
│   ├── stores/            # Pinia stores（camelCase 命名）
│   ├── api/               # API 调用层（按业务域分文件）
│   ├── types/             # TypeScript 类型定义
│   └── utils/             # 工具函数
├── backend/
│   ├── controller/        # REST 控制器
│   ├── service/           # 业务逻辑（接口 + impl 分离）
│   ├── mapper/            # MyBatis Plus Mapper
│   ├── entity/            # 数据库实体
│   ├── dto/               # 数据传输对象
│   └── config/            # 配置类
└── docs/

**命名规则速查：**
- Vue 组件文件：`PascalCase.vue`（如 `PromptCard.vue`）
- Pinia store 文件：`use{Domain}Store.ts`（如 `useUserStore.ts`）
- API 文件：`{domain}Api.ts`（如 `promptApi.ts`）
- 后端 Controller：`{Domain}Controller.java`
- 统一 API 前缀：`/api/v1`

---

## 4. 架构设计约定

**四层架构（生成功能时明确所属层）：**
1. **内容社区层** — 发布/搜索/评论/点赞/收藏/分类
2. **学习成长层** — 教程/技能体系/创作者等级 Lv1–Lv5
3. **工具运行层** — Playground/变量/Workflow/Agent
4. **商业化层** — 变现/企业版/API 平台

**Skill 定义**（生成 Skill 相关代码时使用此数据模型）：
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
| **1 MVP** | 首页·详情页·登录注册·发布·评论·点赞·收藏·搜索·分类 | 🚧 进行中 |
| **2** | 技能系统·教学·Prompt 变量·Playground | ⬜ 未开始 |
| **3** | 竞赛·企业版·Workflow·Agent·API 平台 | ⬜ 未开始 |

> 生成功能前先确认它属于哪个 Phase。**当前只实现 Phase 1 的内容**，不得提前为 Phase 2/3 生成代码。

---

## 6. 内联决策规则（AI 遇到以下场景直接执行，无需确认）

场景 → 行为
新建 Vue 组件       → Composition API + <script setup> + TypeScript
新建 API 调用       → 放入 src/frontend/api/{domain}Api.ts，用 axios 封装
新建后端接口        → Controller 只做路由，业务逻辑全在 Service
数据库查询          → 优先 MyBatis Plus 的 LambdaQueryWrapper，不写 SQL
需要缓存            → Redis，key 格式为 "业务域:实体ID"（如 "prompt:123"）
UI 组件选型         → 用户端用 Naive UI，管理端用 Element Plus
样式冲突            → Tailwind 优先，覆盖用 :deep() 而非 !important
错误处理            → 前端统一 try/catch + ElMessage，后端统一 GlobalExceptionHandler

---

## 7. 防错护栏（禁止项）

- ❌ 禁止在 Vue 文件中混用 Options API 和 Composition API
- ❌ 禁止在 Controller 层写业务逻辑
- ❌ 禁止硬编码环境变量（用 `.env` 文件）
- ❌ 禁止跨 Phase 提前实现功能（Phase 1 期间不写 Playground 代码）
- ❌ 禁止在无接口定义的情况下生成实现代码（先出 TypeScript interface / Java DTO）
- ❌ 禁止用 `any` 类型（必须明确 TypeScript 类型）

---

## 8. 标准输出格式（减少重复说明）

每次生成代码时，按此顺序输出：

1. **简述**：一句话说明本次生成的内容和所属层
2. **代码**：完整文件内容（带文件路径注释）
3. **关联**：列出需要同步修改的其他文件（如路由注册、store 引用）
4. **下一步**：建议的下一个子任务（不超过 3 条）

---