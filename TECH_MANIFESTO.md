# 🐉 龙屿·铸龙图谱 (Technical Architecture)

## 核心架构 (Core Architecture)
- **后端 (Go)**: 基于 Gin 框架的 RESTful API，使用 GORM (SQLite/MySQL) 进行持久化。
- **前端 (Vue 3)**: Vite 构建，组合式 API (Setup)，响应式深色幻想风格 UI。
- **实时通信**: 原生 WebSocket 实现全局消息广播。
- **状态维护**: Redis 用于频率限制与令牌桶控流。

## 数据模型 (Data Models)
- `User`: 支持称号、灵力值(Experience)、游侠宣言(Motto)等个性化字段。
- `Message`: 支持 AI 兴趣识别(AIInterest)、撤回状态、秘宝强制回复。
- `Archive`: 分类存储（0: 岛屿行纪, 1: 铸龙图谱）。
- `Feedback`: 匿名鳞笺系统，支持主的回响。

## 技术特性 (Tech Features)
- **AI 交互**: 深度集成 DeepSeek API，实现龙主异步点评与自动史诗生成。
- **视觉风格**: 纯 CSS 实现的黑曜石毛玻璃感，GPU 硬件加速滚动。
- **权限控制**: 基于 JWT 的用户认证。
- **自省系统**: AI 自动扫描代码变更并同步技术进展（开发中）。

## 当前版本 (Current Status)
- **当前阶段**: v1.1.0 游侠身份系统与双路史诗架构已完成。
