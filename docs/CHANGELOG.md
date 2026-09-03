# Changelog — PromptOS

所有对外可见的变更记录在本文件。版本号：`v主.次.修订`（主=大改/不兼容，次=新功能，修订=修复）。

## [Unreleased]

### Fixed
- 统一数据库初始化入口：backend 对真正空库自动应用 `schema.sql` 基线后运行迁移，开发 Compose 不再隐式挂载 schema；迁移矩阵新增真实空库场景并通过幂等验证
- 开发 Compose 为 MySQL 显式创建与 backend 配套的 `promptos_app` 账号，避免新卷因账号缺失误降级到内存存储
- 亮色主题统一为蓝白配色，主题切换改为可访问的太阳/月亮图标按钮
- 首页、搜索和个人主页不再把演示数据作为运行时 API 失败回退
- 补齐认证、上传、社交互动和详情加载的错误反馈与重试路径
- 恢复前端测试依赖，修复 ESLint 警告并统一 Docker 密钥默认值为开发占位符

### Security（计划中）
- 生产 compose 的数据库凭据、JWT 和 OAuth 开关改为环境变量注入，应用与迁移数据库账号分离
- 生产启动在 MySQL/迁移失败时 fail-closed，不再降级到内存存储
- 验证码改为 HMAC 摘要、Redis 原子消费，生产响应和日志不再暴露验证码
- 公开用户响应移除邮箱、账号状态和 OAuth 绑定字段
- 生产播种回归测试确保空库不创建演示账号或 Prompt；线上演示账号已在备份后禁用并失效旧会话

### Added
- 首页标题飘带使用已有提示词标题和封面图，支持多行、不同速度和 3D 视觉效果
- 增加首页、提示词卡片与详情页冒烟测试

### Changed
- 开发启动入口改为前台显示前后端实时日志，关闭窗口或按 Ctrl+C 时自动清理子进程和容器
- 启动脚本支持复用已有 PromptOS MySQL/Redis 容器，并忽略本地 PID 文件
- 工作台浏览历史改为分页加载，相关推荐改为后端实时查询
- MySQL 连接池参数化，并为本地和生产 Compose 增加日志轮转与禁止提权配置

### Fixed
- 修复提示词卡片默认链接为空字符串导致无法进入详情页的问题
- 修复提示词详情页在作者、评论用户数据缺失时渲染异常的问题
- 修复评论/历史分页响应与前端 DTO 不一致、举报枚举错误和互动无法取消的问题
- 修复评论加载失败时无法重试、相关推荐依赖当前页面缓存的问题
- CI 固定 Node 24、使用 npm ci，补齐 MySQL schema 集成初始化、迁移矩阵路径，并升级 AWS SDK 安全补丁
- GitHub Actions 前端、后端、契约、迁移矩阵、Docker fresh-start 和安全扫描已在 `ffcfaf7` 上全部通过
- 本批代码提交 `27eb81e` 尚未部署到生产；生产仍运行 `20260830-b584585`，发布需走备份、迁移、健康检查和回滚流程
- 线上演示账号处置记录：备份 `/srv/backups/promptsystem/20260830-demo-removal/`，固定密码登录已返回 401，6 条 Prompt 归属 `PromptOS Official`；本次仅更新数据与文档，尚未发布新的应用镜像
- 完成首次恢复演练：MySQL 与 uploads 备份在临时资源恢复，上一版本 ready 通过（RTO 约 50 秒）；临时容器、卷、网络和目录已清理
- 发布检查清单已实化为当前生产参数：域名、SSH、Compose 项目、loopback 端口、备份、迁移和回滚命令均已固定记录
- 收紧前端站内 redirect 校验，拒绝编码协议相对路径、反斜杠和控制字符
- 生产 API 请求失败时不再静默切换演示数据，改为真实错误和可重试空状态
- 增加登出 JWT 撤销回归测试，旧 token 写入 denylist 后立即失效
- 最新提交 `2f9201a` 的 backend-ci `33275544929`、security-scan `33275544928` 均成功

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
