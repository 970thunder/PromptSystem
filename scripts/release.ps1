# PromptOS production release pipeline.
# Builds locally; the server receives only a compose file and runtime images.
[CmdletBinding()]
param(
    [Parameter(Mandatory = $true)][string]$Version,
    [string]$Domain = 'promptsystem.isoumao.top',
    [string]$ServerHost = '103.42.182.205',
    [string]$ServerUser = 'root',
    [int]$SshPort = 2680,
    [string]$SshKey = 'E:\Web\服务器密钥\foxi_103.42.182.205',
    [string]$ProjectName = 'promptsystem',
    [bool]$EmailAuthEnabled = $true,
    [string]$ImageArchivePath = '',
    [switch]$SkipTests,
    [switch]$SkipDeploy
)
$ErrorActionPreference = 'Stop'
Set-Location (Split-Path $PSScriptRoot -Parent)

function Step([string]$Message) { Write-Host "==> $Message" -ForegroundColor Cyan }
function Fail([string]$Message) { throw $Message }
function Require([string]$Command) { if (-not (Get-Command $Command -ErrorAction SilentlyContinue)) { Fail "缺少命令：$Command" } }
function Invoke-Checked([string]$Command, [string[]]$Arguments) {
    & $Command @Arguments
    if ($LASTEXITCODE -ne 0) { Fail "$Command 执行失败（退出码 $LASTEXITCODE）" }
}

if ($Version -notmatch '^[0-9A-Za-z][0-9A-Za-z.-]{2,63}$') { Fail '版本只能包含字母、数字、点和连字符' }
Require git; Require docker; Require npm; Require ssh; Require scp
if (-not (Test-Path $SshKey)) { Fail "SSH 私钥不存在：$SshKey" }
if (-not (Select-String -Path 'docs\CHANGELOG.md' -Pattern ([regex]::Escape("[$Version]")) -Quiet)) { Fail "CHANGELOG.md 缺少 [$Version] 条目" }
$dirty = git status --porcelain
if ($dirty) { Fail "工作区不干净，请先提交：`n$dirty" }

if (-not $SkipTests) {
    Step '运行后端静态检查与测试'
    Push-Location src\backend
    try {
        $formatted = gofmt -l .
        if ($formatted) { Fail "gofmt 未通过：$formatted" }
        Invoke-Checked go @('vet', './...')
        Invoke-Checked go @('test', './...')
    } finally { Pop-Location }

    Step '运行前端 lint、测试和生产构建'
    Push-Location src\frontend
    try {
        Invoke-Checked npm @('ci')
        Invoke-Checked npm @('run', 'lint:check')
        Invoke-Checked npm @('test', '--', '--run')
        Invoke-Checked npm @('run', 'build')
    } finally { Pop-Location }
}

$releaseRoot = Join-Path (Get-Location) "temp\release-$Version"
if (Test-Path $releaseRoot) { Remove-Item -LiteralPath $releaseRoot -Recurse -Force }
New-Item -ItemType Directory -Path $releaseRoot | Out-Null
$backendImage = "$ProjectName-backend`:$Version"
$frontendImage = "$ProjectName-frontend`:$Version"

$imageArchive = Join-Path $releaseRoot "${ProjectName}-images-$Version.tar"
Copy-Item deploy\promptsystem\docker-compose.yml (Join-Path $releaseRoot 'docker-compose.yml')

if ([string]::IsNullOrWhiteSpace($ImageArchivePath)) {
    Step '本机构建生产镜像'
    Invoke-Checked docker @('build', '--pull=false', '-t', $backendImage, 'src/backend')
    $emailArg = if ($EmailAuthEnabled) { 'true' } else { 'false' }
    Invoke-Checked docker @('build', '--pull=false', '--build-arg', 'VITE_API_BASE_URL=/api/v1', '--build-arg', 'VITE_APP_TITLE=PromptOS', '--build-arg', 'VITE_ENABLE_PROMPT_API=true', '--build-arg', 'VITE_GITHUB_OAUTH_ENABLED=false', '--build-arg', "VITE_EMAIL_AUTH_ENABLED=$emailArg", '--build-arg', 'VITE_SKILL_ENABLED=false', '--build-arg', 'VITE_PLAYGROUND_ENABLED=false', '--build-arg', 'VITE_CREATOR_ACADEMY_ENABLED=false', '--build-arg', 'VITE_MARKETPLACE_ENABLED=false', '-t', $frontendImage, 'src/frontend')

    Invoke-Checked docker @('save', '-o', $imageArchive, $backendImage, $frontendImage)
    Require gzip
    Invoke-Checked gzip @('-f', $imageArchive)
    $imageArchive = "$imageArchive.gz"
} else {
    Step '校验 CI 生产镜像归档'
    $sourceArchive = (Resolve-Path -LiteralPath $ImageArchivePath -ErrorAction Stop).Path
    if ($sourceArchive -notmatch '\.tar\.gz$') { Fail 'ImageArchivePath 必须是 .tar.gz 镜像归档' }
    Invoke-Checked docker @('load', '--input', $sourceArchive)
    $loadedImages = @(& docker image inspect $backendImage $frontendImage --format '{{.Id}}' 2>$null)
    if ($LASTEXITCODE -ne 0 -or $loadedImages.Count -ne 2) {
        Fail "CI 归档未包含预期镜像：$backendImage 和 $frontendImage"
    }
    Copy-Item -LiteralPath $sourceArchive -Destination "$imageArchive.gz"
    $imageArchive = "$imageArchive.gz"
}

$hash = (Get-FileHash $imageArchive -Algorithm SHA256).Hash.ToLowerInvariant()

if ($SkipDeploy) {
    Write-Host "已完成本地发布制品：$releaseRoot（SHA-256 $hash）" -ForegroundColor Green
    exit 0
}

$sshArgs = @('-i', $SshKey, '-p', "$SshPort", '-o', 'BatchMode=yes')
$scpArgs = @('-i', $SshKey, '-P', "$SshPort", '-o', 'BatchMode=yes')
$remote = "$ServerUser@$ServerHost"
$remoteDir = "/srv/releases/$ProjectName/$Version"
$backupDir = "/srv/backups/$ProjectName/$Version"

Step '服务器备份 MySQL 与上传卷（串行）'
$backupCommand = @"
set -eu
dir='$backupDir'
install -d -m 700 "`$dir"
. /opt/secrets/$ProjectName/app.env
docker exec promptsystem-mysql sh -c 'mysqldump -uroot -p"`$MYSQL_ROOT_PASSWORD" --single-transaction --routines --events "`$MYSQL_DATABASE"' | gzip -c > "`$dir/mysql.sql.gz"
docker run --rm --entrypoint sh -v promptsystem_promptsystem_uploads:/data -v "`$dir":/backup mysql:8.4 -c 'tar -czf /backup/uploads.tar.gz -C /data .'
sha256sum "`$dir/mysql.sql.gz" "`$dir/uploads.tar.gz" > "`$dir/SHA256SUMS"
sha256sum -c "`$dir/SHA256SUMS"
gzip -t "`$dir/mysql.sql.gz"
docker run --rm --entrypoint sh -v "`$dir":/backup mysql:8.4 -c 'tar -tzf /backup/uploads.tar.gz >/dev/null'
echo BACKUP_VERIFIED
"@
& ssh @sshArgs $remote $backupCommand
if ($LASTEXITCODE -ne 0) { Fail '服务器备份失败，终止发布' }

Step '上传 release 文件和镜像包'
& ssh @sshArgs $remote "install -d -m 755 '$remoteDir'"
if ($LASTEXITCODE -ne 0) { Fail '服务器 release 目录创建失败' }
Invoke-Checked scp ($scpArgs + @($imageArchive, "$remote`:$remoteDir/"))
Invoke-Checked scp ($scpArgs + @((Join-Path $releaseRoot 'docker-compose.yml'), "$remote`:$remoteDir/docker-compose.yml"))

Step '服务器串行加载镜像并部署原 Compose 项目'
$archiveName = "${ProjectName}-images-$Version.tar.gz"
$deployCommand = "set -eu; cd '$remoteDir'; sha256sum '$archiveName' >/tmp/${ProjectName}-$Version.sha256; test `$(cut -d ' ' -f1 /tmp/${ProjectName}-$Version.sha256) = '$hash'; gzip -dc '$archiveName' | docker load; rm -f '$archiveName'; sed -i 's/^PROMPTSYSTEM_VERSION=.*/PROMPTSYSTEM_VERSION=$Version/' /opt/secrets/$ProjectName/app.env; chmod 600 /opt/secrets/$ProjectName/app.env; docker compose -p '$ProjectName' --env-file /opt/secrets/$ProjectName/app.env -f docker-compose.yml config --quiet; docker compose -p '$ProjectName' --env-file /opt/secrets/$ProjectName/app.env -f docker-compose.yml up -d --no-build"
& ssh @sshArgs $remote $deployCommand
if ($LASTEXITCODE -ne 0) { Fail "部署失败。请用上一 release '$ProjectName' 项目名回滚，不要删除卷。" }

Step '线上健康检查与 HTTPS 验证'
$verifyCommand = "set -eu; for i in `$(seq 1 30); do curl -fsS https://$Domain/api/v1/health/ready >/tmp/promptos-ready && break || sleep 2; done; grep -q 'storageMode.*mysql' /tmp/promptos-ready; curl -fsS https://$Domain/ >/dev/null; cat /tmp/promptos-ready"
& ssh @sshArgs $remote $verifyCommand
if ($LASTEXITCODE -ne 0) { Fail '健康检查失败，请按 docs/DEPLOYMENT.md 回滚到上一 release' }

Step '健康检查通过后更新 current 发布指针'
$promoteCommand = "set -eu; ln -sfn '$remoteDir' '/srv/releases/$ProjectName/current'"
& ssh @sshArgs $remote $promoteCommand
if ($LASTEXITCODE -ne 0) { Fail 'current 发布指针更新失败；当前容器仍保持运行，请人工核对 release 目录' }

Write-Host "发布成功：$Version；镜像归档 SHA-256：$hash；备份目录：$backupDir" -ForegroundColor Green
