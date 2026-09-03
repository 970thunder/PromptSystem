# PromptOS API 契约

> 版本：v1（本轮迭代）
> 基础路径：`/api/v1`
> 统一响应信封：`{ "code": number, "message": string, "errorCode": string?, "data": any }`
> HTTP status 表达协议结果；`errorCode` 为稳定的机器可读错误标识，前端展示 `message`，逻辑判断使用 `errorCode`。

## 通用约定

- 认证：浏览器生产会话使用 `HttpOnly; Secure; SameSite=Lax` 的 `promptos_session` Cookie，前端请求携带凭据并使用可读的 `promptos_csrf` Cookie 值设置 `X-CSRF-Token` 保护 `POST`/`PUT`/`PATCH`/`DELETE` 写操作。`GET`/`HEAD`/`OPTIONS`/`TRACE` 不要求 CSRF Header。旧版 API 客户端仍可使用 `Authorization: Bearer <token>`，Bearer 请求不读取会话 Cookie，也不要求 CSRF Header。
- 跨域：生产 `ALLOWED_ORIGIN` 必须是正式 HTTPS 源的逗号分隔白名单，禁止 `*`；白名单源收到 `Access-Control-Allow-Credentials: true`，未列出的 Origin 返回 `403 ORIGIN_NOT_ALLOWED`。同源请求不需要 Origin Header，`OPTIONS` 预检成功返回 `204`。生产 Redis 必须设置独立 `REDIS_PASSWORD`，Redis 只接受 Compose 内网连接。
- 分页：`page`（默认 1，`>=1`）、`pageSize`（默认 12，`1..100`）。非法值返回 `400`。
- 请求体：单个 JSON 值，未知字段被拒绝，超限返回 `413 BODY_TOO_LARGE`。
- 错误码示例：`AUTH_INVALID_CREDENTIALS`、`AUTH_TOKEN_MISSING`、`AUTH_TOKEN_INVALID`、`AUTH_TOKEN_EXPIRED`、`AUTH_TOKEN_REVOKED`、`AUTH_USER_DISABLED`、`USER_NOT_FOUND`、`CANNOT_FOLLOW_SELF`、`PROMPT_NOT_FOUND`、`PROMPT_FORBIDDEN`、`COMMENT_NOT_FOUND`、`COMMENT_PARENT_NOT_FOUND`、`COMMENT_PARENT_MISMATCH`、`INVALID_COMMENT_CONTENT`、`INVALID_REPORT_REASON`、`REPORT_DETAIL_TOO_LONG`、`INVALID_CATEGORY`、`INVALID_TAG`、`INVALID_PAGE`、`INVALID_PAGE_SIZE`、`INVALID_JSON`、`ORIGIN_NOT_ALLOWED`、`INVALID_UPLOAD_OWNERSHIP`、`INVALID_IMAGE_FORMAT`、`IMAGE_REQUIRED`、`IMAGE_TOO_LARGE`、`REQUEST_TOO_LARGE`、`UPLOAD_CONCURRENCY_LIMITED`、`UPLOAD_DAILY_QUOTA_EXCEEDED`、`UPLOAD_CAPACITY_EXCEEDED`、`UPLOAD_QUOTA_UNAVAILABLE`、`UPLOAD_REFERENCE_FAILED`、`UPLOAD_LIFECYCLE_FAILED`、`HISTORY_CLEAR_FAILED`、`DATA_EXPORT_FAILED`、`INTERNAL_ERROR`。
- 内部错误不直接暴露给客户端：`err.Error()` / SQL 细节不会出现在响应里，业务错误统一走稳定 `errorCode` + 展示用 `message`。

## 健康检查

### `GET /health`
历史兼容接口，报告进程与存储模式。

### `GET /health/live`
存活探针：进程可服务即返回 `200`。

### `GET /health/ready`
就绪探针：检查 MySQL 存储模式。开发环境降级返回 `200` 且 `degraded: true`；非开发环境降级返回 `503`。

响应示例：
```json
{ "code": 200, "message": "Success", "data": { "status": "ready", "storageMode": "mysql", "degraded": false } }
```

## 分类

### `GET /categories`
返回分类列表。支持 `type` 查询（`prompt`/`skill`）。

## Prompt

### `GET /prompts`
Prompt 列表。查询参数：`page`、`pageSize`、`categoryId`、`userId`、`sort`、`keyword`、`model`、`tag`。

`sort` 使用 `latest`（默认）或 `popular`；`tag` 用于能力/主题标签筛选。前端历史链接中的 `tab=workflow`、`tab=agent` 仅作兼容输入，分别规范化为 `tag=流程`、`tag=智能体`，服务端无需处理 `tab`。

响应：
```json
{ "code": 200, "message": "Success", "data": { "list": [ ... ], "total": 0, "page": 1, "pageSize": 12 } }
```

### `GET /prompts/search`
搜索接口，参数与 `/prompts` 一致（包括 `page`、`pageSize`、`keyword`、`categoryId`、`model`、`tag` 和 `sort=latest|popular`）。

### `POST /prompts`（需登录）
发布 Prompt。请求体：
```json
{ "title": "...", "description": "...", "cover": "...", "images": [], "content": "...", "systemPrompt": "", "model": "Midjourney v6", "params": { "temperature": 0.7 }, "categoryId": 1, "tags": ["品牌"], "status": 1 }
```
错误：`400`（非法分类/标签/图片 URL）、`401`。

### `GET /prompts/{id}`
Prompt 详情。不存在返回 `404 PROMPT_NOT_FOUND`。

### `PUT /prompts/{id}`（需登录，本人）
更新 Prompt。非本人返回 `403`。

### `DELETE /prompts/{id}`（需登录，本人）
软删除 Prompt。返回 `200`。

### `POST /prompts/{id}/like | /favorite | /report`（需登录）
互动操作。`like`/`favorite` 均**幂等**：同一用户对同一 Prompt 重复操作只生效一次，计数只加 1。响应返回 `{ prompt, applied }`，`applied=true` 表示本次是首次生效。

### `DELETE /prompts/{id}/like | /favorite`（需登录）
取消点赞/取消收藏。与 `POST` 对应，**幂等**并反转计数（同一事务内删除明细并递减 `prompts.likes/favorites`）。响应返回 `{ prompt, applied }`。

### `GET /prompts/{id}/interaction`（需登录）
返回当前用户对该 Prompt 的互动状态：`{ "liked": boolean, "favorited": boolean }`。前端据此渲染点赞/收藏按钮的选中态，不依赖反规范化计数猜测。Prompt 不存在或已软删除返回 `404 PROMPT_NOT_FOUND`。

`report` 提交举报，`reason` 必须是受限枚举 `spam` / `abuse` / `nsfw` / `other` 之一，`detail` 最多 500 字（runes）。举报**幂等**：同一用户对同一目标只保留一条自动举报，重复提交不重复计数，响应返回 `{ report, applied }`。对已软删除的 Prompt 举报返回 `404 PROMPT_NOT_FOUND`；非法 `reason` 返回 `400 INVALID_REPORT_REASON`。

### `POST /prompts/{id}/view`（可选登录）
浏览计数接口，**匿名友好**（无需 Bearer 也可调用）。见下方「收藏/浏览/互动」隐私约定。

### `GET /prompts/{id}/comments`
评论列表。支持 `sort`（`latest`/`popular`/`oldest`），并返回统一分页信封 `{ list, total, page, pageSize }`。

### `POST /prompts/{id}/comments`（需登录）
创建评论。请求体：`{ "content": "...", "parentId": null }`。最多两层回复。

### `GET /home/summary`
首页聚合数据：已发布 Prompt 数、创作者数、热门标签/分类。实时计算（短期可缓存）。

## 收藏/浏览/互动

### 计数与明细一致性（B5-02）
`likes`/`favorites`/`view_histories` 明细与 `prompts.likes/favorites/views` 反规范化计数**在同一事务内写入**：要么明细与计数同增、要么都回滚，杜绝「有明细无计数」或「有计数无明细」的半写。互动均为**幂等**：

- `like`：`likes` 表 `(user_id, target_type, target_id)` 唯一约束保证重复动作只增一次计数。
- `favorite`：同上，`favorites` 表唯一约束。
- `view`（登录用户）：`view_histories` 的 `(user_id, prompt_id)` 唯一约束保证同一用户对同一 Prompt 只产生一条历史、计数只增一次。

其他写入边界：Prompt 与标签在同一事务提交；Prompt/评论举报在同一事务内校验公开目标、执行幂等插入并读取结果；关注/取关按稳定用户锁顺序在同一事务内更新关系并计算派生关注数。Prompt 写入与上传元数据属于不同 Store，若引用标记失败，服务层会先重试标记当前 Prompt 引用，再将旧引用退回 `pending`；补偿失败保持可重试状态，不直接删除对象。

历史不一致可用迁移 `sql/migrations/0012_recalibrate_counters.sql` 一键重算（可重复执行、幂等）。

### 浏览隐私与去重（B5-03）
- **匿名浏览**：只增加 `prompts.views` 总浏览计数，**不写浏览历史**（无法归属到用户）。`POST /prompts/{id}/view` 无需登录即可调用。
- **登录用户浏览**：写入/刷新 `view_histories`，**同一用户对同一 Prompt 只保留一行**，重复浏览只刷新 `viewed_at`（历史更新时间），不新建行、不重复计数；首次浏览才使 `views` 计数 +1。
- 用户只能读取**自己的**历史（服务端以鉴权上下文为准）；软删除的 Prompt 永不出现在历史响应。
- `views` 计数语义：匿名每次 +1，登录用户首次 +1（去重后）。因此 `views` 是「匿名独立浏览 + 登录去重后首次」的近似口径，重算迁移以 `view_histories` 为可重建基线（详见迁移头注释）。

### 举报与审核状态闭环（B5-04）
- `reason` 为受限枚举：`spam`、`abuse`、`nsfw`、`other`（代码内以类型化常量定义），非法值返回 `400 INVALID_REPORT_REASON`。
- `detail` 上限 500 字（runes），超限拒绝。
- 举报**幂等**：`reports` 表 `(user_id, target_type, target_id)` 唯一约束保证同一用户对同一目标只保留一条、重复提交不重复计数。
- 对已软删除目标（Prompt/评论）举报返回 `404 PROMPT_NOT_FOUND` / `404 COMMENT_NOT_FOUND`。
- 本轮不实现审核后台（Phase 3）；`reports.status` 保留 `pending/reviewed/rejected` 供未来审核流程使用。未来审核队列建议补充索引 `(target_type, target_id, status)` 与按 `status, created_at` 的取队列索引。

## 用户与认证

### `POST /user/register`
注册。请求体：`{ "username", "email", "password", "captcha" }`。

### `POST /user/captcha`
发送验证码。请求体：`{ "email" }`。非生产返回 `devCode`；生产不返回，必须配置 SMTP。SMTP 不可用时返回 `502`（`EMAIL_SEND_FAILED`），发送失败不会保留验证码。

### `POST /user/login`
登录。请求体：`{ "email", "password" }`。失败统一 `401 AUTH_INVALID_CREDENTIALS`（不存在或密码错误不区分）。

### `POST /user/password/reset`
重置密码。请求体：`{ "email", "captcha", "password" }`。

### `POST /user/logout`（需登录）
登出：将当前 JWT `jti` 写入 Redis denylist，TTL 等于 token 剩余有效期，并清除会话和 CSRF Cookie。

### `GET /user/info` / `PUT /user/info`（需登录）
当前用户资料读取与更新。

### `GET /user/favorites`、`/user/likes`、`/user/drafts`（需登录）
当前用户收藏/点赞/草稿列表。

### `GET /user/history`（需登录）
当前用户浏览历史，**分页返回**（`page`/`pageSize`，同全局约定），响应为 `{ list, total, page, pageSize }`。只返回**本人**历史（用户 ID 取自鉴权上下文，不接受第三方 userId 参数），且**软删除（已下架）的 Prompt 不会出现**在结果中。新增/刷新浏览见下方隐私约定。

### `DELETE /user/history`（需登录）
清空当前用户的浏览历史。只删除 `view_histories` 中本人的记录，不回退 Prompt 的累计浏览计数。成功返回 `{ "cleared": true }`。

### `GET /user/data-export`（需登录）
导出当前用户可访问的账户数据，返回 `exportedAt`、安全用户 DTO、本人保留的 Prompt（含草稿）、收藏、点赞和浏览历史。响应不包含密码哈希、GitHub 绑定 ID 或邮箱以外的认证材料；导出频率受用户维度限流。

### `DELETE /user/account`（需登录）
注销当前账户。账户转为 disabled，密码、GitHub 绑定和直接标识被匿名化，`session_version` 递增使所有旧 JWT 立即失效；个人点赞、收藏、举报、关注和浏览历史清理。本人 Prompt 和评论记录按保留策略留存，但因作者已禁用而不再公开。成功返回 `{ "deleted": true }`；该操作可安全重试。

### `GET|POST|DELETE /users/{id}/follow`、`GET /users/{id}/follow-status`（需登录）
关注/取关/关注状态。

### `GET /users/{id}/following`、`GET /users/{id}/followers`
关注列表。

## GitHub OAuth

### `GET /auth/github`
发起 OAuth。未配置时返回 `503`。

### `GET /auth/github/callback`
回调。校验 state/cookie/TTL，换取一次性 code 并跳转前端 `/auth/callback?code=...`。**URL 不携带 JWT**。

### `POST /auth/exchange`
用一次性 code 换取 JWT。code 60 秒过期、只允许使用一次。

- 请求：`{ "code": "..." }`
- 成功：`200`，返回 `{ code, message, data: { token, user } }`（与登录/注册同构）
- 失败：`400 INVALID_EXCHANGE_CODE`（缺失/无效/已用过/过期）；code 对应的用户已禁用或会话版本不匹配时 `401 AUTH_TOKEN_REVOKED`

## 评论

### `POST /comments/{id}/like`、`POST /comments/{id}/report`（需登录）
评论点赞/举报。`like` 幂等（同一用户对同一评论只增一次计数）。`report` 的 `reason` 同为受限枚举（`spam`/`abuse`/`nsfw`/`other`）、`detail` ≤ 500 字，且幂等（唯一约束去重）；对不存在的评论返回 `404 COMMENT_NOT_FOUND`。

## 上传

### `POST /uploads/images`（需登录，multipart）
上传图片。整个请求受大小限制，校验真实图片（可解码、宽高/像素阈值、MIME 一致），并受并发槽、单用户 UTC 日配额和总容量配额限制。配额由 `UPLOAD_MAX_CONCURRENT`、`UPLOAD_DAILY_QUOTA_MB`、`UPLOAD_TOTAL_QUOTA_MB` 配置；Redis 可用时日配额按用户原子累计字节，生产 Redis 不可用返回 `503 UPLOAD_QUOTA_UNAVAILABLE`。

## 静态文件

### `GET /uploads/{path}`
本地上传文件。设置 `X-Content-Type-Options: nosniff`、明确 Content-Type、缓存策略；禁止目录列表与路径穿越。
