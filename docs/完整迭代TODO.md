# PromptOS 完整迭代 TODO

更新时间：2026-09-04

当前进度：53/81 已完成，28 项待完成（2026-09-04）。

本文是 PromptOS 后续开发、生产加固和服务器运维的唯一总清单。执行时遵守 `E:\Web\服务器部署总说明.md`、`AGENTS.md`、`docs/API契约.md` 和 `docs/DEPLOYMENT.md`。只有在代码、测试、服务器状态或恢复演练提供可复核证据后才允许将 `[ ]` 改成 `[x]`。

## 当前基线

- 生产域名：`https://promptsystem.isoumao.top`
- Compose 项目：`promptsystem`；当前发布目录：`/srv/releases/promptsystem/20260830-b584585`；上一回滚目录：`/srv/releases/promptsystem/20260829-a9ba2cf`
- 服务器：7.7 GiB 内存、无 Swap、58 GiB 根盘（2026-09-04 实测可用内存约 3.3 GiB、剩余约 30 GiB，根盘使用率 48%）
- 公网只开放 80/443；PromptOS 前端/后端分别绑定 `127.0.0.1:3092/5092`
- RustFS 转发链路可用，但 PromptOS 尚无独立 bucket/低权限凭据，上传暂存独立 Docker 卷
- 服务器已有多个 Compose 项目，禁止全局 prune、删除未知卷、改动其他项目或在服务器编译源码

## 服务器条件适配与执行顺序

以下顺序把 `E:\Web\服务器部署总说明.md` 的约束直接转成迭代门槛。代码、测试和镜像在本机构建；公网服务器只执行拉取/加载、备份、迁移、切换和验证。

| 阶段 | 迭代项 | 服务器验收门槛 |
|---|---|---|
| 1. 契约与代码 | `A-02`-`A-07`、`A-09`、`D-02`-`D-10`、`S-02`-`S-09`、`F-06`、`F-08`-`F-11` | 本机单元/集成/E2E、`go vet`、前端 lint/build、契约回归和安全扫描通过；不占用生产资源 |
| 2. 低内存运维 | `A-10`、`A-11`、`O-01`、`O-04`-`O-07`、`O-10` | 任务使用 systemd timer/cron；无 Swap 时串行执行；告警接收地址、RustFS 链路和 nginx reload 均有实测记录 |
| 3. 数据与对象 | `P0-06`、`D-11`-`D-14`、`S-10`、`S-13` | SMTP 真实凭据、独立 RustFS bucket/低权限凭据、跨故障域副本和备份恢复证据齐备；条件缺失保持未勾选 |
| 4. 发布自动化 | `A-12`、`R-06`、`R-07`、`R-08` | 本机构建并固定镜像 tag/digest；服务器保留当前+上一版；备份校验、迁移、ready、HTTPS 冒烟和回滚记录完整 |

服务器资源预算和不可变约束：

- 服务器约 7.7 GiB 内存且无 Swap；发布前后保持至少 2 GiB 可用内存，镜像加载、备份、迁移、重启、恢复演练不得并行。
- 根盘约 58 GiB；达到 70%/80%/90% 分别触发观察、清理和发布阻断。只清理已确认的悬空镜像/build cache，不执行全局或 volume prune。
- 生产 Compose 项目固定为 `promptsystem`，应用只监听 `127.0.0.1:3092/5092`，公网仅 80/443；升级不得新建项目名、替换卷名或停止其他项目。
- 发布目录必须同时保留当前版本 `/srv/releases/promptsystem/<current>` 与上一可回滚版本；服务器不保存源码、node_modules、构建上下文或压缩包。
- 当前上传仍在独立 Docker 卷；在 `D-12/D-13` 的独立 bucket、凭据和迁移验收完成前，不得宣称已完成 RustFS 迁移或删除原卷。

外部前置条件未满足时的处理：真实 SMTP、告警通知接收端、PromptOS 独立 RustFS bucket/凭据、跨故障域备份目标任一缺失，对应项目只能记录为阻塞/待验证，不能用本地 mock、现有他站点凭据或 CI 通过替代生产证据。

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

- [x] **A-01 统一数据库初始化**：空库、旧 schema、部分迁移库通过同一入口到达最终版本，不再人工先导入 `schema.sql`。
- [x] **A-02 Service 层**：认证、互动、举报、上传引用和缓存失效从 handler 抽到窄业务服务。
- [x] **A-03 Store 语义一致**：MySQL/内存实现通过同一契约测试；内存实现仅限开发和测试。
- [x] **A-04 前端数据层收敛**：View 不重复直连 API；Store 统一 loading/error/empty/success、缓存和竞态。
- [x] **A-05 API 单一契约**：引入 OpenAPI/JSON Schema 或等价生成校验，TypeScript DTO 不手工漂移。
- [x] **A-06 统一错误模型**：所有接口返回稳定 `errorCode`，禁止返回内部错误文本或通过字符串分支。
- [x] **A-07 请求取消与竞态**：搜索、详情、评论使用 AbortController/请求序列，旧响应不能覆盖新状态。
- [x] **A-08 数据库连接池**：配置最大/空闲连接与生命周期，并采集连接池指标。
- [x] **A-09 SQL 分页与排序**：评论热门排序在数据库完成；主要列表执行 `EXPLAIN` 并补必要复合索引。
- [ ] **A-10 轻量后台任务**：上传回收、数据审计、备份校验优先使用 systemd timer/cron，不新增高内存常驻套件。
- [x] **A-11 可观测性**：提供请求、错误、延迟、数据库、Redis、上传和任务指标。
- [ ] **A-12 发布架构自动化**：本机构建、镜像校验、上传、迁移、切换、验证和回滚脚本化；服务器不编译。

## D 数据

- [x] **D-01 上传所有者约束**：`uploads.owner_id` 建立用户引用或等价强校验。
- [x] **D-02 多态目标完整性**：likes/favorites/reports/comments 写入前校验目标，并定期扫描孤儿记录。
- [x] **D-03 删除生命周期**：统一 Prompt、评论、用户、举报和上传的禁用/软删/回收规则。
- [x] **D-04 上传垃圾回收**：草稿删除、替换图片、发布失败和注销产生的未引用对象延迟回收。
- [x] **D-05 汇总计数审计**：定期核对 views/likes/favorites 与明细表，并对偏差告警。
- [x] **D-06 事务一致性**：发布、互动、评论、举报、关注的明细与汇总在同一事务或有明确补偿。
- [x] **D-07 标签规范化**：统一大小写、空白、长度和唯一性；热门标签由后端聚合。
- [x] **D-08 公开查询边界**：草稿、删除内容、禁用用户内容不能出现在首页和搜索。
- [x] **D-09 数据字典**：记录全部表、字段、状态值、索引、外键和保留期限。
- [x] **D-10 个人数据能力**：账号注销、数据导出、浏览历史清除和注销后会话失效。
- [ ] **D-11 MySQL 定时备份**：每日逻辑备份、压缩、校验、保留和失败告警。
- [ ] **D-12 PromptOS 独立 RustFS bucket**：使用独立最小权限凭据，不复用其他站点主凭据。
- [ ] **D-13 上传迁移与双副本**：从 Docker 卷迁移到 RustFS，另保留不同故障域副本。
- [ ] **D-14 恢复演练**：每月恢复数据库和对象，使用上一应用版本通过 ready 与关键流程。

## S 安全

- [x] **S-01 会话撤销**：登出调用后端；密码重置、禁用、注销使旧 token 立即失效。
- [ ] **S-02 JWT 存储策略**：先完成 CSP/XSS 审计，再评估迁移 HttpOnly+Secure+SameSite Cookie。
- [x] **S-03 GitHub OAuth 开关**：显式 enable 配置；未配置时前端不显示可点击入口，启用时校验 redirect/state/code。
- [ ] **S-04 CSP**：从 Report-Only 开始制定 Vue 生产 CSP，再切换强制模式。（已加入 Report-Only，待生产报告复核后切换强制）
- [ ] **S-05 CORS/CSRF**：生产仅允许正式域名；Cookie 会话启用 CSRF 防护。
- [x] **S-06 重定向安全**：站内 redirect 覆盖双斜杠、反斜杠、协议相对和编码变体。
- [x] **S-07 分维度限流**：登录、注册、验证码、重置、评论、举报、搜索、上传按 IP/账号/用户限制。
- [x] **S-08 内容安全**：Prompt/评论/用户名/标签/Markdown/URL 统一长度、控制字符和脚本校验。
- [x] **S-09 上传安全与配额**：MIME、解码、像素、格式、并发、单用户日配额和总容量限制。
- [ ] **S-10 Redis 隔离**：独立网络、ACL/密码，禁止公网访问。
- [x] **S-11 容器加固**：非 root、`no-new-privileges`、`cap_drop`、可行的只读根文件系统、资源与日志限制。
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
- 最新 CI：提交 `2f9201a` 的 backend-ci `33275544929` 和 security-scan `33275544928` 均成功；backend 包含 gofmt、vet、race、MySQL/Redis 集成、迁移矩阵、build 和 Docker ready，安全扫描包含 npm audit、govulncheck 与 Compose 策略。

### 2026-09-04

- `A-01`：`src/backend/internal/database/migrate.go` 在当前数据库无任何表时自动应用随镜像发布的 `sql/schema.sql` 基线，再通过 `schema_migrations` 顺序执行增量迁移；已有 schema 或部分迁移库不会覆盖数据。基线执行跳过仅用于已创建数据库的 `CREATE DATABASE`/`USE` 引导语句，兼容生产最小权限迁移账号。开发 Compose 移除 MySQL 的 `schema.sql` 隐式挂载并显式创建与 backend 配套的 `promptos_app` 账号，`README.md`、`docs/DEPLOYMENT.md` 和迁移 README 改为单一启动入口。
- 验证：`go test ./...`、`go vet ./...`、`docker compose config --quiet`、`git diff --check` 通过；临时 MySQL 实测 `TestMigrationMatrix` 的 fresh（真正空库）、baseline、partial 三场景及二次运行幂等性全部通过。提交 `15ea2f9` 的 GitHub Actions backend `33781675791`、frontend `33781675792`、security `33781675829` 全部成功。生产尚未在本批发布，下一次发布仍需按备份、迁移、ready 和回滚流程执行。
- `A-06`：存储层统一使用 sentinel error（用户、Prompt、评论、举报、关注等），API 通过 `errors.Is` 集中映射稳定 `errorCode`；未知存储错误统一 `500 INTERNAL_ERROR`，不再返回内部错误文本或用错误字符串分支。响应写出层为遗漏的 4xx/5xx 信封补默认稳定码。新增登录、自己关注、不存在 Prompt 评论、越权更新和未知错误不泄露回归测试。`go test ./...`、`go vet ./...`、`go build ./cmd/api`、`docker compose config --quiet`、`git diff --check` 通过；本机 race 未执行成功（Windows 环境缺少 `gcc`），GitHub Actions backend `33784076660`（含 Linux `go test -race`、迁移矩阵、Docker health）和 security `33784076727` 均成功。提交 `3cdb64e` 已推送，生产尚未发布。
- `A-07`：API 客户端支持 `AbortSignal`；Prompt Store 对首页列表、详情、评论及加载更多请求执行取消和请求序号校验；搜索页取消旧搜索并在卸载时终止；详情页卸载时取消待处理请求；Axios 取消不再触发网络错误提示。新增旧详情/评论响应不覆盖新状态测试。前端 `npm test -- --run`（7 files/14 tests）、`npm run lint:check`、`npm run build`、`git diff --check` 通过，生产尚未发布。
- `D-07`：应用层和 MySQL 种子统一 trim、合并空白、大小写规范化、长度限制和去重；新增 `0015_normalize_prompt_tags.sql` 清理历史标签冲突并保持幂等，热门标签继续由后端聚合。新增 MySQL 隔离库迁移回归测试，验证旧标签归一化为单个 canonical 值且重复迁移无变化；`go test ./internal/store ./internal/database` 通过（未设置 `PROMPTOS_TEST_MYSQL_DSN` 时集成测试按约定跳过）。生产尚未发布。
- `D-08`：内存和 MySQL 公开读取统一要求 Prompt 已发布且作者账号启用；首页汇总、分类计数、搜索、公开详情、历史、收藏/点赞列表及互动入口均过滤禁用作者内容，避免已发布数据因作者禁用而泄露。新增内存回归测试和 MySQL 隔离库集成测试；本机 `go test ./...`、`go vet ./...`、`go build ./cmd/api` 通过，GitHub Actions backend `33788170152` 的真实 MySQL 集成/race、迁移矩阵和 Docker health 全部通过。生产尚未发布。
- `A-09`：MySQL 评论列表按规范化 `latest`/`oldest`/`popular` 在 `LIMIT/OFFSET` 前排序，新增根评论时间与热度复合索引并保持 `schema.sql`/增量迁移一致；集成测试验证热门结果不会被分页截断，并用 `EXPLAIN FORMAT=JSON` 断言 `idx_target_parent_likes` 计划候选。GitHub Actions backend `33788170152` 全部通过，生产尚未发布。
- `A-03`：新增 `runPromptManagerContract` 共享契约测试，内存和 MySQL 均覆盖创建与标签规范化、分页总数、互动幂等、浏览历史去重、删除隐藏及非所有者更新权限；修正 MySQL 非所有者更新返回 `ErrPromptForbidden` 的语义。生产尚未发布，真实 MySQL 由 backend CI 集成作业验证。
- `D-09`：新增 `docs/数据字典.md`，按当前 `schema.sql` 记录全部 12 张表的字段、状态枚举、索引、外键、多态目标约束、软删/级联和上传保留策略，并明确迁移与审计入口；`docker compose config --quiet`、`git diff --check` 通过。生产尚未发布。
- `D-02`：新增 `AuditMySQLPolymorphicIntegrity` 只读审计、`promptos-integrity-audit` 一次性命令和真实 MySQL 回归测试，扫描 comments/likes/favorites/reports 的未知 `target_type` 与孤儿目标；既有写入路径已在事务前校验公开 Prompt/Comment 目标。审计发现问题退出 `1`，连接或执行失败退出非 `0`，无异常退出 `0`，不自动修改数据。命令随 backend 镜像发布，部署文档给出带 `flock` 的 systemd timer/cron 串行执行方式，适配无 Swap 服务器。`gofmt`、`go test ./...`、`go vet ./...`、API/审计二进制构建、前端 lint/Vitest/build、`docker compose config --quiet` 和 `git diff --check` 通过；GitHub Actions backend `33790437831` 的 Linux race、真实 MySQL、迁移矩阵和 Docker health 全部通过，security `33790437800` 通过。生产尚未部署，告警接收端仍未配置；本机 Docker build 因 Docker Desktop 无法访问 Docker Hub 未完成。
- `D-10`：新增 `GET /api/v1/user/data-export`、`DELETE /api/v1/user/history` 和 `DELETE /api/v1/user/account`；内存/MySQL Store 均支持本人 Prompt 导出、历史清除和事务化注销。注销禁用账户、清除密码/GitHub 绑定、匿名化用户名/邮箱、递增 `session_version`，清理个人点赞/收藏/举报/关注/浏览明细并修正 Prompt/评论计数；保留 Prompt/评论记录以满足审计和外键保留策略，禁用作者内容不再公开。个人中心增加导出、清空浏览记录和注销入口。`go test ./...`、MySQL `TestMySQLDeleteAccountAnonymizesAndClearsPersonalRows`、前端 `npm run lint:check`、`npm test -- --run`（7 files/14 tests）和 `npm run build` 通过。生产尚未发布，注销后上传对象仍由 `D-04` 延迟回收任务负责，真实 SMTP/告警/RustFS 依赖项目保持未验收。
- `A-11/D-05/S-07`：新增轻量 Prometheus `/metrics` 端点，记录 HTTP 请求/错误/平均延迟、上传、任务和 MySQL/Redis/上传依赖状态；新增只读 Prompt 计数审计并纳入低峰维护命令；登录、注册、验证码、重置、评论、举报、搜索、互动和上传按 IP/邮箱/用户分维度限流，生产 Redis 不可用时 fail-closed。新增限流与 metrics 回归测试。后端 `go test ./...`、`go vet ./...`、三个二进制构建和前端全量校验通过；生产 timer/告警尚未配置，故 `A-10/O-01` 仍待服务器验收。
- `S-11`：生产 Compose 为 backend/frontend 增加 `no-new-privileges`、最小 Linux capabilities、只读根文件系统、tmpfs、PID/内存上限，并保留现有日志轮转和 loopback 端口边界；security workflow 增加策略断言。生产尚未重启验证，前端 nginx 的 `NET_BIND_SERVICE` 例外需在下一发布窗口用 `config`、启动和 HTTPS 冒烟验收。
- `S-08`：`src/backend/internal/store/moderation.go` 统一校验 Prompt、评论、用户名和简介的 UTF-8、控制字符、长度及脚本/危险 URL 片段；标签和图片 URL 同步拒绝控制字符与超长值。内存/MySQL 写入路径共用规则，API 将校验错误映射为稳定 `errorCode`。新增边界回归测试；`gofmt -l .`、`go test ./...`、`go vet ./...`、`git diff --check` 通过。生产尚未发布，CSP 强制模式和浏览器渲染验证仍由 `S-04`/`F-10` 负责。
- `A-04`：将 `SearchView` 的 API 直连、AbortController、请求序号、分页、去重、错误和 loading 状态收敛到 `src/frontend/src/stores/prompt.ts`；视图仅保留路由、筛选和展示职责，卸载时由 Store 统一取消搜索。新增搜索分页去重与旧请求响应隔离回归测试；前端 `npm test -- --run`（7 files/16 tests）、`npm run lint:check`、`npm run build`、`git diff --check` 通过。生产尚未发布。
- `S-09`：上传入口保留整请求/单文件大小限制、真实 MIME/扩展名一致性、标准解码和 20MP 像素上限，并新增 `UPLOAD_MAX_CONCURRENT` 并发槽、`UPLOAD_DAILY_QUOTA_MB` 用户日配额和 `UPLOAD_TOTAL_QUOTA_MB` 总容量；Redis 实现按用户/UTC 日期原子字节计数，开发/测试无 Redis 时使用进程内计数，数据库已用量与进程内预留量共同防止并发超卖。新增 `ActiveUploadBytes` Store 契约、Redis `IncrementBy`、配额/并发回归测试；`go test ./...`、`go vet ./...`、`docker compose config --quiet`、`git diff --check` 通过。生产尚未发布，RustFS 独立 bucket 和跨故障域副本仍由 `D-12/D-13` 负责。
- `A-05`：新增 `docs/api-contract.schema.json` 作为 v1 共享 DTO 的 JSON Schema，`scripts/generate-api-contract.mjs` 生成 `src/frontend/src/types/api-contract.generated.ts`；前端 `User`、`Prompt`、`Comment`、分页和响应信封类型改为生成类型别名。新增 `npm run contract:check`，CI 在契约回归前验证生成文件无漂移；Schema 变更会触发前端 workflow。`npm run contract:check`、`npm test -- --run`（7 files/16 tests）、`npm run lint:check`、`npm run build`、`git diff --check` 通过。生产尚未发布。
- `A-02`：新增 `src/backend/internal/service` 业务层；`AuthService` 统一认证、账户导出/注销、关注和 JWT 撤销，`PromptService` 统一发布/更新/删除、互动、举报、浏览、上传引用校验及内容缓存失效，`CommentService` 统一评论创建、点赞和举报。API handler 仅保留 HTTP 解析、鉴权、限流和稳定错误映射；正式装配与直接构造的测试 server 均通过惰性 accessor 使用同一 Service。新增 Service 回归测试；`go test ./...`、`go vet ./...`、`gofmt -l .`、`git diff --check` 通过。生产尚未发布。
- `D-03`：明确并实现 Prompt 上传引用生命周期：上传先为 `pending`，Prompt 写入成功后才转 `referenced`；Prompt 更新/删除后按该用户保留的发布内容和草稿实际引用集合，将不再使用的旧对象退回 `pending`，交由延迟回收。评论/举报继续保留软删和审计记录，禁用用户内容不公开，账户注销沿用禁用/匿名化策略。新增 `ListReferencedUploadKeys`、`UnreferenceUploadsByOwner` 及内存/MySQL 实现和回归测试；`go test ./...` 通过。生产尚未发布。
- `D-04`：维护命令的上传回收核心拆为可测试的 `cleanupUploadRecords`；只处理超过安全窗口的 `pending` 对象，provider 不匹配或对象删除失败保留 `pending`，仅在物理删除成功后转 `trashed`，支持下一轮重试。内存回归覆盖最近对象不删、旧对象删除、失败/错 provider 保留；`go test ./cmd/maintenance ./internal/service ./internal/store` 通过。生产 timer 尚未配置，待 `A-10/O-01` 服务器验收。
- `D-06`：MySQL Prompt/评论举报把公开目标校验、幂等写入和结果读取放入同一事务；关注/取关按稳定用户锁顺序在同一事务内写入关系并计算 follower/following 汇总；Prompt/标签写入原有事务保持不变；上传引用元数据使用单事务批量标记，跨 Prompt/上传 Store 失败时由 `FinalizeUploadReferences` 先重试标记当前引用再回收旧引用，补偿成功可安全返回，双失败保留可重试状态。新增 `TestMySQLReportAndFollowTransactions`（并发关注/举报、重复操作、删除目标拒绝）和 `TestPromptServiceCompensatesUploadReferenceFailure`；本机 `go test ./...`、`go vet ./...`、API/维护/完整性审计构建、`git diff --check` 通过；使用临时隔离 MySQL 库实测 `go test ./internal/store -count=1` 和 `TestMigrationMatrix` 均通过，测试库已清理。生产尚未发布。
- 线上只读核验（2026-09-04）：服务器 `free -h` 显示总内存 7.7 GiB、可用约 3.3 GiB、Swap 0；`df -h /` 显示 58G 总量、28G 已用、30G 可用（48%）。`docker compose ls` 确认 `promptsystem` 使用 `/srv/releases/promptsystem/20260830-b584585/docker-compose.yml`，仅保留 `20260830-b584585` 与 `20260829-a9ba2cf` 两个 release 目录；frontend/backend 仍分别绑定 `127.0.0.1:3092/5092`，公网入口为 80/443，其他 Compose 项目保持运行。线上 ready 返回 `200`、`environment=production`、`storageMode=mysql`、`degraded=false`。RustFS 转发服务 active，当前实测 bridge 监听为 `172.21.0.1:13902`、隧道为 `127.0.0.1:13900`；总说明中的旧 `172.17.0.1` 已修正。当前线上容器仍是旧发布策略（`ReadonlyRootfs=false`、未设置 cap/pids 限制），故 `S-11` 生产验收、`D-12/D-13` RustFS 迁移和所有告警/timer 项目保持未完成。
