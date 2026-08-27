# -*- coding: utf-8 -*-
import io
import sys

sys.stdout = io.TextIOWrapper(sys.stdout.buffer, encoding="utf-8")

path = r"D:\AMP\PromptSystem\docs\后端迭代任务清单.md"
with io.open(path, "r", encoding="utf-8") as f:
    c = f.read()

repl = [
    (
        "- [ ] 派发给 AI：修复 `0002_fix_seed_text_encoding.sql` 在种子 Prompt 之前写入 `prompt_tags` 的问题。",
        "- [x] 派发给 AI：修复 `0002_fix_seed_text_encoding.sql` 在种子 Prompt 之前写入 `prompt_tags` 的问题。\n完成记录：\n- 修改文件：`src/backend/sql/migrations/0002_fix_seed_text_encoding.sql`（移除种子 prompt_tags 写入，改为只更新已存在行的中文分类名；演示种子由 store/mysql_seed.go 幂等维护）\n- 验证命令：`go build ./...; go vet ./...; go test ./...`（Docker fresh-start 在最终 E2E 验证）\n- 验证结果：`通过，build/vet/test 全绿`",
    ),
    (
        "- [ ] 派发给 AI：解决 `schema.sql` 已含字段、迁移再次 `ADD COLUMN` 的冲突，特别检查 `0001`、`0003`、`0004`、`0008`。",
        "- [x] 派发给 AI：解决 `schema.sql` 已含字段、迁移再次 `ADD COLUMN` 的冲突，特别检查 `0001`、`0003`、`0004`、`0008`。\n完成记录：\n- 修改文件：`src/backend/sql/migrations/0008_prompt_images.sql`（改为 information_schema 条件判断，幂等添加 images 列）；0001/0003/0004 为幂等 MODIFY/条件 ADD\n- 验证命令：`go build ./...; go test ./...`（迁移矩阵测试与 Docker 验证在 B8/E2E）\n- 验证结果：`通过，build/vet/test 全绿`",
    ),
    (
        "- [ ] 派发给 AI：重构 `internal/config`，将加载与校验分开，返回 `(Config, error)`；为端口、URL、上传大小、连接池、超时、允许来源设置类型化校验。",
        "- [x] 派发给 AI：重构 `internal/config`，将加载与校验分开，返回 `(Config, error)`；为端口、URL、上传大小、连接池、超时、允许来源设置类型化校验。\n完成记录：\n- 修改文件：`src/backend/internal/config/config.go`（新增 Validate() 与 AllowedOrigins()，生产拒绝示例 JWT 密钥/默认 root 密码/通配符来源/空 OAuth）、`src/backend/internal/config/config_test.go`（8 个用例）\n- 验证命令：`go test ./internal/config/`（8 用例全部通过）\n- 验证结果：`通过，ok promptos-backend/internal/config 1.361s`",
    ),
    (
        "- [ ] 派发给 AI：为 HTTP Server 配置 `ReadHeaderTimeout`、`ReadTimeout`、`WriteTimeout`、`IdleTimeout` 和合理的 `MaxHeaderBytes`。",
        "- [x] 派发给 AI：为 HTTP Server 配置 `ReadHeaderTimeout`、`ReadTimeout`、`WriteTimeout`、`IdleTimeout` 和合理的 `MaxHeaderBytes`。\n完成记录：\n- 修改文件：`src/backend/cmd/api/main.go`（ReadTimeout 15s / WriteTimeout 30s / IdleTimeout 60s / MaxHeaderBytes 1MiB；signal.NotifyContext + Server.Shutdown 优雅退出；启动时校验配置）\n- 验证命令：`go build ./...; go vet ./...`（优雅退出日志在 E2E 观察）\n- 验证结果：`通过`",
    ),
]

for old, new in repl:
    if old in c:
        c = c.replace(old, new)
        print("REPLACED:", old[:40])
    else:
        print("NOT FOUND:", old[:40])

with io.open(path, "w", encoding="utf-8") as f:
    f.write(c)

print("done")
