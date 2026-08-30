# PromptOS MVP 全页面质量验收

## 验收范围

本轮覆盖当前远端 MVP 已实现页面和接口：首页、发现搜索、社区、Prompt 详情、登录、注册、找回密码、GitHub 回调、发布/草稿、个人主页，以及后端健康检查、分类、Prompt、评论、互动、关注、验证码和图片上传。

## 自动化结果

| 项目 | 命令 | 结果 |
|---|---|---|
| 前端构建 | `cd src/frontend && npm run build` | 通过 |
| 前端单测 | `cd src/frontend && npm test` | 3 个文件、5 个测试通过 |
| 前端 lint | `cd src/frontend && npm run lint:check` | 通过，无 warning |
| 后端静态检查 | `cd src/backend && go vet ./...` | 通过 |
| 后端单测 | `cd src/backend && go test ./...` | 通过 |

## 功能验收表

| 功能 | 路径/API | 前置条件 | 验收结果 | 状态 |
|---|---|---|---|---|
| 首页信息流 | `/`、`/api/v1/home/summary`、`/api/v1/prompts` | 服务可用 | 加载精选、最新内容、分类和标签；分类默认两行附近数量并可展开全部；接口失败显示错误态与重试 | 通过 |
| 搜索筛选 | `/search`、`/api/v1/prompts/search` | 服务可用 | 支持关键词、分类、模型、标签、排序和加载更多；失败不展示假数据 | 通过 |
| 社区页 | `/community` | 服务可用 | 工作流、智能体标签内容和最新动态独立展示 | 通过 |
| Prompt 详情 | `/prompt/:id` | 公开 Prompt | 展示图集、模型参数、结构化内容、相关推荐、评论和互动 | 通过 |
| 登录 | `/login`、`POST /api/v1/user/login` | 已注册账号 | 成功保存 JWT 并回到安全的 redirect；失败提示原因 | 通过 |
| 注册 | `/register`、`POST /api/v1/user/captcha`、`POST /api/v1/user/register` | 邮件服务或开发环境 | 邮箱验证码、密码确认和自动登录流程可用 | 通过（外部邮件需配置） |
| 找回密码 | `/forgot-password` | 可接收验证码 | 验证邮箱后重置密码并返回登录 | 通过（外部邮件需配置） |
| 发布与草稿 | `/publish`、`POST/PUT /api/v1/prompts` | 已登录 | 多步骤表单、图片上传、JSON 校验、发布、编辑和保存草稿可用 | 通过 |
| 个人主页 | `/profile`、`/profile/:userId` | 已登录查看自己的工作台 | 资料、头像、已发布、草稿、收藏、点赞、浏览、关注和粉丝列表可用 | 通过 |
| 社区互动 | Prompt 详情相关 API | 已登录 | 点赞、收藏、评论、回复、评论点赞、举报、关注均有成功/失败反馈 | 通过 |
| 主题切换 | 全局导航 | 任意页面 | Sun/Moon icon button 有 title、aria-label、aria-pressed，亮暗 token 正确 | 通过 |
| 图片上传 | `POST /api/v1/uploads/images` | 已登录、合法图片 | 上传成功返回 URL；非法类型和失败请求有提示 | 通过 |

## 视觉验收

- 亮色主背景为 `#F6F9FF`，卡片为白色，控件为浅蓝色，不再使用米白主背景。
- 暗色主题保持高对比文本、边框和交互状态。
- 页面通过 Tailwind 响应式布局，已在 `1440x900` 与 `390x844` 验证首页、搜索、社区和详情无横向溢出。
- Playwright 运行态检查：公共页面无浏览器控制台 error；未登录访问 `/publish` 会跳转 `/login?redirect=/publish`。

| 场景 | 截图 |
|---|---|
| 首页亮色桌面 | [home-light-desktop.png](screenshots/home-light-desktop.png) |
| 首页暗色桌面 | [home-dark-desktop.png](screenshots/home-dark-desktop.png) |
| 首页暗色移动端 | [home-dark-mobile.png](screenshots/home-dark-mobile.png) |
| 搜索页暗色桌面 | [search-dark-desktop.png](screenshots/search-dark-desktop.png) |
| 社区页暗色桌面 | [community-dark-desktop.png](screenshots/community-dark-desktop.png) |
| Prompt 详情暗色桌面 | [detail-dark-desktop.png](screenshots/detail-dark-desktop.png) |
| 登录页暗色桌面 | [login-dark-desktop.png](screenshots/login-dark-desktop.png) |

## 已知限制

- Docker Engine 不可用时无法完成容器级验收；本轮接口使用本地后端内存 store 验证。
- 生产邮件/短信、GitHub OAuth、R2/COS、HTTPS、备份、告警仍需外部配置。
- Vite 构建提示主 JS chunk 约 1.5 MB，功能不受影响，但上线前应做代码分包和 Naive UI 按需加载。
- `npm audit` 当前存在依赖安全告警，上线前需审查并升级可安全升级的依赖。
