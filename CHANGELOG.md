# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Fixed
- 补全 HTTP 服务超时配置（Read/Write/IdleTimeout、MaxHeaderBytes），加固慢连接与慢请求防护
- 修复空房延迟清理与新连接注册的竞态：清理定时器原先先 `Stop()` 再从 Hub 摘除房间，窗口内新连接会拿到已停止的 `RoomHub`，`register` 在无人接收的 channel 上永久阻塞并泄漏 handler goroutine；现在先在写锁内确认并摘除 map 条目再停止，`register`/`unregister` 同时监听 `stop` 通道兜底
- 修复慢客户端被广播清理后在线人数永久偏高、空房不再触发延迟清理的问题（`broadcastToClients` 删除客户端后同步 `updateOnline` 并触发 `cleanupRoomLater`）
- 新增空房清理顺序与停止后注册不阻塞的回归测试

## [v2.2.0] - 2026-09-28
### Changed
- 前端界面与视觉效果深度美化：
  - 优化全局设计系统：微质感环境渐变背景、消息气泡立体阴影与微光描边、平滑缩放与过渡曲线
  - 升级登录认证界面：新增一键填入演示账号快捷卡片与四宫格架构微卡片（全双工推送、双 Token 续签、多房间隔离、历史漫游）
  - 升级侧边栏：强化用户头像光环动效、重绘新建房间面板交互、平滑筛选与活跃频道色条指引
  - 升级主聊天视窗：丰富未进房时的教学看板（三维特性网格）、细化连接状态指示（已连接/重连中/已断开）、日期分组双向细线与空消息破冰芯片
- 精简并优化 README.md：去除大量 emoji，表述更加干练务实，以清晰列表呈现功能特性、技术栈及运行指引，修复截图为相对路径
- 前端整体切换为浅色精美主题：雾蓝留白、柔和阴影与圆角卡片，重构 `index.css`、`AuthScreen`、`Sidebar`、`ChatRoom`、`MessageList`、`MessageInput`、`ErrorBoundary`、`Toast`，提升长时间阅读与演示体验
- README 界面预览更新为浅色主题截图（`docs/public/screenshots/chatroom.png` 1440×900，平均亮度 247），深色预览已替换
- 前端界面显示效果优化：消息气泡尾巴指示、自己消息显示头像、头像 hover 放大效果、滚动到底部快捷按钮、帮助面板点击外部关闭、Toast 退出动画、活跃房间左侧色条、头像渐变色扩展至 8 种

## [v2.1.0] - 2026-08-07

全面改进：修复前后端不一致、清理死代码、简化复杂逻辑、补充测试。

### Added
- README「界面预览」：新增聊天室实时界面截图（`docs/public/screenshots/`）
- `processor_test.go`：5 个测试覆盖空消息、正常消息、XSS 过滤、超长内容、持久化失败
- `middleware_test.go`：9 个测试覆盖限流器（允许/拒绝/隔离）、CORS（dev/prod/no-origin）、Bearer 提取、IP 解析
- 代码注释：MessageProcessor 接口设计理由、App.tsx ref 循环依赖模式说明、AuthRuntime 存在原因

### Fixed
- `internal/ws/conn.go`：WebSocket 升级时使用 `Upgrade(..., nil)`，去掉重复写入的 `Sec-WebSocket-Protocol` 响应头（浏览器会因重复响应头解析异常而中断握手，close 1006）
- `frontend/src/App.tsx`：登录后加载房间列表的 effect 不再依赖整个 `chat` 对象，改为 ref 稳定引用，避免每次渲染重复触发 `reloadRooms`（曾打满限速导致 429 toast）
- 前端密码校验从 min 4 改为 min 8，与后端 binding 一致
- data-flow.md 输入状态流图：移除虚构的 is_typing:false，补充接收方超时机制说明
- AuthScreen 营销化文案替换为事实描述

### Removed
- `ErrTokenExpired` 死代码（service/errors.go，无任何引用）
- `format_test.go` 中 .kiro 目录跳过和 Requirements 3.3 注释（来自已删除的工具和 spec 系统）
- `index.css` 中 .unread-badge 和 .msg-actions 死 CSS
- `ChatRoom.tsx` 中空占位 `<div />`
- CI 中多余的 PostgreSQL service container（测试使用 SQLite 内存数据库）

## [v2.0.0] - 2026-08-07

最终教学定稿版。项目定位为全栈开发教学材料，不再迭代维护。

### Removed
- 删除 `.claude/` 目录（settings.json + skills/verify/SKILL.md），AGENTS.md 为唯一规范指令文件
- 删除 `.github/workflows/release.yml`（多平台二进制、Docker GHCR push、checksums、GitHub Release），教学项目不需要重型发布流水线
- 从 CI 移除 Codecov 覆盖率上传步骤
- 删除 `docs/.vitepress/theme/style.css` 中的 ADR 相关死代码（约 60 行）
- 删除分布式消息同步：`internal/ws/realtime.go`、`WSSession` 模型、`PodID` 配置、`internal/repository` 包
- 删除 Prometheus 监控：`internal/metrics` 包、`/metrics` 端点、`deploy/prometheus`、docker-compose 中的 prometheus/grafana 服务
- 删除 `internal/mw` 包（合并入 `internal/server`）、`internal/log` 包（合并入 `internal/config`）
- 删除 `internal/auth/ticket.go`（合并入 `internal/auth/auth.go`）
- 删除 docs 中的 whitepaper、deep-dives、decisions、deployment/learning-path/development-guide 等过度设计或与代码不符的文档

### Changed
- 重新定位为"全栈开发教学材料"，README 增加推荐源码阅读路线
- 同步 docs 首页、FAQ 定位语言
- 修复 `docs/tutorials/testing.md`：删除已不存在的 Prometheus 监控实验、修正密码长度（4-128 → 8-128）、将静态回退实验改为构建产物部署实验
- 修复 `docs/tutorials/local-dev.md`：删除不存在的 `web/` 静态回退运行模式
- 修复 `scripts/dev.sh`：Go 版本提示从 1.21+ 更正为 1.24+
- 精简 `docs/.vitepress/theme/style.css`：移除 ADR 死代码和无用的 CSS 变量
- 简化 `internal/auth`：移除 `GormAdapter` 与接口适配层，DB 操作直接接收 `*gorm.DB`
- 简化 `internal/service`：移除 repository 版双套实现，单一 service 直接依赖 `*gorm.DB`
- 简化 `internal/ws/processor` 与 `conn`：移除 hub 引用、sessionID、trackHeartbeat、publish
- `config.InitLog` 取代独立的 `clog.Init`
- 重写 docs 站点首页、系统架构、数据流、数据模型、FAQ、本地开发文档，去除虚构的基准数据与 K8s/多实例描述

## [v1.0.0] - 2026-05-01

### Changed
- Enhanced password validation: minimum length increased from 4 to 8 characters
- Fixed token refresh race condition in frontend API client
- Updated CODE_OF_CONDUCT with proper contact information

### Removed
- Cleaned up redundant files: build artifacts, node_modules, plugin data
- Removed outdated `docs/monitoring/` (use `docs/zh/monitoring/` instead)

### Documentation
- Synchronized VERSION and CHANGELOG to v1.0.0
- Project now in archive-ready state

## [v0.3.0] - 2026-04-16

### Added
- Complete bilingual documentation (English/Chinese)
- VitePress documentation site with internationalization support
- English documentation: Getting Started, API, Architecture, Design, FAQ, Manual Testing, Monitoring
- Language switcher between English and Chinese versions

### Changed
- Restructured docs directory: moved Chinese docs to `docs/zh/`, added `docs/en/`
- Standardized all changelog files with professional format
- Updated VitePress config for multi-language support
- Enhanced README with clearer structure and badges

### Documentation
- Professionalized documentation formatting across all docs
- Added comprehensive API documentation in both languages
- Created monitoring guide for Prometheus and Grafana
- Added architecture diagrams and data flow explanations

## [v0.2.0] - 2026-03-08

### Added
- Open source standard files: LICENSE, CONTRIBUTING, CODE_OF_CONDUCT, SECURITY
- CI/CD pipelines with GitHub Actions
- VitePress documentation site
- Docker multi-stage builds
- Kubernetes deployment manifests
- Health check endpoints (`/health`, `/healthz`, `/ready`, `/version`)
- Prometheus metrics integration
- Rate limiting middleware

### Security
- JWT secret validation for production environments
- CORS origin whitelist validation
- WebSocket origin validation matching HTTP CORS
- Input sanitization and validation

### Changed
- Comprehensive backend refactoring with service layer
- Frontend migrated to React 19 + TypeScript + Vite 7
- WebSocket authentication using one-time tickets
- Unified error handling across the application

### Fixed
- Rate limiter goroutine leak
- WebSocket message deduplication
- Connection cleanup on disconnect

## [v0.1.0] - 2025-01-08

### Added
- User registration and login with JWT authentication
- Chat room creation and management
- WebSocket real-time messaging
- PostgreSQL message persistence
- Vanilla JavaScript frontend
- Basic Docker support
