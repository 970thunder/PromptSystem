@echo off
REM ============================================================================
REM  PromptOS 一键停止（Windows 双击入口）
REM  停止前后端进程并 down MySQL/Redis 容器（数据卷保留）
REM ============================================================================
setlocal

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

"%BASH%" "%~dp0scripts\start-dev.sh" stop
pause
