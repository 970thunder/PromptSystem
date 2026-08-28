@echo off
REM ============================================================================
REM  PromptOS 一键启动（Windows 双击入口）
REM  固定端口：前端 28301 / 后端 28302 / MySQL 28303 / Redis 28304
REM  调用 Git Bash 执行 scripts/start-dev.sh
REM ============================================================================
setlocal

REM 查找 Git Bash（常见安装路径）
set "BASH="
for %%p in (
  "%ProgramFiles%\Git\bin\bash.exe"
  "%ProgramFiles(x86)%\Git\bin\bash.exe"
  "%LocalAppData%\Programs\Git\bin\bash.exe"
  "%ProgramFiles%\Git\usr\bin\bash.exe"
) do (
  if exist %%p set "BASH=%%~p"
)

if not defined BASH (
  where bash >nul 2>nul && set "BASH=bash"
)

if not defined BASH (
  echo [ERROR] 未找到 Git Bash，请安装 Git for Windows 后重试。
  pause
  exit /b 1
)

echo [INFO] 使用 Git Bash: %BASH%
"%BASH%" "%~dp0scripts\start-dev.sh" %*
if errorlevel 1 (
  echo.
  echo [ERROR] 启动失败，请查看 scripts 上方日志输出。
)
pause
