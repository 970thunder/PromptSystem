# PromptOS API 契约

> 版本：v1（本轮迭代）
> 基础路径：`/api/v1`
> 统一响应信封：`{ "code": number, "message": string, "errorCode": string?, "data": any }`
> HTTP status 表达协议结果；`errorCode` 为稳定的机器可读错误标识，前端展示 `message`，逻辑判断使用 `errorCode`。

## 通用约定

- 认证：受保护接口使用 `Authorization: Bearer <token>`。
- 分页：`page`（默认 1，`>=1`）、`pageSize`（默认 12，`1..100`）。非法值返回 `400`。
- 请求体：单个 JSON 值，未知字段被拒绝，超限返回 `413 BODY_TOO_LARGE`。
- 错误码示例：`AUTH_INVALID_CREDENTIALS`、`AUTH_TOKEN_EXPIRED`、`PROMPT_NOT_FOUND`、`INVALID_PAGE`、`INVALID_PAGE_SIZE`、`INVALID_JSON`、`ORIGIN_NOT_ALLOWED`、`INTERNAL_ERROR`。

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

响应：
```json
{ "code": 200, "message": "Success", "data": { "list": [ ... ], "total": 0, "page": 1, "pageSize": 12 } }
```

### `GET /prompts/search`
搜索接口，参数与 `/prompts` 一致。

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

### `POST /prompts/{id}/like | /favorite | /view | /report`（需登录）
互动操作。`like`/`favorite` 幂等；`view` 记录浏览；`report` 提交举报。

### `GET /prompts/{id}/comments`
评论列表。支持 `sort`（`hot`/`newest`/`oldest`）。

### `POST /prompts/{id}/comments`（需登录）
创建评论。请求体：`{ "content": "...", "parentId": null }`。最多两层回复。

### `GET /home/summary`
首页聚合数据：已发布 Prompt 数、创作者数、热门标签/分类。实时计算（短期可缓存）。

## 用户与认证

### `POST /user/register`
注册。请求体：`{ "username", "email", "password", "captcha" }`。

### `POST /user/captcha`
发送验证码。请求体：`{ "email" }`。非生产返回 `devCode`；生产不返回。

### `POST /user/login`
登录。请求体：`{ "email", "password" }`。失败统一 `401 AUTH_INVALID_CREDENTIALS`（不存在或密码错误不区分）。

### `POST /user/password/reset`
重置密码。请求体：`{ "email", "captcha", "password" }`。

### `POST /user/logout`（需登录）
登出：将当前 JWT `jti` 写入 Redis denylist，TTL 等于 token 剩余有效期。

### `GET /user/info` / `PUT /user/info`（需登录）
当前用户资料读取与更新。

### `GET /user/favorites`、`/user/likes`、`/user/history`、`/user/drafts`（需登录）
当前用户收藏/点赞/浏览历史/草稿列表。

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

## 评论

### `POST /comments/{id}/like`、`POST /comments/{id}/report`（需登录）
评论点赞/举报。

## 上传

### `POST /uploads/images`（需登录，multipart）
上传图片。整个请求受大小限制，校验真实图片（可解码、宽高/像素阈值、MIME 一致）。

## 静态文件

### `GET /uploads/{path}`
本地上传文件。设置 `X-Content-Type-Options: nosniff`、明确 Content-Type、缓存策略；禁止目录列表与路径穿越。
