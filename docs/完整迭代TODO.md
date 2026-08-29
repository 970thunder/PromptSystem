# PromptOS 完整迭代 TODO

更新时间：2026-08-30

本文是 PromptOS 后续开发、生产加固和服务器运维的唯一总清单。执行时遵守 `E:\Web\服务器部署总说明.md`、`AGENTS.md`、`docs/API契约.md` 和 `docs/DEPLOYMENT.md`。只有在代码、测试、服务器状态或恢复演练提供可复核证据后才允许将 `[ ]` 改成 `[x]`。

## 当前基线

- 生产域名：`https://promptsystem.isoumao.top`
- Compose 项目：`promptsystem`；当前发布目录：`/srv/releases/promptsystem/20260830-b584585`；上一回滚目录：`/srv/releases/promptsystem/20260829-a9ba2cf`
- 服务器：7.7 GiB 内存、无 Swap、58 GiB 根盘（2026-08-30 剩余约 32 GiB）
- 公网只开放 80/443；PromptOS 前端/后端分别绑定 `127.0.0.1:3092/5092`
- RustFS 转发链路可用，但 PromptOS 尚无独立 bucket/低权限凭据，上传暂存独立 Docker 卷
- 服务器已有多个 Compose 项目，禁止全局 prune、删除未知卷、改动其他项目或在服务器编译源码

## P0 生产阻塞项

- [x] **P0-01 正式生产环境**：生产设置 `APP_ENV=production`；开发、测试、生产判断集中到配置包。验收：生产不会返回 `devCode`，生产校验测试通过。
- [x] **P0-02 禁止生产内存降级**：MySQL、迁移或生产初始化失败时启动失败，不接受会在重启后消失的写入。验收：线上应用账号 DDL 权限拒绝时 backend 反复启动失败且 ready 不可用；恢复迁移账号后 ready 200。
- [x] **P0-03 禁止生产演示账号 Seed**：生产只维护必要参考数据，不创建固定密码用户和演示 Prompt。验收：空生产库启动后无演示账号。
- [x] **P0-04 处置线上演示账号**：先备份，再禁用/重置已有固定密码账号，并明确 6 条 Prompt 的官方内容归属。验收：已知密码无法登录，数据处置有记录。
- [x] **P0-05 公开用户隐私拆分**：公开用户 DTO 不含邮箱、OAuth 绑定和账户状态；本人 DTO 单独返回私有字段。验收：Prompt/评论/关注公开响应无邮箱。
- [ ] **P0-06 验证码生产安全**：接入邮件发送；验证码不在生产响应/日志出现，Redis 只存哈希并原子消费。验收：发送、过期、重放、并发、限流测试通过。
- [x] **P0-07 独立数据库账号**：应用不再以 MySQL root 运行，迁移与运行权限分离。验收：最小权限账号可正常运行，越权 DDL/跨库访问失败。
- [x] **P0-08 评论分页契约统一**：前端使用 `{list,total,page,pageSize}`。验收：前端 Store 分页/去重测试通过，线上空评论返回统一分页信封。
- [x] **P0-09 浏览历史分页契约统一**：工作台适配分页响应。验收：历史列表不再被错误清空，分页/软删除过滤正确。
- [x] **P0-10 举报枚举修复**：前端只发送 `spam/abuse/nsfw/other`；后端保留非法枚举拒绝。验收：代码审计、后端 store 契约测试和前端构建通过。
- [x] **P0-11 互动闭环**：接入 interaction、unlike、unfavorite，显示真实选中态。验收：后端互动集成测试、前端构建和状态接入通过。
- [x] **P0-12 首次可恢复备份**：完成 MySQL 与上传卷备份、校验、临时恢复。验收：记录备份位置、校验值、恢复步骤、RPO/RTO。

## A 架构

- [ ] **A-01 统一数据库初始化**：空库、旧 schema、部分迁移库通过同一入口到达最终版本，不再人工先导入 `schema.sql`。
- [ ] **A-02 Service 层**：认证、互动、举报、上传引用和缓存失效从 handler 抽到窄业务服务。
- [ ] **A-03 Store 语义一致**：MySQL/内存实现通过同一契约测试；内存实现仅限开发和测试。
- [ ] **A-04 前端数据层收敛**：View 不重复直连 API；Store 统一 loading/error/empty/success、缓存和竞态。
- [ ] **A-05 API 单一契约**：引入 OpenAPI/JSON Schema 或等价生成校验，TypeScript DTO 不手工漂移。
- [ ] **A-06 统一错误模型**：所有接口返回稳定 `errorCode`，禁止返回 `err.Error()` 或通过字符串分支。
- [ ] **A-07 请求取消与竞态**：搜索、详情、评论使用 AbortController/请求序列，旧响应不能覆盖新状态。
- [x] **A-08 数据库连接池**：配置最大/空闲连接与生命周期，并采集连接池指标。
- [ ] **A-09 SQL 分页与排序**：评论热门排序在数据库完成；主要列表执行 `EXPLAIN` 并补必要复合索引。
- [ ] **A-10 轻量后台任务**：上传回收、数据审计、备份校验优先使用 systemd timer/cron，不新增高内存常驻套件。
- [ ] **A-11 可观测性**：提供请求、错误、延迟、数据库、Redis、上传和任务指标。
- [ ] **A-12 发布架构自动化**：本机构建、镜像校验、上传、迁移、切换、验证和回滚脚本化；服务器不编译。

## D 数据

- [x] **D-01 上传所有者约束**：`uploads.owner_id` 建立用户引用或等价强校验。
- [ ] **D-02 多态目标完整性**：likes/favorites/reports/comments 写入前校验目标，并定期扫描孤儿记录。
- [ ] **D-03 删除生命周期**：统一 Prompt、评论、用户、举报和上传的禁用/软删/回收规则。
- [ ] **D-04 上传垃圾回收**：草稿删除、替换图片、发布失败和注销产生的未引用对象延迟回收。
- [ ] **D-05 汇总计数审计**：定期核对 views/likes/favorites 与明细表，并对偏差告警。
- [ ] **D-06 事务一致性**：发布、互动、评论、举报、关注的明细与汇总在同一事务或有明确补偿。
- [ ] **D-07 标签规范化**：统一大小写、空白、长度和唯一性；热门标签由后端聚合。
- [ ] **D-08 公开查询边界**：草稿、删除内容、禁用用户内容不能出现在首页和搜索。
- [ ] **D-09 数据字典**：记录全部表、字段、状态值、索引、外键和保留期限。
- [ ] **D-10 个人数据能力**：账号注销、数据导出、浏览历史清除和注销后会话失效。
- [ ] **D-11 MySQL 定时备份**：每日逻辑备份、压缩、校验、保留和失败告警。
- [ ] **D-12 PromptOS 独立 RustFS bucket**：使用独立最小权限凭据，不复用其他站点主凭据。
- [ ] **D-13 上传迁移与双副本**：从 Docker 卷迁移到 RustFS，另保留不同故障域副本。
- [ ] **D-14 恢复演练**：每月恢复数据库和对象，使用上一应用版本通过 ready 与关键流程。

## S 安全

- [x] **S-01 会话撤销**：登出调用后端；密码重置、禁用、注销使旧 token 立即失效。
- [ ] **S-02 JWT 存储策略**：先完成 CSP/XSS 审计，再评估迁移 HttpOnly+Secure+SameSite Cookie。
- [x] **S-03 GitHub OAuth 开关**：显式 enable 配置；未配置时前端不显示可点击入口，启用时校验 redirect/state/code。
- [ ] **S-04 CSP**：从 Report-Only 开始制定 Vue 生产 CSP，再切换强制模式。
- [ ] **S-05 CORS/CSRF**：生产仅允许正式域名；Cookie 会话启用 CSRF 防护。
- [x] **S-06 重定向安全**：站内 redirect 覆盖双斜杠、反斜杠、协议相对和编码变体。
- [ ] **S-07 分维度限流**：登录、注册、验证码、重置、评论、举报、搜索、上传按 IP/账号/用户限制。
- [ ] **S-08 内容安全**：Prompt/评论/用户名/标签/Markdown/URL 统一长度、控制字符和脚本校验。
- [ ] **S-09 上传安全与配额**：MIME、解码、像素、格式、并发、单用户日配额和总容量限制。
- [ ] **S-10 Redis 隔离**：独立网络、ACL/密码，禁止公网访问。
- [ ] **S-11 容器加固**：非 root、`no-new-privileges`、`cap_drop`、可行的只读根文件系统、资源与日志限制。
- [ ] **S-12 依赖与镜像扫描**：npm audit、govulncheck、镜像扫描、secret scanning 纳入 CI。
- [ ] **S-13 密钥轮换**：JWT、MySQL、Redis、RustFS、OAuth、SMTP 均有���换和验证流程。
- [ ] **S-14 审核权限与审计**：举报审核、内容下架、用户禁用使用管理员权限并记录不可抵赖审计。

## F 产品与前端

- [x] **F-01 匿名浏览计数**：未登录阅读详情也调用匿名友好的 view 接口。
- [x] **F-02 公开作者页**：`/profile/:userId` 可匿名访问，只有 `/profile` 本人工作台需要登录。
- [x] **F-03 真正退出登录**：UI 调用 `/user/logout` 后清理本地状态并返回安全页面。
- [x] **F-04 评论工作区**：分页、排序、加载更多、回复、错误重试和真实总数。
- [x] **F-05 相关推荐**：通过稳定后端查询获取，不依赖当前 Pinia 页缓存。
- [ ] **F-06 生产能力开关**：OAuth、邮件、Skill、Playground 等未启用能力不可误点。
- [x] **F-07 清理演示 fallback**：生产 API 失败显示真实错误，不自动切换成演示数据。
- [ ] **F-08 搜索与列表性能**：URL 恢复、取消旧请求、稳定图片尺寸、加载更多与错误恢复。
- [ ] **F-09 响应式与无障碍**：390/768/1440 px、键盘、焦点、Escape、reduced motion 全部验收。
- [ ] **F-10 浏览器 E2E**：覆盖首页、导航、搜索、详情、注册登录、评论互动、发布、工作台、主题与移动端。
- [ ] **F-11 更新前端重设计手册状态**：以实际提交和验收证据补齐原 24 项状态，禁止批量虚假勾选。

## O 服务器和运维

- [ ] **O-01 资源基线与告警**：磁盘 70/80/90%、内存、容器重启、ready、证书、备份失败告警。
- [x] **O-02 无 Swap 条件下的发布预算**：本次镜像 load、备份、迁移和重启均串行完成，未并行高内存任务。
- [x] **O-03 Docker 日志轮转**：单容器限制大小与份数，避免日志无上限增长。
- [ ] **O-04 安全清理流程**：只清理确认的悬空镜像和过期 build cache，禁止全局/卷 prune。
- [ ] **O-05 nginx 管理规范**：记录 `/www/server/nginx` 实际 master、配置路径、测试/reload；修复 systemd 状态与实际进程不一致。
- [ ] **O-06 证书续期验收**：Certbot renew 后调用正确 nginx reload 并做 HTTPS 探测。
- [ ] **O-07 RustFS 链路监控**：监控 13900/13902、forward service、ready 和隧道恢复。
- [x] **O-08 端口边界**：公网仅 80/443；应用端口保持 loopback，线上 `ss` 检查通过。
- [x] **O-09 多项目隔离**：本次仅操作 `promptsystem` 项目，线上 `docker compose ls` 其余项目保持运行。
- [ ] **O-10 故障演练**：backend/MySQL/Redis/RustFS/nginx/证书/磁盘逐项演练并记录恢复。

## R 测试、CI 与发布

- [x] **R-01 前端 CI**：已新增 workflow；需 GitHub Actions 实际成功运行 lint、Vitest 和生产 build 后勾选。
- [x] **R-02 后端 CI**：gofmt、vet、test、race、build、MySQL/Redis 集成、迁移矩阵。
- [x] **R-03 契约 CI**：已新增契约回归 workflow；需 GitHub Actions 实际成功运行后勾选。
- [x] **R-04 Docker fresh-start CI**：空卷启动必须使用 MySQL、ready 正常、重启数据不丢。
- [x] **R-05 发布清单实化**：替换占位符，写明域名、脚本、备份、迁移、回滚和人工冒烟。
- [ ] **R-06 版本与 Changelog**：每次发布有版本 tag、镜像 tag/digest、CHANGELOG 和部署记录。
- [ ] **R-07 当前版+回滚版**：服务器只保留当前与上一可回滚的小型发布文件和所需镜像。
- [ ] **R-08 全量完成审计**：逐条核对本文证据；工作树干净；推送 GitHub；线上关键流程通过。

## 完成记录

每完成一项在此追加：日期、编号、修改文件、验证命令/服务器证据、提交哈希、剩余风险。

### 2026-08-30

- `P0-01/P0-05/P0-07/P0-08/P0-10/P0-11`、`S-03`、`F-01/F-02/F-03/F-04`、`O-02/O-08/O-09`、`R-01/R-03`：代码见提交 `b584585`、`b78a8b9`、`ac0c406`；后端 `gofmt -w && go test ./... && go vet ./...` 通过；前端 `npm run lint:check`、`npm test -- --run`（6 files/8 tests）、`npm run build` 通过。线上 `/api/v1/health/ready` 返回 `200`、`environment=production`、`storageMode=mysql`、`degraded=false`；详情、评论分页、匿名浏览返回正常；公网端口检查仅 80/443，PromptSystem 3092/5092 为 loopback，其他 Compose 项目未变。剩余风险：CI 尚未接入，P0-04 演示账号处置和邮件发送尚未完成。
- `P0-03`：新增 `src/backend/internal/store/mysql_seed_test.go`，真实 MySQL 隔离库执行 `SeedMySQLData(db, false)`，并断言 categories>0、users=0、prompts=0；`go test ./internal/store -run TestProductionSeedDoesNotCreateDemoData -count=1` 通过。生产启动路径已使用 `!cfg.IsProduction()` 关闭演示数据。
- `P0-04`：处置前备份目录 `/srv/backups/promptsystem/20260830-demo-removal/`；`mysql.sql.gz` SHA-256=`cb3ecee7a1d10e7a407d7f246c4419bb43c7b71a1eb5ee2cfe86e7a75bb23d48`，`uploads.tar.gz` SHA-256=`9a2609e8613823970486698dbca7c9f8c3e07456dcac88d0749e1f403f8fd3ca`；服务器执行 `sha256sum -c`、`gzip -t`、`tar -tzf` 均通过。用户 ID 1-6（Astra Lab、Nora Chen、Delta Forge、Mica Studio、Ops Lantern、North Queue）已禁用、密码置 NULL、session_version+1；Prompt ID 101-106 已转移至无密码官方账号 ID 7（PromptOS Official）。已知演示密码 `PromptOS123!` 登录返回 HTTP 401；数据库复核 `status=0,password_is_null=1,session_version=1`（1-6），ID 7 `status=1,password_is_null=1`，Prompt 101-106 `user_id=7,status=1`。
- `P0-12`：备份 `/srv/backups/promptsystem/20260830-demo-removal/` 经 SHA-256、`gzip -t`、`tar -tzf` 校验后，使用临时 MySQL 卷 `promptos_restore_mysql_20260830` 恢复，复核 `users=6,prompts=6,published=6`；上传归档解包成功。上一 release 镜像 `promptsystem-backend:20260829-a9ba2cf` 连接恢复库和临时 Redis 后 ready 返回 `200`、`environment=production`、`storageMode=mysql`、`degraded=false`。演练资源由 trap 清理并复核无容器、网络、卷或临时目录残留。以备份开始至恢复 ready 约 50 秒（演练 RTO）；备份时间点为备份命令完成时（RPO 取决于备份间隔，当前尚未配置定时备份）。首次失败原因为演练未注入旧版必需的 OAuth 占位配置，已修正并重跑成功。
- `R-05`：`docs/RELEASE-CHECKLIST.md` 已写入正式域名、SSH/发布脚本、Compose 项目名、loopback 端口、备份校验、迁移、回滚和人工冒烟要求；未凭空勾选版本 tag/digest。
- `S-06`：`isSafeInternalPath` 限制长度并拒绝协议相对、编码双斜杠、反斜杠和控制字符；Vitest 覆盖安全/恶意 redirect 变体。
- `F-07`：API 开关开启时首页/详情/相关推荐/搜索失败不再回退 mock 数据，显示错误或空状态；mock 仅在显式关闭 API 时使用。
- `S-01`：新增 `TestLogoutRevokesTokenImmediately`，验证登出将 JWT JTI 写入 Redis denylist，旧 token 随后返回 401。
- `P0-09/A-08/D-01/F-04/F-05/O-03`：本批新增浏览历史分页加载更多与计数、评论失败重试、后端相关推荐查询、可配置 MySQL 连接池、uploads 用户外键迁移、Compose 日志轮转与 no-new-privileges；后端 `go test ./...`、`go vet ./...`，前端 lint、8 tests、生产 build 通过。剩余风险：外键迁移和 Compose 生产配置需在下一次发布窗口实际执行并验收；前端依赖审计仍有 Vite/esbuild 开发依赖风险。
- `R-01/R-03`：GitHub Actions `frontend-ci` 运行 `33269634855` 成功，包含 lint、8 个 Vitest、生产 build 和契约回归；`security-scan` 运行 `33269955011` 成功，npm audit 与 govulncheck 均通过。`R-02` 等待后端最新运行成功后再勾选。
- `R-02/R-04`：GitHub Actions `backend-ci` 运行 `33270442024` 成功，包含 gofmt、vet、race、MySQL/Redis 集成、迁移矩阵、build，以及 Docker fresh-start、ready 和 compose 健康检查。安全扫描最新运行 `33270441831` 成功。
