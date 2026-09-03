# Changelog — PromptOS

所有对外可见的变更记录在本文件。版本号：`v主.次.修订`（主=大改/不兼容，次=新功能，修订=修复）。

## [Unreleased]

### Fixed
- 修复 MySQL 大小写不敏感排序规则下标签迁移未执行大小写更新的问题，改用二进制比较并补充 CI 失败回归
- 统一 API 错误模型：存储层改用 sentinel error，错误响应始终带稳定 `errorCode`；未知内部错误返回 `500 INTERNAL_ERROR`，不泄露 SQL 或内部错误文本
- 统一数据库初始化入口：backend 对真正空库自动应用 `schema.sql` 基线后运行迁移，开发 Compose 不再隐式挂载 schema；迁移矩阵新增真实空库场景并通过幂等验证
- 开发 Compose 为 MySQL 显式创建与 backend 配套的 `promptos_app` 账号，避免新卷因账号缺失误降级到内存存储
- 补全迁移目录文档中的 `0009`-`0014` 当前文件清单，避免运维按过期列表漏跑迁移
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
- 新增 `docs/api-contract.schema.json` 共享 DTO Schema 和生成器，前端核心 User/Prompt/Comment/分页类型由生成文件提供，CI 增加生成结果漂移检查
- 首页标题飘带使用已有提示词标题和封面图，支持多行、不同速度和 3D 视觉效果
- 增加首页、提示词卡片与详情页冒烟测试
- 增加低内存友好的一次性多态目标完整性审计命令，输出 JSON 并以退出码驱动定时任务告警
- 增加个人数据导出、浏览历史清除和账号注销 API；个人中心提供对应操作入口
- 增加低内存友好的 Prometheus `/metrics` 端点、Prompt 计数审计和维护命令
- 生产 Compose 增加最小 capability、只读根文件系统、tmpfs 及 PID/内存上限；前端 nginx 增加 CSP Report-Only
- 安全 CI 增加 Gitleaks 全历史机密扫描和 Trivy backend/frontend 运行镜像扫描；Go 加密依赖升级至已修复版本，前端 nginx 运行基线同步安全补丁
- 统一前端生产能力开关；未启用的 OAuth、邮件验证、Skill、Playground、创作者学院和交易市场入口不可误点
- 浏览器认证迁移到 HttpOnly 会话 Cookie，写请求增加 CSRF 双提交校验；旧版 Bearer 客户端保持兼容
- 前端主题初始化移出内联脚本，生产 nginx CSP 切换为强制策略；API 增加 CORS/CSRF 预检和拒绝回归测试
- 生产 Redis Compose 强制独立密码、protected mode 和认证 healthcheck；缺少 `REDIS_PASSWORD` 时配置校验 fail-closed
- 修复首页标题飘带在响应式更新时重新随机排序导致的动画抽搐；搜索/列表继续保持请求取消、分页去重和稳定封面尺寸
- 完善主要页面响应式和无障碍边界：补齐头像文件输入标签，验证 390/768/1440 视口、键盘焦点与 Escape 菜单关闭
- 在迭代清单中单列当前功能缺失、架构/数据/安全风险和服务器前置条件，明确 Skill/Playground、SMTP、管理员审核、RustFS、定时备份、线上旧 release 等未完成边界
- 发布脚本增加镜像归档 SHA-256 强校验、加载后清理和健康后 `current` 指针切换，降低无 Swap/小根盘服务器的发布残留风险

### Changed
- 上传回收命令改为可测试的安全清理核心：仅回收超过窗口的 `pending` 对象，provider 不匹配或删除失败保留待重试，物理删除成功后才转为 `trashed`
- Prompt 图片引用改为显式生命周期：写入成功后才标记 `referenced`，更新或删除后不再使用的旧对象回到 `pending`，由延迟回收任务安全清理
- 新增 `internal/service` 业务层，集中认证/会话、Prompt 互动与缓存失效、举报和上传引用校验；API handler 不再直接编排这些跨 Store 业务操作
- 上传增加可配置的并发槽、单用户 UTC 日字节配额和总容量配额；Redis 使用原子字节计数，数据库已用量与进程内预留量共同防止并发超卖
- 完善事务一致性：举报目标校验与幂等写入、关注关系与派生计数、上传引用批量标记均具备原子边界；跨 Prompt/上传 Store 失败时执行可重试补偿
- 搜索页的数据请求、取消、竞态保护、分页去重和错误状态收敛到 Prompt Store，视图只负责路由筛选与展示
- 列表、搜索、详情和评论请求支持 AbortController 与请求序号，取消旧请求并丢弃过期响应，避免快速导航或筛选时状态回退
- Prompt 标签在内存、MySQL 种子和写入路径统一规范化；新增历史数据迁移清理大小写/空白冲突，迁移可安全重复执行
- 公开 Prompt 查询统一过滤草稿、删除内容和禁用作者：首页汇总、搜索、详情、历史与互动列表不再泄露不可见内容
- MySQL 评论分页按 `latest`/`oldest`/`popular` 在数据库排序后再分页，并增加根评论时间与热度复合索引
- 内存与 MySQL Prompt Store 新增共享契约回归测试，统一创建、分页、互动幂等、浏览历史和删除/越权更新语义
- 新增与当前 MySQL schema 对齐的数据字典，记录表字段、状态、索引、外键、多态目标和保留策略
- 多态互动写入路径校验目标存在性；新增 comments、likes、favorites、reports 孤儿与未知类型扫描及真实 MySQL 回归测试
- 开发启动入口改为前台显示前后端实时日志，关闭窗口或按 Ctrl+C 时自动清理子进程和容器
- 启动脚本支持复用已有 PromptOS MySQL/Redis 容器，并忽略本地 PID 文件
- 工作台浏览历史改为分页加载，相关推荐改为后端实时查询
- MySQL 连接池参数化，并为本地和生产 Compose 增加日志轮转与禁止提权配置
- 注销事务清理个人互动/历史明细并修正反规范化计数；禁用账号的 Prompt/评论保留用于审计但不再公开
- 登录、注册、验证码、重置、评论、举报、搜索、互动和上传改为 IP/邮箱/用户分维度限流；生产 Redis 不可用时安全能力 fail-closed
- Prompt、评论、用户名、简介、标签和图片 URL 统一执行长度、控制字符与脚本内容校验，内存/MySQL 入口共享规则并补充回归测试
- 同步服务器总说明与 PromptOS 迭代清单：记录 2026-09-04 线上资源、Compose/卷/端口事实，修正 RustFS 当前 bridge 地址，并明确 23 项待办的生产验收门槛
- 在服务器总说明增加完整 81 项迭代矩阵，按 P0、架构、数据、安全、前端、运维和发布逐项标注状态、执行层与服务器门槛；同步完成基线为 58 项

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
