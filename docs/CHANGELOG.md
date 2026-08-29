# Changelog — PromptOS

所有对外可见的变更记录在本文件。版本号：`v主.次.修订`（主=大改/不兼容，次=新功能，修订=修复）。

## [Unreleased]

### Security（计划中）
- compose 的 `MYSQL_ROOT_PASSWORD` 与 `JWT_SECRET` 改为环境变量注入，消除硬编码 root 密码

### Added
- 首页标题飘带使用已有提示词标题和封面图，支持多行、不同速度和 3D 视觉效果
- 增加首页、提示词卡片与详情页冒烟测试

### Changed
- 开发启动入口改为前台显示前后端实时日志，关闭窗口或按 Ctrl+C 时自动清理子进程和容器
- 启动脚本支持复用已有 PromptOS MySQL/Redis 容器，并忽略本地 PID 文件

### Fixed
- 修复提示词卡片默认链接为空字符串导致无法进入详情页的问题
- 修复提示词详情页在作者、评论用户数据缺失时渲染异常的问题

## [20260829-a9ba2cf] - 2026-08-29

### Deployment
- 首次部署至 `promptsystem.isoumao.top`。
- 使用独立 MySQL、Redis、上传卷和 `promptsystem` Compose 项目；生产入口接入 HTTPS nginx 反代。

## [v0.0.1] - YYYY-MM-DD

- MVP 初始版本（package.json 0.0.1；历史任务编号见 `docs\后端迭代任务清单.md`）
