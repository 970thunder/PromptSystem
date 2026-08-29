# scripts/release.ps1 — PromptOS 发布入口（骨架）
# 流程：prepare → check → deploy → verify → record（见 E:\Web\GOVERNANCE.md 第 4 节）
# 用法：powershell -File scripts\release.ps1 -Version v1.0.1
param(
    [Parameter(Mandatory = $true)][string]$Version,
    [switch]$SkipTests   # 仅紧急热修时用，事后必须补跑
)
$ErrorActionPreference = "Stop"
Set-Location (Split-Path $PSScriptRoot -Parent)   # 仓库根

function Fail($m) { Write-Host "x $m" -ForegroundColor Red; exit 1 }
function Step($m) { Write-Host "==> $m" -ForegroundColor Cyan }

# ---- 0. 前置检查 ----
Step "0. 前置检查"
if (-not (Test-Path "docs\CHANGELOG.md")) { Fail "缺少 docs\CHANGELOG.md（契约文件）" }
$dirty = git status --porcelain
if ($dirty) { Fail "工作区不干净，先提交或清理：`n$dirty" }
if (-not (Select-String -Path "docs\CHANGELOG.md" -Pattern ([regex]::Escape($Version)) -Quiet)) {
    Fail "CHANGELOG 中没有 $Version 条目——先写发布说明再发布"
}
Write-Host "OK: git 干净，CHANGELOG 含 $Version"

# ---- 1. 本地验证 ----
if (-not $SkipTests) {
    Step "1. 后端静态检查 + 测试"
    Push-Location src\backend
    $fmt = gofmt -l .
    if ($fmt) { Pop-Location; Fail "gofmt 未通过：`n$fmt" }
    go vet ./...;  if ($LASTEXITCODE -ne 0) { Pop-Location; Fail "go vet 失败" }
    go test ./...; if ($LASTEXITCODE -ne 0) { Pop-Location; Fail "go test 失败" }
    Pop-Location

    Step "1b. 前端构建"
    Push-Location src\frontend
    npm run build; if ($LASTEXITCODE -ne 0) { Pop-Location; Fail "前端构建失败" }
    Pop-Location
}

# ---- 2. 服务器备份（保留最近 3 版） ----
Step "2. 服务器备份"
# TODO(部署方式确认后填写)：
#   ssh <server> "mysqldump -u<user> -p ... > /opt/backups/promptos/$Version.sql"
#   uploads 数据卷一并纳入备份

# ---- 3. 部署 ----
Step "3. 部署"
# TODO：二选一确认后填死——
#   A. 服务器 git pull + docker compose build && docker compose up -d
#   B. 本地构建镜像按 $Version 打 tag 推送，服务器 compose 引用该 tag（推荐：镜像 tag 即回滚点）

# ---- 4. 健康检查 ----
Step "4. 健康检查"
# TODO：curl -fsS https://<域名>/ 与 API 健康端点
#   失败则回滚：切回上一版镜像 tag / release 目录，必要时恢复数据库备份

# ---- 5. 记录 ----
Step "5. 打 tag"
git tag -a $Version -m "Release $Version"
Write-Host "完成。请在 CHANGELOG 补'已发布'状态后推送：git push origin master --tags"
