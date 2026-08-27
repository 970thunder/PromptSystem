# B0-01 基线记录（由编排者记录，2026-08-27）

## 环境
- Go: go1.27.0 windows/amd64
- Node: v24.19.0, npm 11.17.0
- Docker: 29.7.2, Compose v5.4.0（Docker Desktop 已启动）

## 后端状态
- cd D:\AMP\PromptSystem\src\backend
- go version: go1.27.0 windows/amd64
- go build ./...: 通过（无输出，exit 0）
- go vet ./...: 通过（无输出，exit 0）
- go test ./...: 全部 ok（cmd/api 无测试文件，internal/api, auth, database, store 均 ok）

## 前端状态
- cd D:\AMP\PromptSystem\src\frontend
- npm run lint:check: 通过
- npm run build: 通过（警告：index chunk 1494 kB > 500 kB，gzip 420 kB）

## Docker 服务
- docker compose up -d --build: 成功，mysql/redis 健康，backend/frontend 运行
- http://localhost:8080/api/v1/health: {"code":200,...,"storageMode":"mysql"}
- http://localhost:3000/: 200

## 工作树状态（本轮修改前）
- 删除（未提交）：AGENTS.md, CLAUDE.md, TODO.md, docs/superpowers/plans/..., docs/superpowers/specs/...
- 修改（未提交）：src/frontend/package-lock.json
- 未跟踪：docs/前端迭代任务清单.md, docs/后端迭代任务清单.md
- 分支 master，与 origin/master 同步

## 结论
- 基线通过：go build/vet/test 全绿、前端 lint/build 全绿、Docker 四服务健康、health 显示 storageMode=mysql。
- 已知工程问题（见清单 0.2）在本轮解决。
