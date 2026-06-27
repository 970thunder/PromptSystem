# AI Prompt & Skill Platform TODO

## 2026-05-17 Additions

- [x] Add cover image upload flow for Prompt publish
- [x] Support local object storage and Cloudflare R2-compatible storage
- [x] Persist prompts beyond in-memory mock data
- [x] Persist login/register users to MySQL
- [x] Implement real search page with backend filtering
- [x] Implement usable profile page with published prompts list

> 后续开发严格按本文件推进。每完成一个任务，必须同步勾选；新增需求先写入 TODO，再开发。

**技术栈（与仓库一致）**：前端 Vue 3 + Vite + Pinia + Naive UI；后端 **Go**（`net/http`、`internal/api` + `internal/store`、MySQL）。不以 `CLAUDE.md` 旧版 Spring/MyBatis 路径为准。

## 状态约定

- [ ] 未开始
- [x] 已完成
- [~] 进行中
- [!] 阻塞/需决策

## 当前开发策略

1. 先保证前后端能编译、能启动、能联调。
2. Phase 1 只做 Prompt 社区 MVP，不提前扩展 Skill、比赛、学院、企业版。
3. 所有页面先完成可用体验，再接真实后端接口。
4. 组件、类型、接口优先复用，避免一次性页面代码堆叠。
5. 后端统一使用 Go，并优先提供 Docker 化开发与部署链路。

---

# Sprint 0 - 开发准入与工程闭环

## 0.1 前端编译修复

- [x] 安装前端依赖并生成 package-lock.json
- [x] 修复 `src/frontend/src/utils/request.ts` 中 Naive UI 消息 API 使用错误
- [x] 移除前端未使用的 TypeScript 参数和导入
- [x] 确认 `npm run build` 通过
- [x] 确认 `npm run lint` 通过
- [x] 启动前端开发服务并确认首页可访问

## 0.2 Go 后端编译修复

- [x] 移除 Spring Boot 主路径骨架
- [x] 创建 Go 后端项目结构
- [x] 提供 `health` / `categories` / `prompts` MVP 接口
- [x] 确认 `go test ./...` 通过
- [x] 确认 Go 后端服务可启动

## 0.3 本地联调环境

- [x] 提供 Docker Compose MySQL 服务
- [x] 提供 Docker Compose Redis 服务
- [x] 提供 Docker Compose 前后端联调链路
- [x] 确认 Docker Compose 下 MySQL schema 自动初始化通过
- [x] 确认 Docker Compose 下 Redis 服务可访问
- [x] 确认 Vite `/api` 代理到 `http://localhost:8080`
- [x] 前端调用后端健康接口或 Prompt 接口成功

## 0.4 项目规范

- [x] 初始化 Git 仓库或确认已有版本管理方案
- [x] 添加根目录 `.gitignore`
- [x] 添加根目录 `README.md`，写明启动、构建、数据库初始化方式
- [x] 确认前后端环境变量命名规范
- [x] 确认提交前检查命令：前端 build/lint，后端 build/test

---

# Phase 1 - MVP 核心版本

## 1. 项目初始化

- [x] 创建前端项目（Vue3 + TypeScript + Vite）
- [x] 配置 TailwindCSS
- [x] 配置 Pinia
- [x] 配置 Vue Router
- [x] 配置 Axios 请求模块
- [x] 配置 ESLint + Prettier
- [x] 配置环境变量
- [ ] 配置 Git Hooks
- [x] 配置前端项目目录结构
- [x] 创建后端项目（Go）
- [x] 配置 Go 路由与中间件约定（`ServeMux` + `withCORS` + `withAuth`）
- [x] 配置 MySQL 环境变量
- [x] 配置 Redis 环境变量
- [x] 配置 JWT 基础能力
- [x] 创建初版数据库 schema

---

## 2. UI 设计系统

### 2.1 全局设计

- [ ] 设计全局配色方案
- [ ] 设计字体规范
- [ ] 设计圆角规范
- [ ] 设计阴影规范
- [ ] 设计动画规范
- [ ] 设计暗黑主题
- [ ] 设计响应式规则
- [ ] 整理 Tailwind theme tokens

### 2.2 基础组件

- [ ] Button 按钮
- [ ] Input 输入框
- [ ] Search 搜索框
- [ ] Modal 弹窗
- [ ] Drawer 抽屉
- [ ] Dropdown 下拉菜单
- [ ] Tabs 标签页
- [ ] Card 卡片
- [ ] Avatar 头像
- [ ] Tag 标签
- [ ] Pagination 分页
- [ ] Skeleton 骨架屏
- [ ] Loading 加载
- [ ] Empty 空状态

---

## 3. Prompt 数据与接口

### 3.1 后端 Prompt 基础接口（Go store + api）

- [x] Prompt struct 字段与前端类型对齐（`internal/store`）
- [x] Prompt store 接口与 MySQL/内存实现（`interfaces.go`、`mysql_prompts.go`、`memory_prompts.go`）
- [x] HTTP handlers（`internal/api/server.go`）
- [x] 获取 Prompt 列表接口
- [x] 获取 Prompt 详情接口
- [x] 创建 Prompt 接口（`POST /api/v1/prompts`）
- [x] 更新 Prompt 接口（`PUT /api/v1/prompts/:id`）
- [x] 删除/下架 Prompt 接口（`DELETE /api/v1/prompts/:id`）
- [x] 分类列表接口

### 3.2 前端 Prompt 数据层

- [x] 定义 Prompt / Category / User 类型
- [x] 封装 Prompt API
- [x] 封装 Category API
- [x] 建立 Prompt Pinia store
- [x] 首页支持接口数据与 mock 数据降级

---

## 4. 首页开发

### 4.1 顶部导航

- [x] Logo
- [x] 搜索框
- [x] 分类导航
- [x] 登录按钮
- [ ] 用户菜单
- [ ] 移动端导航

### 4.2 Banner 区域

- [x] 热门推荐 Banner
- [ ] 活动 Banner
- [ ] 动态轮播或精选横滑

### 4.3 Prompt 瀑布流

- [x] Prompt 卡片组件
- [ ] 瀑布流布局
- [ ] 卡片 hover 动画
- [x] 响应式适配
- [ ] 空状态
- [x] 加载状态
- [ ] 无限滚动或分页加载

### 4.4 分类系统

- [x] Prompt 分类
- [ ] Skill 分类入口占位
- [ ] 标签筛选
- [x] 热门标签

---

## 5. Prompt 详情页

### 5.1 页面结构

- [x] AI 结果展示区
- [ ] 图片轮播
- [x] Prompt 信息区
- [x] 参数展示
- [x] Prompt 正文
- [x] System Prompt
- [ ] Few-shot 展示
- [ ] Workflow 展示
- [x] 相关推荐

### 5.2 交互功能

- [x] 一键复制
- [x] 点赞
- [x] 收藏
- [ ] 分享
- [x] 评论列表
- [x] 评论发布
- [x] 回复评论

---

## 6. 用户系统

### 6.1 用户功能

- [x] 登录
- [x] 注册
- [x] JWT 鉴权
- [ ] 邮箱验证码开发环境方案
- [ ] 找回密码
- [x] 修改资料
- [x] GitHub 绑定提醒与个人中心入口
- [ ] 上传头像
- [x] 用户主页

### 6.2 用户数据

- [x] 收藏列表
- [x] 点赞记录
- [ ] 浏览历史
- [ ] 粉丝系统
- [ ] 关注系统

---

## 7. Prompt 发布系统

### 7.1 发布功能

- [x] 上传封面图
- [x] 输入 Prompt
- [x] 输入 System Prompt
- [x] 输入模型参数
- [x] 添加标签
- [x] 分类选择
- [ ] 发布草稿
- [x] 编辑 Prompt
- [x] 删除 Prompt

### 7.2 内容审核

- [ ] 敏感词检测
- [ ] AI 内容审核
- [ ] 举报系统

---

## 8. 搜索系统

### 8.1 MVP 搜索

- [x] Prompt 标题搜索
- [x] Prompt 描述搜索
- [x] 标签搜索
- [x] 分类筛选
- [x] 模型筛选
- [x] 搜索结果页

### 8.2 搜索优化

- [ ] 热门搜索
- [ ] 搜索建议
- [ ] 搜索历史
- [ ] ElasticSearch 接入

---

## 9. 评论互动系统

### 9.1 评论功能

- [x] 评论发布
- [x] 回复评论
- [x] 评论点赞
- [ ] 评论排序
- [ ] 评论举报

### 9.2 社区互动

- [x] Prompt 点赞
- [x] Prompt 收藏
- [ ] 关注创作者
- [ ] 消息通知
- [ ] 系统通知

---

## 10. 创作者中心

### 10.1 创作者后台

- [ ] 数据统计
- [ ] Prompt 管理
- [ ] Skill 管理占位
- [ ] 粉丝管理
- [ ] 收益统计占位
- [ ] 发布统计

### 10.2 数据分析

- [ ] 浏览量
- [ ] 点赞量
- [ ] 收藏量
- [ ] 复制次数
- [ ] 转化率
- [ ] 用户增长

---

# Phase 2 - 进阶功能

## 11. Skill 系统

- [ ] Skill 创建
- [ ] Skill 编辑
- [ ] Workflow 结构设计
- [ ] 输入变量系统
- [ ] 输出模板系统
- [ ] Workflow 节点展示
- [ ] Skill 详情页
- [ ] Workflow 可视化
- [ ] Skill 运行区
- [ ] Skill 案例展示

## 12. Prompt Playground

- [ ] 在线运行 Prompt
- [ ] 多模型切换
- [ ] Prompt 调试
- [ ] 输出结果对比
- [ ] Prompt 评分
- [ ] Token 统计

## 13. Prompt 变量系统

- [ ] 变量占位符
- [ ] 动态变量输入
- [ ] Prompt 模板系统
- [ ] Prompt 参数化

## 14. Workflow 系统

- [ ] 节点系统
- [ ] 连线系统
- [ ] Prompt 节点
- [ ] Tool 节点
- [ ] Memory 节点
- [ ] 条件节点
- [ ] 输出节点
- [ ] Workflow 执行
- [ ] 节点状态显示
- [ ] 执行日志
- [ ] 错误处理

## 15. AI 学院系统

- [ ] 课程分类
- [ ] 视频课程
- [ ] Markdown 文章
- [ ] 学习进度
- [ ] 收藏课程
- [ ] 评论课程
- [ ] 学习记录
- [ ] 考核系统
- [ ] AI 考试
- [ ] 证书系统

## 16. 比赛系统

- [ ] 比赛创建
- [ ] Prompt 投稿
- [ ] 投票系统
- [ ] 排行榜
- [ ] 奖励系统
- [ ] Prompt PK
- [ ] 模型 PK
- [ ] Workflow PK

---

# Phase 3 - 商业化与企业版

## 17. 企业版

- [ ] 企业组织
- [ ] 团队管理
- [ ] 权限管理
- [ ] 企业 Prompt 库
- [ ] 企业 Workflow
- [ ] 私有知识库
- [ ] 企业 Agent
- [ ] API 调用
- [ ] 使用统计

## 18. Prompt 交易市场

- [ ] Prompt 付费
- [ ] Skill 付费
- [ ] 订单系统
- [ ] 收益分成
- [ ] 创作者提现

## 19. API 开放平台

- [ ] API Key
- [ ] API 文档
- [ ] 调用统计
- [ ] API 限流
- [ ] API 权限

## 20. Agent 系统

- [ ] Agent 创建
- [ ] 多 Agent 协作
- [ ] Memory 系统
- [ ] Tool Calling
- [ ] RAG 能力

## 21. 后台管理系统

- [ ] 用户管理
- [ ] Prompt 审核
- [ ] Skill 审核
- [ ] 举报处理
- [ ] 内容管理
- [ ] Banner 管理
- [ ] 分类管理
- [ ] 比赛管理
- [ ] 数据统计

---

# 技术优化 TODO

## 性能优化

- [ ] 图片懒加载
- [ ] CDN 接入
- [ ] Redis 缓存
- [ ] SSR 优化
- [ ] SEO 优化
- [ ] 首屏优化

## 安全优化

- [ ] XSS 防护
- [ ] CSRF 防护
- [ ] 接口限流
- [x] JWT 安全（Bearer + 过期配置）
- [x] 文件上传安全（MIME/大小校验）
- [x] 密码 BCrypt 加密

## 运维部署

- [x] Docker 部署
- [ ] CI/CD
- [ ] Nginx 配置
- [ ] HTTPS 配置
- [ ] 日志监控
- [ ] Prometheus 监控

---

# 长期扩展 TODO

- [ ] AI 生成 Prompt
- [ ] AI 优化 Prompt
- [ ] Prompt 自动评分
- [ ] Prompt 推荐算法
- [ ] AI 创作者生态
- [ ] 多语言国际化
- [ ] AI SaaS 生态
- [ ] Agent Marketplace
- [ ] Workflow Marketplace
- [ ] AI App Store

---

# 最终开发要求

1. 前后端分离
2. 高组件化
3. 高扩展性
4. AI Native 设计
5. 社区化优先
6. 内容优先
7. 支持未来 Agent 扩展
8. 支持企业化扩展

最终目标：打造一个集 Prompt 社区、Skill 平台、AI 学院、Workflow 系统、Agent 生态于一体的 AI Native 平台。
