# Changelog — PromptOS

所有对外可见的变更记录在本文件。版本号：`v主.次.修订`（主=大改/不兼容，次=新功能，修订=修复）。

## [Unreleased]

### Security（计划中）
- 生产 compose 的数据库凭据、JWT 和 OAuth 开关改为环境变量注入，应用与迁移数据库账号分离
- 生产启动在 MySQL/迁移失败时 fail-closed，不再降级到内存存储
- 验证码改为 HMAC 摘要、Redis 原子消费，生产响应和日志不再暴露验证码
- 公开用户响应移除邮箱、账号状态和 OAuth 绑定字段

### Added
- 首页标题飘带使用已有提示词标题和封面图，支持多行、不同速度和 3D 视觉效果
- 增加首页、提示词卡片与详情页冒烟测试

### Changed
- 开发启动入口改为前台显示前后端实时日志，关闭窗口或按 Ctrl+C 时自动清理子进程和容器
- 启动脚本支持复用已有 PromptOS MySQL/Redis 容器，并忽略本地 PID 文件

### Fixed
- 修复提示词卡片默认链接为空字符串导致无法进入详情页的问题
- 修复提示词详情页在作者、评论用户数据缺失时渲染异常的问题
- 修复评论/历史分页响应与前端 DTO 不一致、举报枚举错误和互动无法取消的问题

## [v0.2.0] - 2026-08-30

### Deployment
- 发布至 `promptsystem.isoumao.top`，Compose 项目 `promptsystem`。
- 当前 release：`/srv/releases/promptsystem/20260830-b584585`；回滚 release：`20260829-a9ba2cf`。
- 服务器备份：`/srv/backups/promptsystem/20260830-b584585/`，SHA-256 已记录在总 TODO。

## [20260829-a9ba2cf] - 2026-08-29

### Deployment
- 首次部署至 `promptsystem.isoumao.top`。
- 使用独立 MySQL、Redis、上传卷和 `promptsystem` Compose 项目；生产入口接入 HTTPS nginx 反代。

## [v0.0.1] - YYYY-MM-DD

- MVP 初始版本（package.json 0.0.1；历史任务编号见 `docs\后端迭代任务清单.md`）
