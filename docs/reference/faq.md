# 常见问题

## 项目定位

### 这是一个什么项目？

一个**定稿的全栈开发教学项目**：代码不再迭代，定位为学习参考。它用 Go + React + PostgreSQL + WebSocket 实现了一个实时聊天室，覆盖认证、房间、消息、实时通信的完整链路。

项目包含的工程化实践本身就是学习材料：

- Go + 前端的双端测试
- CI 流水线（GitHub Actions）
- Docker 多阶段构建
- VitePress 文档站

这是一个**单实例**教学项目，不包含多副本部署、分布式同步或监控体系。

### 我能学到什么？

**后端**：
- Gin 路由与中间件的组织方式
- JWT + Refresh Token 鉴权流程
- GORM 与 PostgreSQL 的配合
- WebSocket 房间广播实现

**前端**：
- React Hooks 状态管理
- WebSocket 连接与重连策略
- Token 自动刷新机制
- TypeScript 类型设计

**工程化**：
- 测试编写与 CI 配置
- Docker 多阶段构建

### 推荐的源码阅读顺序？

见 [快速开始](/tutorials/local-dev) 页面的"推荐阅读路线"，或直接查看 [README](https://github.com/holtwood-projects/chatroom) 中的阅读路线表。

---

## 技术选型

### 为什么同时有 REST API 和 WebSocket？

它们解决的问题不同：

| 场景 | 技术选择 |
|------|----------|
| 注册、登录 | REST API |
| 查房间、查历史 | REST API |
| 实时消息 | WebSocket |
| 在线状态、输入提示 | WebSocket |

REST 适合"请求-响应"模式，WebSocket 适合"实时推送"。

### 为什么用 PostgreSQL？

PostgreSQL 同时承担关系数据存储和 WebSocket 房间状态，单一数据源减少技术栈复杂度，适合教学。

### 为什么用 Tailwind CSS v4？

- 无需配置文件（`tailwind.config.js`）
- 构建更快（Rust 编写的引擎）
- 原子化 CSS，开发效率高
- 适合教学，减少 CSS 抽象

---

## 前端问题

### 前端测试用的是什么？

- **测试框架**：Vitest
- **测试类型**：单元测试（API、Socket、Storage）

运行测试：
```bash
npm --prefix frontend run test
```

### Token 存储在哪里？

**当前实现**：localStorage（教学简化）

**生产建议**：
- Access Token：内存 + httpOnly cookie
- Refresh Token：httpOnly cookie + Secure + SameSite

localStorage 有 XSS 风险，但教学项目优先考虑简单易懂。

---

## 后端问题

### 配置是如何加载的？

**关键点**：后端直接读取进程环境变量，**不会自动加载 `.env` 文件**。

```bash
# .env.example 是配置模板
# 运行 go run ./cmd/server 不会读取 .env

# 方式 1：直接设置环境变量
export JWT_SECRET=your-secret
go run ./cmd/server

# 方式 2：Docker Compose
# docker-compose.yml 中通过 environment 配置

# 方式 3：手动 source
set -a && source .env && set +a
go run ./cmd/server
```

### WebSocket 如何认证？

WebSocket 使用**一次性 Ticket** 认证，而非直接使用 Access Token：

```
1. 前端调用 POST /api/v1/ws/tickets 获取 ticket
2. 服务端生成并存储 ticket（有效期 60 秒）
3. 前端在 WebSocket Subprotocol 中携带 ticket
4. 服务端验证并消费 ticket
5. 建立 WebSocket 连接
```

**设计理由**：
- 避免在 URL 中暴露 Access Token
- Ticket 一次性消费，防止重放攻击
- 与特定房间绑定，防止跨房间滥用

---

## 部署问题

### `ALLOWED_ORIGINS` 是做什么的？

非 `dev` 环境下的来源校验：

| 场景 | 校验方式 |
|------|----------|
| HTTP CORS | 检查 Origin 头是否在白名单 |
| WebSocket 升级 | 检查 Origin 头是否在白名单 |

未命中白名单时，仅允许严格同源请求。

**配置示例**：
```dotenv
ALLOWED_ORIGINS=https://chat.example.com,https://app.example.com:8443
```

### 如何在 Docker 中运行？

```bash
# 完整环境（数据库 + 应用）
docker compose up -d

# 仅数据库，后端本地运行
docker compose up -d postgres
go run ./cmd/server
```

### 文档站如何发布？

仓库已配置 GitHub Actions 工作流：

1. 推送到 `master` 分支
2. 自动构建 VitePress 文档站
3. 部署到 GitHub Pages

---

## 测试问题

### Go 测试需要 PostgreSQL 吗？

不需要。测试使用 SQLite 内存数据库，不依赖外部服务。

```bash
# 直接运行测试
go test -race ./...
```

### 如何运行所有测试？

```bash
# Go 测试
make test
# 或
go test -race -cover ./...

# 前端测试
npm --prefix frontend run test

# 全部测试
make test && npm --prefix frontend run test
```

---

## 其他问题

### 版本号规则？

遵循 [语义化版本](https://semver.org/lang/zh-CN/)。版本历史见 [`CHANGELOG.md`](https://github.com/holtwood-projects/chatroom/blob/main/CHANGELOG.md)。
