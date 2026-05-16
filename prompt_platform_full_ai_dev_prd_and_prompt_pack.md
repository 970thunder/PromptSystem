# AI Prompt & Skill 平台完整开发文档（AI任务提示词版）

# 一、项目定位

项目名称（暂定）：

- PromptVerse
- PromptHub
- SkillFlow
- PromptLab
- PromptOS

项目定位：

打造一个集 Prompt 分享、Skill 工作流、AI 教学、Prompt 竞赛、创作者生态、企业 Prompt 管理于一体的 AI Native 内容社区平台。

平台目标：

1. 成为 Prompt 工程社区
2. 成为 AI 创作者平台
3. 成为 Prompt 工程学习平台
4. 成为 Skill 工作流平台
5. 后续扩展 Agent 与 Workflow 生态

---

# 二、产品核心方向

平台整体分为：

```text
内容社区层
├── Prompt广场
├── Skill广场
├── AI案例库
├── 创作者社区
├── 排行榜
└── 比赛活动

学习成长层
├── Prompt学院
├── Skill教学
├── Agent教学
├── 实战案例
└── 考核系统

工具运行层
├── Prompt运行
├── Workflow运行
├── Prompt Playground
├── Prompt对比测试
└── AI优化系统

商业化层
├── Prompt交易
├── Skill市场
├── 企业Prompt库
├── API开放平台
└── 企业协作空间
```

---

# 三、整体风格定义

视觉方向：

- 深色科技风
- AI Native 风格
- 强视觉封面
- 卡片流布局
- 沉浸式内容浏览
- 类 Pinterest / Civitai / Liblib 风格

关键词：

```text
Dark
Glassmorphism
AI Community
Immersive
Modern
Minimal
Creator Economy
```

设计要求：

1. 首页社区化
2. 内容优先
3. 图片优先
4. Prompt 结构清晰
5. 卡片视觉统一
6. 动效轻量化
7. 高级感与专业感并存

---

# 四、整体信息架构

```text
首页
├── 推荐
├── 热门
├── 最新
├── 关注
└── 比赛活动

Prompt
├── 图片生成
├── 文案写作
├── 编程开发
├── 视频生成
├── Agent Prompt
└── 工作流Prompt

Skill
├── 内容创作
├── 电商运营
├── 数据分析
├── AI客服
├── 编程自动化
└── 多Agent协作

学院
├── Prompt基础
├── Prompt工程
├── Skill开发
├── Workflow设计
├── Agent开发
└── 企业AI实践

创作者
├── 创作者主页
├── 发布管理
├── 数据分析
├── 收益系统
└── 粉丝系统

比赛
├── Prompt挑战赛
├── Workflow比赛
├── AI海报赛
├── Agent设计赛
└── 排行榜

企业版
├── 企业Prompt库
├── 团队空间
├── 企业Workflow
├── 权限系统
└── API平台
```

---

# 五、首页设计方案

# 首页布局结构

```text
┌──────────────────────────────────┐
│ LOGO 搜索框 分类 导航 登录注册   │
├──────────────────────────────────┤
│ Banner                           │
│ 今日热门Prompt / 活动            │
├──────────────────────────────────┤
│ 分类Tabs                         │
│ 全部 绘画 编程 文案 视频 Agent   │
├──────────────────────────────────┤
│ Prompt瀑布流                     │
│                                  │
│ ┌──────┐ ┌──────┐ ┌──────┐      │
│ │封面图│ │封面图│ │封面图│      │
│ │标题  │ │标题  │ │标题  │      │
│ │模型  │ │模型  │ │模型  │      │
│ └──────┘ └──────┘ └──────┘      │
│                                  │
├──────────────────────────────────┤
│ 热门Skill                        │
├──────────────────────────────────┤
│ Prompt挑战赛                     │
├──────────────────────────────────┤
│ 创作者推荐                       │
└──────────────────────────────────┘
```

---

# 六、Prompt详情页设计

# 页面布局

```text
┌──────────────────────────────────┐
│ 左侧：结果展示                   │
│                                  │
│ 大图                             │
│ 视频                             │
│ 对话结果                         │
│                                  │
├──────────────────────────────────┤
│ 右侧：Prompt信息                 │
│                                  │
│ 标题                             │
│ 作者                             │
│ 分类                             │
│ 标签                             │
│ 模型                             │
│ 参数                             │
│ 点赞 收藏 分享                   │
│ 复制按钮                         │
└──────────────────────────────────┘
```

# 下方模块

```text
Prompt正文
System Prompt
Few-shot
Negative Prompt
Workflow结构
评论区
相关推荐
```

---

# 七、Skill页面设计

# Skill定位

Skill 不等于 Prompt。

Skill 是：

```text
Prompt + Structure + Workflow + Constraints
```

# 页面结构

```text
Skill名称
Skill简介
适用场景

输入参数
├── 行业
├── 风格
├── 目标用户
└── 输出格式

Workflow结构图

案例展示

在线运行

版本更新记录
```

---

# 八、Prompt学院模块

# 课程体系

## Prompt基础

- 什么是Prompt
- Prompt结构
- Role Prompt
- Few-shot
- CoT
- ReAct

## Prompt工程

- Prompt模块化
- Prompt复用
- Prompt变量系统
- Prompt调优
- Token优化

## Skill开发

- Skill设计
- Workflow设计
- 多Prompt协作
- AI任务拆解

## Agent开发

- Tool Calling
- Memory
- RAG
- Planning
- Multi-Agent

---

# 九、比赛系统设计

# Prompt挑战赛

用户提交：

- Prompt
- 结果图
- 模型
- 参数

支持：

- 投票
- 点赞
- 收藏
- 排名
- 评分

# 排行榜

```text
日榜
周榜
月榜
年度榜
```

# 奖励系统

- 勋章
- 创作者认证
- Skill认证
- 平台积分
- 现金奖励

---

# 十、用户系统

# 用户等级

```text
Lv1 Prompt新手
Lv2 Prompt设计师
Lv3 Skill架构师
Lv4 Workflow工程师
Lv5 AI创作者
```

# 用户功能

```text
个人主页
作品发布
收藏
关注
评论
点赞
数据统计
```

---

# 十一、创作者中心

# 功能模块

```text
作品管理
Skill管理
数据分析
收益统计
粉丝管理
版本管理
```

# 数据统计

```text
浏览量
点赞
收藏
复制次数
运行次数
转化率
```

---

# 十二、Prompt数据结构设计

# Prompt对象结构

```json
{
  "title": "爆款小红书文案Prompt",
  "description": "用于生成高互动小红书内容",
  "category": "文案",
  "tags": ["小红书", "运营", "营销"],
  "cover": "image_url",
  "model": "GPT-4.1",
  "prompt": "完整Prompt内容",
  "systemPrompt": "系统Prompt",
  "fewShot": [],
  "workflow": [],
  "params": {
    "temperature": 0.7,
    "top_p": 0.9
  },
  "stats": {
    "views": 1000,
    "likes": 300,
    "favorites": 120
  }
}
```

---

# 十三、Skill DSL设计

```json
{
  "skillName": "AI短视频脚本生成",
  "description": "生成短视频结构化脚本",
  "inputs": [
    {
      "name": "topic",
      "type": "string"
    }
  ],
  "workflow": [
    {
      "type": "prompt",
      "name": "角色定义"
    },
    {
      "type": "prompt",
      "name": "结构生成"
    },
    {
      "type": "prompt",
      "name": "情绪增强"
    }
  ]
}
```

---

# 十四、推荐技术架构

# 前端

```text
Vue3
TypeScript
Vite
TailwindCSS
Pinia
Vue Router
```

# UI框架

推荐：

```text
Naive UI
shadcn-vue
Element Plus（后台）
```

# 后端

```text
Golang
net/http or Gin
MySQL
Redis
ElasticSearch
MinIO
RabbitMQ
```

# 容器化

```text
Docker
Docker Compose
Nginx
```

# AI层

```text
OpenAI
Claude
Gemini
OpenRouter
Dify
LangChain
OpenClaw
```

---

# 十五、数据库核心表

# 用户表 users

```text
id
username
avatar
email
password
bio
level
experience
status
created_at
```

# Prompt表 prompts

```text
id
title
description
cover
content
system_prompt
model
params
category_id
user_id
views
likes
favorites
status
created_at
```

# Skill表 skills

```text
id
name
description
workflow
schema
cover
user_id
views
likes
```

# 评论表 comments

```text
id
target_type
target_id
user_id
content
created_at
```

# 收藏表 favorites

```text
id
user_id
target_type
target_id
created_at
```

---

# 十六、AI开发提示词（核心）

# 首页开发提示词

```text
你现在是一名资深前端架构师。

请使用 Vue3 + TypeScript + TailwindCSS 开发一个 AI Prompt 社区首页。

要求：

1. 深色科技风
2. 类似 Civitai / Liblib 的 AI 社区风格
3. 使用瀑布流卡片布局
4. 卡片展示封面图、标题、模型、点赞数
5. 顶部包含搜索栏、分类导航、登录按钮
6. 页面具有高级感与沉浸感
7. 卡片 hover 具有轻微动画
8. 支持响应式布局
9. 使用组件化结构
10. 代码需符合生产级规范

请输出：

- 页面结构
- 完整Vue代码
- Tailwind样式
- 组件拆分建议
```

---

# Prompt详情页开发提示词

```text
请使用 Vue3 + TailwindCSS 开发一个 Prompt 详情页。

页面布局：

左侧：
- AI生成结果展示
- 图片轮播
- 视频展示

右侧：
- Prompt标题
- 作者信息
- 模型信息
- 参数配置
- Prompt正文
- 复制按钮
- 收藏点赞按钮

下方：
- 评论区
- 推荐Prompt

风格要求：
- 深色科技风
- 高级感
- AI社区风格
- 类似 Midjourney / Civitai
```

---

# Skill工作流页面提示词

```text
请设计一个 AI Skill 工作流页面。

要求：

1. 展示 Workflow 节点
2. 展示 Prompt 链路
3. 支持节点连接
4. 展示输入输出
5. 支持运行结果展示
6. 类似 Dify / Langflow 风格
7. 页面具有工程化与专业感

技术栈：
Vue3
TailwindCSS
TypeScript
```

---

# 后端开发提示词

```text
请使用 Golang 开发 AI Prompt 平台后端。

项目结构要求：

cmd
internal/handler
internal/service
internal/repository
internal/model
internal/config
internal/platform

功能包括：

1. 用户登录注册
2. Prompt发布
3. Prompt搜索
4. 评论点赞收藏
5. Skill管理
6. 文件上传
7. 权限控制
8. Redis缓存
9. JWT鉴权
10. ElasticSearch搜索

数据库：MySQL
缓存：Redis
部署：Docker Compose
```

---

# 数据库设计提示词

```text
请为 AI Prompt 平台设计完整数据库。

要求：

1. 满足高扩展性
2. 满足社区化需求
3. 满足Prompt工程需求
4. 满足Skill工作流需求
5. 满足比赛系统需求
6. 满足企业版需求

输出：

- 表结构
- 字段说明
- 索引设计
- ER图结构
```

---

# UI设计提示词

```text
请设计一个 AI Prompt 社区平台 UI。

风格要求：

- 深色科技风
- AI Native
- 高级感
- 沉浸式
- 卡片流
- 大图封面
- 类似 Civitai + Apple + Midjourney

需要设计：

1. 首页
2. Prompt详情页
3. Skill页
4. 创作者主页
5. 比赛页
6. 后台管理系统

要求输出：

- 页面布局
- 色彩方案
- 字体方案
- 卡片设计
- 按钮设计
- 动效建议
```

---

# 十七、推荐配色方案

# 主色

```text
#0B0F19
#111827
#7C3AED
#9333EA
#06B6D4
```

# 辅助色

```text
#FFFFFF
#A1A1AA
#27272A
#18181B
```

---

# 十八、设计规范

# 圆角

```text
卡片：20px
按钮：14px
输入框：16px
```

# 阴影

```text
柔和发光阴影
玻璃拟态阴影
```

# 动效

```text
hover轻微上浮
渐变边框
卡片发光
```

---

# 十九、未来扩展方向

# 后续扩展

```text
Agent市场
Workflow市场
AI API平台
AI SaaS工具
AI自动优化Prompt
AI评分系统
AI生成Prompt
企业私有空间
```

---

# 二十、MVP开发优先级

# 第一阶段

```text
首页
Prompt详情页
登录注册
发布系统
评论点赞收藏
搜索
分类
```

# 第二阶段

```text
Skill系统
教学系统
Prompt变量系统
Prompt Playground
```

# 第三阶段

```text
比赛系统
企业版
Workflow
Agent系统
API开放平台
```

---
