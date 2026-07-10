# Phase 1 TODO 闭环设计

## 背景

当前分支为 `feature/todo-completion`。项目规则明确当前阶段是 Phase 1 MVP，只允许实现 Prompt 社区 MVP 范围内的功能，不提前开发 Phase 2/3 的 Skill 运行、Playground、Workflow 执行、企业版、支付、Agent 等能力。

`TODO.md` 中仍有 Phase 1 未完成项，同时包含后续阶段和长期优化。用户已确认本轮只闭环 Phase 1 未完成项，不跨 Phase。

## 目标

本轮目标是完成 Phase 1 中边界明确、能用现有架构正式落地的 TODO，并同步勾选清单。具体范围：

- 配置 Git Hooks。
- 完成 UI 设计系统基础：Tailwind tokens、Naive UI 主题覆盖、基础组件复用规范。
- 首页增加 Skill 分类入口占位，只作为 Phase 2 入口预告。
- Prompt 详情页补齐 Few-shot 与 Workflow 展示。
- 发布、更新、评论链路增加基础敏感词检测。
- 复用现有 `reports` 表和评论举报风格，扩展 Prompt 举报入口。
- 修复 Markdown 预览的 XSS 风险。
- 保留并验证此前响应信封空数组修复。

## 非目标

本轮不做以下内容：

- ElasticSearch 接入。
- Skill 创建、编辑、详情、运行、Workflow 可视化。
- Prompt Playground、变量系统、Prompt 自动评分。
- AI 内容审核外部模型调用。
- 消息通知、系统通知的完整投递系统。
- 创作者中心完整后台、收益统计、转化率、用户增长分析。
- 企业版、支付、API 开放平台、Agent 系统、后台管理端。

AI 内容审核在 TODO 中标记为需决策：需要模型提供方、密钥、审核策略、误杀处理和人工复核流程，不能用静默放行或假实现伪装完成。

## 证据与复用路径

- 项目规则：`AGENTS.md`、`CLAUDE.md` 均约束当前 Phase 为 Phase 1 MVP。
- 前端栈：Vue 3 + TypeScript + Vite + TailwindCSS + Pinia + Naive UI。
- 后端栈：Go `net/http` + `internal/api` + `internal/store` + MySQL/Memory Store。
- API 响应信封：`src/backend/internal/api/server.go` 中 `apiResponse[T]`。
- Prompt store：`src/backend/internal/store/interfaces.go`、`mysql_prompts.go`、`memory_prompts.go`。
- 评论和举报：`src/backend/internal/store/comments.go`、`mysql_comments.go`、`reports` 表。
- Markdown 预览：`src/frontend/src/utils/markdownPreview.ts` 和 `PublishView.vue` 的 `v-html` 使用点。
- UI 基础：`src/frontend/tailwind.config.js`、`src/frontend/src/App.vue`。
- 首页入口：`src/frontend/src/views/HomeView.vue`。
- 详情页：`src/frontend/src/views/PromptDetailView.vue`。

## 架构设计

### 工程准入

在前端项目接入 Husky 与 lint-staged。Git Hook 只做提交前轻量可执行检查，不引入跨 Phase 工具链。后端测试仍通过 README 和本轮验证命令执行，避免提交钩子过慢。

### UI 设计系统

复用现有 Tailwind 与 Naive UI，不新增独立 UI 框架。将主题 token 收敛到 Tailwind `theme.extend` 和 `App.vue` 的 `themeOverrides`，让页面继续沿用现有类名和 Naive Provider。

基础组件 TODO 按“项目已使用 Naive UI + Tailwind utility 组件模式”闭环：补充复用规范和实际 token，而不是重建一套 Button/Input/Card 包装组件，避免重复造轮子。

### Skill 分类入口占位

首页保留 Prompt 分类为主流程，增加 Skill 分类入口占位区，读取现有 schema 中 type=2 的分类语义或使用前端静态占位。占位只展示方向，不跳转到 Skill 运行/创建页面。

### Prompt 详情结构化展示

Few-shot 和 Workflow 展示不新增 Phase 2 执行能力。前端从 Prompt 内容中解析轻量结构段落：

- Few-shot：展示示例输入/输出文本块。
- Workflow：展示步骤列表，只用于阅读说明。

若内容没有可解析结构，展示“未提供结构化示例/流程”的空状态，不伪造业务数据。

### 内容审核

新增 store 层敏感词校验能力，供 Prompt 发布/更新、评论创建复用。规则来源先使用项目内静态规则，覆盖明显的脚本注入、钓鱼、密钥泄露提示和非法内容关键词。

校验失败时后端返回明确 `400` 错误，前端按现有 request/message 机制显示错误。禁止失败静默通过。

### 举报系统

复用 `reports` 表、`Report` 类型和现有评论举报响应结构，扩展 Prompt 举报：

- 后端新增 Prompt 举报接口，要求 JWT。
- Memory 与 MySQL 实现保持幂等：同一用户对同一 Prompt 只能生成一条 pending 举报。
- 前端在详情页增加“举报 Prompt”入口，并复用 Naive UI dialog。

### XSS 防护

`renderMarkdownPreview` 继续先转义用户输入，再把允许的 Markdown 语法转换成受控 HTML。链接只允许 `http:`、`https:`、`mailto:` 和站内相对路径，其他协议输出为纯文本或无害链接。

同时补充前端单元级脚本或构建检查可覆盖的测试入口；若当前前端没有测试框架，则用 `npm run build` 和 ESLint 结果作为最低验证，并在 TODO 中记录测试缺口。

## 数据与迁移

本轮优先复用已有表结构：

- `reports` 已支持 `target_type` 为 `comment, prompt, skill`，可直接用于 Prompt 举报。
- `prompts` 暂不增加 Few-shot/Workflow 字段，避免把 Phase 2 Workflow 模型提前写入 schema。
- 敏感词规则使用代码内常量，不新增数据库表。

因此默认不新增迁移。若实现过程中发现当前 MySQL store 无法可靠写入 Prompt 举报，才补最小迁移，并同步 `schema.sql` 与 `migrations/README.md`。

## 错误处理

- 后端继续使用 `{ code, message, data }` 响应信封。
- 敏感词命中返回 `400`，message 明确指出内容不符合发布规范。
- 未登录举报返回 `401`。
- 重复举报返回 `200`，`applied=false`。
- Prompt 不存在返回 `404`。

## 安全设计

- 不在前端硬编码密钥或环境路径。
- 不引入外部 AI 审核服务和密钥。
- Markdown 预览不允许脚本、事件属性和危险协议。
- 后端敏感词检测在可信执行端执行，前端只做提示，不作为安全边界。
- 举报接口必须鉴权。

## 验收标准

- `TODO.md` 中本轮完成项同步勾选，无法完成项标记 `[!]` 并说明决策缺口。
- `go test ./...` 通过。
- `go build ./cmd/api` 通过。
- `npm run build` 通过。
- `npx eslint . --ext .vue,.js,.jsx,.cjs,.mjs,.ts,.tsx,.cts,.mts` 无 error。
- 浏览器验证首页、详情页、发布页关键路径可渲染，无新增 console error。
- Markdown 预览对 `javascript:` 链接和 `<script>` 输入不会生成可执行内容。
- Prompt 发布/评论命中敏感词时被后端拒绝。
- Prompt 举报重复提交保持幂等。

## 待决策项

- AI 内容审核：需要确认模型服务、密钥管理、审核等级、误杀处理、人工复核入口和日志保留策略。
- 消息/系统通知：需要确认通知类型、接收人、已读状态、投递方式和保留周期。
- 创作者中心数据分析：需要确认指标口径、数据归属、时间窗口和权限边界。
