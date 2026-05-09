# 龙屿 (Dragon Islet) 🐉

一个打破传统的个人博客，以“中央自由聊天区”为核心，接入 DeepSeek AI 领主，打造酷炫的西方龙风格互动社区。

## 🚀 技术栈

### 后端 (Go)
- **框架**: Gin
- **ORM**: GORM (MySQL)
- **缓存**: Redis (用于发言限流)
- **AI**: DeepSeek API (内容安全检查 & 自动回复)
- **架构**: 严格的工程化分层 (`Handler` -> `Service` -> `Repository`)

### 前端 (Vue 3)
- **构建工具**: Vite
- **样式**: Vanilla CSS + Glassmorphism (龙鳞纹理, 熔岩动效)
- **动画**: AI 审核扫描动画, 消息渐入动画

## 📂 项目结构

- `cmd/server`: 项目入口。
- `internal/handler`: 接口层，负责请求解析和响应。
- `internal/service`: 业务逻辑层，核心 AI 逻辑和权限校验在此实现。
- `internal/repository`: 数据访问层。
- `internal/initialize`: 各种组件的初始化逻辑 (DB, Redis, Config)。
- `pkg/deepseek`: 封装的 DeepSeek SDK。

## 🛠️ 快速开始

### 1. 配置环境
在 `configs/config.yaml` 中配置你的 MySQL, Redis 和 DeepSeek API Key。或者使用环境变量：
- `DB_DSN`: `user:pass@tcp(host:port)/dbname`
- `REDIS_ADDR`: `host:port`
- `DEEPSEEK_API_KEY`: `your_key`

### 2. 运行后端
```bash
cd dragon-islet
go mod tidy
go run cmd/server/main.go
```

### 3. 运行前端
```bash
cd dragon-islet-web
npm install
npm run dev
```

## ✨ 亮点功能
1. **龙语审核**: 每一条发出的誓言都会经过 DeepSeek 的安全检查，前端有酷炫的“扫描”动画。
2. **AI 领主**: 龙屿之主（DeepSeek）会随机点评游侠们的发言，并明确标注回复对象。
3. **龙族律法**: 每 5 分钟只能发言一次（Redis 原子操作实现）。
