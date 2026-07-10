# Phase 1 TODO Completion Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Close the approved Phase 1 TODO items without implementing Phase 2/3 runtime features.

**Architecture:** Keep the current Vue 3 + Pinia + Naive UI frontend and Go `net/http` backend split. Put HTTP parsing in `internal/api`, moderation/report business rules in `internal/store`, and visual-only Phase 2 placeholders in frontend views.

**Tech Stack:** Go 1.25+, `net/http`, MySQL/memory store, Vue 3, TypeScript, Vite, TailwindCSS, Pinia, Naive UI, Husky, lint-staged.

---

## File Structure

- Modify `AGENTS.md` and `CLAUDE.md`: update current focus to Phase 1 TODO completion.
- Modify `TODO.md`: mark completed Phase 1 items and mark AI moderation/notification/analytics items as decision-gated when appropriate.
- Modify `src/frontend/package.json` and `src/frontend/package-lock.json`: add Husky/lint-staged scripts and dev dependencies.
- Create `src/frontend/.husky/pre-commit`: run lint-staged from the frontend package.
- Modify `src/frontend/tailwind.config.js`: consolidate tokens for color, radius, shadow, motion, spacing, and screens.
- Modify `src/frontend/src/App.vue`: centralize Naive UI theme overrides from the same visual direction.
- Modify `src/frontend/src/views/HomeView.vue`: add Skill category placeholder section.
- Create `src/frontend/src/utils/promptStructure.ts`: parse read-only Few-shot and Workflow blocks from prompt content.
- Modify `src/frontend/src/types/index.ts`: add display-only structure types.
- Modify `src/frontend/src/views/PromptDetailView.vue`: display Few-shot, Workflow, and Prompt report action.
- Modify `src/frontend/src/utils/markdownPreview.ts`: harden markdown preview output against unsafe protocols.
- Modify `src/frontend/src/api/promptApi.ts` and `src/frontend/src/stores/prompt.ts`: add Prompt report API wrapper.
- Create `src/backend/internal/store/moderation.go`: shared moderation checks.
- Create `src/backend/internal/store/moderation_test.go`: moderation unit tests.
- Modify `src/backend/internal/store/prompts.go`: enforce moderation in memory prompt create/update.
- Modify `src/backend/internal/store/mysql_prompts.go`: enforce moderation in MySQL prompt create/update and add Prompt report persistence.
- Modify `src/backend/internal/store/comments.go` and `mysql_comments.go`: enforce moderation in comment creation and reuse report lookup helpers.
- Modify `src/backend/internal/store/interfaces.go`: add `ReportPrompt` to `PromptManager`.
- Modify `src/backend/internal/api/server.go`: add Prompt report route and call moderation-protected store methods.
- Create or modify backend tests under `src/backend/internal/store/*_test.go` and `src/backend/internal/api/*_test.go` for response/report/moderation contracts.

---

### Task 1: Protect Existing Baseline And Focus Metadata

**Files:**
- Modify: `AGENTS.md`
- Modify: `CLAUDE.md`
- Keep: `src/backend/internal/api/server.go`
- Keep: `src/backend/internal/api/response_test.go`
- Keep: `src/frontend/package-lock.json`

- [ ] **Step 1: Verify branch and working tree**

Run:

```powershell
git status --short --branch
```

Expected: branch is `feature/todo-completion`; existing modified files remain visible.

- [ ] **Step 2: Update focus metadata**

Set current focus in both rule files to:

```markdown
| 当前聚焦模块 | **Phase 1 — TODO 闭环：工程准入、设计系统、安全审核、详情展示** |
```

- [ ] **Step 3: Keep response contract test in scope**

Confirm `src/backend/internal/api/response_test.go` contains:

```go
// Package api tests the HTTP response envelope contract.
package api

func TestAPIResponseIncludesEmptySliceData(t *testing.T) {
    payload, err := json.Marshal(apiResponse[[]store.Comment]{
        Code:    200,
        Message: "Success",
        Data:    []store.Comment{},
    })
    if err != nil {
        t.Fatalf("Marshal() error = %v", err)
    }

    body := string(payload)
    if !strings.Contains(body, `"data":[]`) {
        t.Fatalf("expected empty slice data in response envelope, got %s", body)
    }
}
```

- [ ] **Step 4: Run backend response test**

Run:

```powershell
cd src/backend
go test ./internal/api -run TestAPIResponseIncludesEmptySliceData -count=1
```

Expected: PASS.

---

### Task 2: Configure Git Hooks

**Files:**
- Modify: `src/frontend/package.json`
- Modify: `src/frontend/package-lock.json`
- Create: `src/frontend/.husky/pre-commit`
- Modify: `TODO.md`

- [ ] **Step 1: Install hook dependencies**

Run:

```powershell
cd src/frontend
npm install --save-dev husky lint-staged
```

Expected: `package.json` and `package-lock.json` include Husky and lint-staged.

- [ ] **Step 2: Add scripts and lint-staged config**

Patch `src/frontend/package.json` scripts/config:

```json
{
  "scripts": {
    "dev": "vite",
    "build": "vue-tsc && vite build",
    "preview": "vite preview",
    "lint": "eslint . --ext .vue,.js,.jsx,.cjs,.mjs,.ts,.tsx,.cts,.mts --fix",
    "lint:check": "eslint . --ext .vue,.js,.jsx,.cjs,.mjs,.ts,.tsx,.cts,.mts",
    "format": "prettier --write src/",
    "prepare": "husky"
  },
  "lint-staged": {
    "*.{vue,js,jsx,cjs,mjs,ts,tsx,cts,mts}": "eslint --fix",
    "*.{vue,js,jsx,cjs,mjs,ts,tsx,cts,mts,json,css,md}": "prettier --write"
  }
}
```

- [ ] **Step 3: Create pre-commit hook**

Create `src/frontend/.husky/pre-commit`:

```sh
#!/usr/bin/env sh
cd "$(dirname "$0")/.."
npx lint-staged
```

- [ ] **Step 4: Verify hook command directly**

Run:

```powershell
cd src/frontend
npx lint-staged --allow-empty
```

Expected: exits 0.

- [ ] **Step 5: Update checklist**

In `TODO.md`, mark `配置 Git Hooks` as `[x]`.

---

### Task 3: Consolidate UI Design System Tokens

**Files:**
- Modify: `src/frontend/tailwind.config.js`
- Modify: `src/frontend/src/App.vue`
- Modify: `TODO.md`

- [ ] **Step 1: Update Tailwind tokens**

Keep existing class consumers working and extend tokens:

```js
colors: {
  brand: {
    ink: '#111111',
    muted: '#555555',
    subtle: '#777777',
    line: 'rgba(0, 0, 0, 0.1)'
  },
  surface: {
    page: '#f5f3ee',
    card: '#ffffff',
    soft: '#faf8f4',
    wash: '#f6f4ef',
    inverse: '#111111'
  },
  accent: {
    success: '#22a06b',
    warning: '#b7791f',
    danger: '#c0392b'
  }
}
```

- [ ] **Step 2: Set radius, shadow, motion, responsive tokens**

Add:

```js
borderRadius: {
  sm: '8px',
  md: '12px',
  lg: '16px',
  xl: '20px',
  card: '20px',
  button: '9999px',
  input: '16px'
},
boxShadow: {
  panel: '0 16px 40px rgba(15, 23, 42, 0.06)',
  panelHover: '0 20px 48px rgba(15, 23, 42, 0.1)',
  focus: '0 0 0 3px rgba(17, 17, 17, 0.12)'
},
transitionTimingFunction: {
  standard: 'cubic-bezier(0.2, 0, 0, 1)'
}
```

- [ ] **Step 3: Update Naive theme overrides**

In `App.vue`, keep `NConfigProvider` and adjust `themeOverrides.common` to match the Tailwind tokens:

```ts
const themeOverrides = computed(() => ({
  common: {
    primaryColor: '#111111',
    primaryColorHover: '#333333',
    primaryColorPressed: '#000000',
    bodyColor: '#f5f3ee',
    cardColor: '#ffffff',
    modalColor: '#ffffff',
    inputColor: '#faf8f4',
    borderColor: 'rgba(0,0,0,0.1)',
    borderRadius: '16px',
    textColorBase: '#111111'
  }
}))
```

- [ ] **Step 4: Update checklist**

Mark UI design system items under `2.1 全局设计` as `[x]`. Mark `2.2 基础组件` as `[x]` only when each item is satisfied by Naive UI + Tailwind token reuse, not by new wrapper components.

- [ ] **Step 5: Verify frontend build**

Run:

```powershell
cd src/frontend
npm run build
```

Expected: PASS.

---

### Task 4: Harden Markdown Preview

**Files:**
- Modify: `src/frontend/src/utils/markdownPreview.ts`
- Modify: `src/frontend/src/views/PublishView.vue`

- [ ] **Step 1: Add safe URL handling**

Replace link rendering helper with explicit protocol validation:

```ts
const allowedLinkProtocols = ['http:', 'https:', 'mailto:']

function sanitizeLinkUrl(rawUrl: string): string {
  const trimmed = rawUrl.trim()
  if (trimmed.startsWith('/') && !trimmed.startsWith('//')) {
    return trimmed
  }

  try {
    const url = new URL(trimmed)
    return allowedLinkProtocols.includes(url.protocol) ? url.href : '#'
  } catch {
    return '#'
  }
}
```

- [ ] **Step 2: Escape generated attributes**

Use:

```ts
function renderSafeLink(label: string, rawUrl: string): string {
  const safeLabel = escapeHtml(label)
  const safeUrl = escapeHtml(sanitizeLinkUrl(rawUrl))
  return `<a href="${safeUrl}" class="underline" target="_blank" rel="noopener noreferrer">${safeLabel}</a>`
}
```

- [ ] **Step 3: Keep source escaping before markdown transforms**

The final implementation must still call `escapeHtml(source)` before converting markdown syntax.

- [ ] **Step 4: Add a local verification command**

Run:

```powershell
cd src/frontend
npm run build
```

Expected: PASS and no type errors.

---

### Task 5: Add Backend Moderation Rules

**Files:**
- Create: `src/backend/internal/store/moderation.go`
- Create: `src/backend/internal/store/moderation_test.go`
- Modify: `src/backend/internal/store/prompts.go`
- Modify: `src/backend/internal/store/mysql_prompts.go`
- Modify: `src/backend/internal/store/comments.go`
- Modify: `src/backend/internal/store/mysql_comments.go`

- [ ] **Step 1: Write failing moderation tests**

Create tests:

```go
func TestModerationRejectsUnsafePromptContent(t *testing.T) {
    input := CreatePromptInput{
        Title: "合法标题",
        Description: "合法描述",
        Content: "请帮我做一个钓鱼链接",
        SystemPrompt: "",
        Model: "GPT-4.1",
        CategoryID: 1,
    }

    err := ValidatePromptModeration(input)
    if err == nil {
        t.Fatal("expected moderation error")
    }
}

func TestModerationAllowsNormalPromptContent(t *testing.T) {
    input := CreatePromptInput{
        Title: "产品海报",
        Description: "生成电商主图",
        Content: "为一款耳机生成干净的产品海报提示词",
        Model: "Midjourney v6",
        CategoryID: 1,
    }

    if err := ValidatePromptModeration(input); err != nil {
        t.Fatalf("expected normal content to pass, got %v", err)
    }
}
```

- [ ] **Step 2: Run tests and confirm failure**

Run:

```powershell
cd src/backend
go test ./internal/store -run TestModeration -count=1
```

Expected: FAIL because `ValidatePromptModeration` is not defined.

- [ ] **Step 3: Implement moderation helper**

Create `moderation.go`:

```go
// Package store 提供社区内容的领域校验与持久化抽象。
package store

import (
    "fmt"
    "strings"
)

type moderationField struct {
    Name  string
    Value string
}

var sensitiveRules = []string{
    "<script",
    "javascript:",
    "onerror=",
    "api_key",
    "begin private key",
    "钓鱼链接",
    "盗号",
}

func ValidatePromptModeration(input CreatePromptInput) error {
    return validateModerationFields([]moderationField{
        {Name: "标题", Value: input.Title},
        {Name: "描述", Value: input.Description},
        {Name: "提示词正文", Value: input.Content},
        {Name: "系统提示词", Value: input.SystemPrompt},
        {Name: "模型", Value: input.Model},
    })
}

func ValidateCommentModeration(content string) error {
    return validateModerationFields([]moderationField{{Name: "评论", Value: content}})
}

func validateModerationFields(fields []moderationField) error {
    for _, field := range fields {
        normalized := strings.ToLower(strings.TrimSpace(field.Value))
        for _, rule := range sensitiveRules {
            if strings.Contains(normalized, rule) {
                return fmt.Errorf("%s包含不符合平台规范的内容", field.Name)
            }
        }
    }
    return nil
}
```

- [ ] **Step 4: Enforce in prompt stores**

At the start of memory `CreatePrompt`, `UpdatePrompt`, MySQL `Create`, and MySQL `Update`, add:

```go
if err := ValidatePromptModeration(input); err != nil {
    return Prompt{}, err
}
```

- [ ] **Step 5: Enforce in comment validation**

In `validateCommentInput`, after content length validation:

```go
if err := ValidateCommentModeration(content); err != nil {
    return err
}
```

- [ ] **Step 6: Verify moderation tests**

Run:

```powershell
cd src/backend
go test ./internal/store -run TestModeration -count=1
```

Expected: PASS.

---

### Task 6: Extend Prompt Report System

**Files:**
- Modify: `src/backend/internal/store/interfaces.go`
- Modify: `src/backend/internal/store/prompts.go`
- Modify: `src/backend/internal/store/memory_prompts.go`
- Modify: `src/backend/internal/store/mysql_prompts.go`
- Modify: `src/backend/internal/api/server.go`
- Modify: `src/frontend/src/types/index.ts`
- Modify: `src/frontend/src/api/promptApi.ts`
- Modify: `src/frontend/src/stores/prompt.ts`
- Modify: `src/frontend/src/views/PromptDetailView.vue`

- [ ] **Step 1: Add store contract**

Extend `PromptManager`:

```go
Report(id int, userID int, reason string, detail string) (Report, bool, error)
```

- [ ] **Step 2: Write memory report behavior**

Add to prompt memory path:

```go
func ReportPrompt(id int, userID int, reason string, detail string) (Report, bool, error) {
    if _, ok := FindPromptByID(id); !ok {
        return Report{}, false, fmt.Errorf("prompt not found")
    }
    input := ReportPromptInput{PromptID: id, UserID: userID, Reason: reason, Detail: detail}
    return reportPrompt(input)
}
```

The internal helper must use key `prompt:{userID}:{promptID}` and return `applied=false` on duplicates.

- [ ] **Step 3: Add MySQL report implementation**

In `MySQLPromptStore.Report`, insert:

```go
INSERT IGNORE INTO reports (user_id, target_type, target_id, reason, detail, status)
VALUES (?, 'prompt', ?, ?, ?, 'pending')
```

Then read back the report with `target_type='prompt'`.

- [ ] **Step 4: Add API route**

In `handlePromptDetail`, add `"report"` under extra path parts:

```go
case "report":
    if r.Method != http.MethodPost {
        writeMethodNotAllowed(w)
        return
    }
    s.withAuth(func(w http.ResponseWriter, r *http.Request) {
        s.handlePromptReport(w, r, id)
    }).ServeHTTP(w, r)
    return
```

- [ ] **Step 5: Add handler**

Add:

```go
func (s *server) handlePromptReport(w http.ResponseWriter, r *http.Request, id int) {
    var payload reportCommentPayload
    if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
        writeJSON(w, http.StatusBadRequest, apiResponse[any]{Code: 400, Message: "Invalid request body"})
        return
    }

    userID, ok := userIDFromContext(r.Context())
    if !ok {
        writeJSON(w, http.StatusUnauthorized, apiResponse[any]{Code: 401, Message: "Unauthorized"})
        return
    }

    report, applied, err := s.promptStore.Report(id, userID, payload.Reason, payload.Detail)
    if err != nil {
        status := http.StatusBadRequest
        if err.Error() == "prompt not found" {
            status = http.StatusNotFound
        }
        writeJSON(w, status, apiResponse[any]{Code: status, Message: err.Error()})
        return
    }

    writeJSON(w, http.StatusOK, apiResponse[reportActionResponse]{
        Code: 200,
        Message: "Success",
        Data: reportActionResponse{Report: report, Applied: applied},
    })
}
```

- [ ] **Step 6: Add frontend API wrapper**

In `promptApi.ts`:

```ts
reportPrompt(
  id: number,
  data: { reason: string; detail?: string }
): Promise<ApiResponse<ReportActionResponse>> {
  return request.post(`/prompts/${id}/report`, data)
}
```

- [ ] **Step 7: Add detail page action**

In `PromptDetailView.vue`, add a secondary button near share:

```vue
<button
  class="detail-btn-share"
  :disabled="promptReporting"
  @click="handlePromptReport"
>
  举报
</button>
```

Handler:

```ts
const handlePromptReport = async () => {
  if (!prompt.value || promptReporting.value) {
    return
  }
  if (!(await ensureAuthenticated())) {
    return
  }
  dialog.create({
    title: '举报提示词',
    content: '确认举报这条提示词？系统会记录为待处理状态。',
    positiveText: '确认举报',
    negativeText: '取消',
    onPositiveClick: async () => {
      promptReporting.value = true
      try {
        const response = await promptStore.reportPrompt(prompt.value!.id, {
          reason: '不当提示词',
          detail: prompt.value!.title.slice(0, 200)
        })
        message.success(response.applied ? '已提交举报' : '你已经举报过这条提示词')
      } finally {
        promptReporting.value = false
      }
    }
  })
}
```

- [ ] **Step 8: Verify backend store tests**

Run:

```powershell
cd src/backend
go test ./internal/store -run "Test.*Report" -count=1
```

Expected: PASS after adding tests for duplicate Prompt reports.

---

### Task 7: Add Few-Shot And Workflow Display

**Files:**
- Create: `src/frontend/src/utils/promptStructure.ts`
- Modify: `src/frontend/src/types/index.ts`
- Modify: `src/frontend/src/views/PromptDetailView.vue`
- Modify: `TODO.md`

- [ ] **Step 1: Add display types**

In `types/index.ts`:

```ts
export interface PromptExample {
  title: string
  input: string
  output: string
}

export interface PromptWorkflowStep {
  title: string
  detail: string
}
```

- [ ] **Step 2: Implement parser**

Create `promptStructure.ts`:

```ts
// 文件作用：从提示词正文中提取只读展示用的 Few-shot 示例和流程步骤。
import type { PromptExample, PromptWorkflowStep } from '@/types'

export function extractPromptExamples(content: string): PromptExample[] {
  const blocks = splitSection(content, /(few[-\s]?shot|示例|案例)/i)
  return blocks.slice(0, 3).map((block, index) => ({
    title: `示例 ${index + 1}`,
    input: extractLabeledText(block, ['input', '输入', '用户']),
    output: extractLabeledText(block, ['output', '输出', '结果'])
  })).filter((item) => item.input || item.output)
}

export function extractPromptWorkflow(content: string): PromptWorkflowStep[] {
  const blocks = splitSection(content, /(workflow|流程|步骤|sop)/i)
  return blocks
    .flatMap((block) => block.split(/\n+/))
    .map((line) => line.replace(/^[-*\d.\s]+/, '').trim())
    .filter(Boolean)
    .slice(0, 6)
    .map((line, index) => ({ title: `步骤 ${index + 1}`, detail: line }))
}
```

Implement helper functions in the same file with explicit string parsing and no `any`.

- [ ] **Step 3: Wire detail page computed values**

In `PromptDetailView.vue`:

```ts
const promptExamples = computed(() => prompt.value ? extractPromptExamples(prompt.value.content) : [])
const promptWorkflow = computed(() => prompt.value ? extractPromptWorkflow(prompt.value.content) : [])
```

- [ ] **Step 4: Add sections below prompt/system prompt cards**

Add two `detail-content-card` sections. Empty state text:

```vue
<p class="detail-structure-empty">当前提示词未提供结构化示例。</p>
<p class="detail-structure-empty">当前提示词未提供结构化流程说明。</p>
```

- [ ] **Step 5: Update checklist**

Mark `Few-shot 展示` and `Workflow 展示` as `[x]`.

- [ ] **Step 6: Verify frontend build**

Run:

```powershell
cd src/frontend
npm run build
```

Expected: PASS.

---

### Task 8: Add Home Skill Category Placeholder

**Files:**
- Modify: `src/frontend/src/views/HomeView.vue`
- Modify: `TODO.md`

- [ ] **Step 1: Add placeholder data**

In `HomeView.vue` script:

```ts
const skillCategoryPlaceholders = [
  { name: '内容创作', desc: 'Prompt 编排与发布流程', icon: 'Content' },
  { name: '电商运营', desc: '商品文案、客服与活动 SOP', icon: 'Ops' },
  { name: '数据分析', desc: '分析任务拆解与报告模板', icon: 'Data' },
  { name: '研发自动化', desc: '代码审查、测试与发布清单', icon: 'Dev' }
] as const
```

- [ ] **Step 2: Render placeholder section**

Add section near category/tag area:

```vue
<section class="skill-entry">
  <div class="skill-entry__head">
    <div>
      <div class="gallery-header__label">Skill</div>
      <h2 class="skill-entry__title">技能分类即将开放</h2>
    </div>
    <span class="skill-entry__badge">Phase 2</span>
  </div>
  <div class="skill-entry__grid">
    <article v-for="item in skillCategoryPlaceholders" :key="item.name" class="skill-entry__card">
      <div class="skill-entry__icon">{{ item.icon }}</div>
      <div class="skill-entry__name">{{ item.name }}</div>
      <p class="skill-entry__desc">{{ item.desc }}</p>
    </article>
  </div>
</section>
```

- [ ] **Step 3: Add responsive Tailwind styles**

Use scoped classes with `@apply`, matching existing card radius and neutral color style.

- [ ] **Step 4: Update checklist**

Mark `Skill 分类入口占位` as `[x]`.

---

### Task 9: Update TODO Decision-Gated Items

**Files:**
- Modify: `TODO.md`

- [ ] **Step 1: Mark AI moderation as blocked**

Set:

```markdown
- [!] AI 内容审核（需确认模型服务、密钥管理、审核等级、误杀处理与人工复核流程）
```

- [ ] **Step 2: Mark notification/analytics items as decision-gated**

For message notification, system notification, and creator analytics that need data ownership or delivery rules, mark `[!]` with short reason in parentheses.

- [ ] **Step 3: Keep Phase 2/3 unchecked**

Do not modify Phase 2/3 TODO status unless explicitly documenting that they remain out of scope.

---

### Task 10: Full Verification And Browser QA

**Files:**
- No production edits expected.

- [ ] **Step 1: Backend tests**

Run:

```powershell
cd src/backend
go test ./...
go build ./cmd/api
```

Expected: both pass.

- [ ] **Step 2: Frontend checks**

Run:

```powershell
cd src/frontend
npm run build
npx eslint . --ext .vue,.js,.jsx,.cjs,.mjs,.ts,.tsx,.cts,.mts
```

Expected: build passes; ESLint has no errors. If the old `v-html` warning remains, explain why or fix it.

- [ ] **Step 3: Browser QA**

Open `http://localhost:3000` and verify:

- 首页 renders Skill placeholder and Prompt feed.
- Prompt detail renders gallery, Few-shot, Workflow, report action, comments.
- Publish page markdown preview does not execute `<script>` or `javascript:` links.

- [ ] **Step 4: Final git status review**

Run:

```powershell
git status --short
git diff --stat
```

Expected: only planned files changed, no generated build artifacts, no secret files.

