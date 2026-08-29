@echo off
REM ============================================================================
REM  PromptOS 一键启动（Windows 双击入口）
REM  固定端口：前端 28301 / 后端 28302 / MySQL 28303 / Redis 28304
REM  调用 Git Bash 执行 scripts/start-dev.sh
REM ============================================================================
setlocal

set "ROOT_DIR=%~dp0"
set "SH=%ROOT_DIR%scripts\start-dev.sh"
set "BASH="

REM 注意：%ProgramFiles(x86)% 绝对不能出现在 for ( ... ) 或 if ( ... ) 代码块中。
REM      变量名里的 ")" 会被 cmd 当成代码块结束符，造成语法错误、双击直接闪退。
REM      因此这里一律使用单行 if exist，不用括号块。
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
echo [INFO ] 脚本     : %SH%
echo [INFO ] 端口     : 前端 28301 / 后端 28302 / MySQL 28303 / Redis 28304
echo.

"%BASH%" "%SH%" %*
set "RC=%ERRORLEVEL%"

echo.
if not "%RC%"=="0" (
  echo [ERROR] 启动失败，退出码 %RC%。请查看上方输出，或打开日志：
  echo         %ROOT_DIR%logs\backend.log
  echo         %ROOT_DIR%logs\frontend.log
  echo.
  echo [INFO ] 窗口已保持打开，可直接查看上方日志；关闭窗口不会留下服务进程。
  cmd /k
) else (
  echo [ OK  ] 开发服务已结束，所有子进程已清理。
)
echo.
exit /b %RC%
