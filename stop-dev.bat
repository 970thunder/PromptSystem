@echo off
REM ============================================================================
REM  PromptOS 一键停止（Windows 双击入口）
REM  停止前后端进程并 down MySQL/Redis 容器（数据卷保留）
REM ============================================================================
setlocal

set "ROOT_DIR=%~dp0"
set "SH=%ROOT_DIR%scripts\start-dev.sh"
set "BASH="

REM 同 start-dev.bat：%ProgramFiles(x86)% 不能放进括号块，否则闪退。
if exist "%ProgramFiles%\Git\bin\bash.exe" set "BASH=%ProgramFiles%\Git\bin\bash.exe"
if not defined BASH if exist "%ProgramFiles(x86)%\Git\bin\bash.exe" set "BASH=%ProgramFiles(x86)%\Git\bin\bash.exe"
if not defined BASH if exist "%LocalAppData%\Programs\Git\bin\bash.exe" set "BASH=%LocalAppData%\Programs\Git\bin\bash.exe"
if not defined BASH if exist "%ProgramFiles%\Git\usr\bin\bash.exe" set "BASH=%ProgramFiles%\Git\usr\bin\bash.exe"
if not defined BASH if exist "%USERPROFILE%\.workbuddy\binaries\PortableGit\current\bin\bash.exe" set "BASH=%USERPROFILE%\.workbuddy\binaries\PortableGit\current\bin\bash.exe"
if not defined BASH if exist "%USERPROFILE%\.workbuddy\binaries\PortableGit\versions\1.2.0\bin\bash.exe" set "BASH=%USERPROFILE%\.workbuddy\binaries\PortableGit\versions\1.2.0\bin\bash.exe"
if not defined BASH (
  where bash >nul 2>nul && set "BASH=bash"
)

if not defined BASH (
  echo [ERROR] 未找到 Git Bash，请安装 Git for Windows 后重试。
  echo.
  cmd /k
  exit /b 1
)

if not exist "%SH%" (
  echo [ERROR] 未找到启动脚本：%SH%
  echo.
  cmd /k
  exit /b 1
)

echo [INFO ] Git Bash : %BASH%
echo.

"%BASH%" "%SH%" stop
set "RC=%ERRORLEVEL%"

echo.
if not "%RC%"=="0" (
  echo [ERROR] 停止过程返回退出码 %RC%。
) else (
  echo [ OK  ] 已停止全部服务。
)
echo.
exit /b %RC%
