# PromptOS TODO

## 已完成

- [x] 前端依赖恢复，`npm run build` 和 `npm test` 通过
- [x] 亮色主题迁移到蓝白 token，主题切换使用 Sun/Moon icon button
- [x] 首页、搜索、个人中心移除运行时 mock 降级
- [x] 页面 API 失败态、重试态和关键操作错误反馈
- [x] 认证、上传、点赞、收藏、关注、评论交互的加载状态
- [x] 后端 `go vet ./...` 和 `go test ./...` 通过

## 上线前必须完成

- [ ] 配置真实邮件/短信验证码服务，关闭开发验证码回填
- [ ] 配置生产 JWT、MySQL、Redis、对象存储、CORS 和 GitHub OAuth
- [ ] 完成 HTTPS、隐私政策、用户协议、备份恢复和告警配置
- [ ] 将公共列表 API 和前端主包纳入性能监控，优化大体积 Naive UI chunk
- [ ] 补充 Playwright 桌面/移动端截图验收并接入 CI

## PromptOS 1.0 后续

- [ ] 图像、Skill、项目、工作流、智能体独立频道
- [ ] 统一内容模型、版本和媒体资产服务
- [ ] 管理端内容审核、举报工单和审计日志
