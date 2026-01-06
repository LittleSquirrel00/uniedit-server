# Tasks: Add Swagger Integration

## Phase 1: 基础设施

### 1.1 安装依赖

- [ ] **1.1.1** 安装 swag CLI 工具
  ```bash
  go install github.com/swaggo/swag/cmd/swag@latest
  ```
  - **Verify**: `swag --version` 输出版本号

- [ ] **1.1.2** 添加 Go 依赖
  ```bash
  go get -u github.com/swaggo/swag
  go get -u github.com/swaggo/gin-swagger
  go get -u github.com/swaggo/files
  ```
  - **Verify**: go.mod 包含依赖

### 1.2 主文档配置

- [ ] **1.2.1** 创建 `cmd/server/docs.go` 添加主文档注解
  - @title UniEdit Server API
  - @version 1.0
  - @description UniEdit 视频编辑器后端 API
  - @host localhost:8080
  - @BasePath /api/v1
  - @securityDefinitions.apikey BearerAuth
  - **Verify**: 编译通过

- [ ] **1.2.2** 更新 `internal/app/app.go` 添加 Swagger 路由
  - 挂载 `/swagger/*any` 路由
  - 导入生成的 docs 包
  - **Verify**: 应用启动正常

---

## Phase 2: Handler 注解

### 2.1 Auth 模块

- [ ] **2.1.1** 注解 `auth/handler.go`
  - POST /auth/login - InitiateLogin
  - POST /auth/callback - Callback
  - POST /auth/refresh - RefreshToken
  - POST /auth/logout - Logout
  - GET /users/me - GetCurrentUser
  - **Verify**: swag init 成功

- [ ] **2.1.2** 注解 API Keys 端点
  - GET/POST/DELETE /keys
  - GET/POST/PATCH/DELETE /api-keys
  - **Verify**: swag init 成功

### 2.2 User 模块

- [ ] **2.2.1** 注解 `user/handler.go`
  - GET/PATCH /users/profile
  - POST /users/verify-email
  - **Verify**: swag init 成功

### 2.3 Billing 模块

- [ ] **2.3.1** 注解 `billing/handler.go`
  - 订阅相关端点
  - 额度相关端点
  - 用量相关端点
  - **Verify**: swag init 成功

### 2.4 Order 模块

- [ ] **2.4.1** 注解 `order/handler.go`
  - POST /orders/subscription
  - POST /orders/topup
  - GET /orders
  - GET /orders/:id
  - POST /orders/:id/cancel
  - GET /invoices
  - **Verify**: swag init 成功

### 2.5 Payment 模块

- [ ] **2.5.1** 注解 `payment/handler.go`
  - POST /payments/intent
  - POST /payments/native
  - GET /payments/methods
  - GET /payments/:id
  - **Verify**: swag init 成功

### 2.6 Git 模块

- [ ] **2.6.1** 注解 `git/handler.go`
  - 仓库管理端点
  - LFS 端点
  - **Verify**: swag init 成功

### 2.7 Collaboration 模块

- [ ] **2.7.1** 注解 `collaboration/handler.go`
  - 团队端点
  - 邀请端点
  - **Verify**: swag init 成功

### 2.8 AI 模块

- [ ] **2.8.1** 注解 AI 相关 handler
  - Provider pool 端点
  - **Verify**: swag init 成功

---

## Phase 3: 生成与验证

### 3.1 生成文档

- [ ] **3.1.1** 运行 swag init
  ```bash
  swag init -g cmd/server/docs.go -o docs
  ```
  - **Verify**: docs/ 目录生成

- [ ] **3.1.2** 验证 Swagger UI
  - 启动应用
  - 访问 http://localhost:8080/swagger/index.html
  - **Verify**: 所有端点正确显示

### 3.2 CI 集成

- [ ] **3.2.1** 添加 Makefile 命令
  ```makefile
  swagger:
      swag init -g cmd/server/docs.go -o docs
  ```

- [ ] **3.2.2** 添加 CI 验证步骤
  - 运行 swag init
  - 验证无文件变更
  - **Verify**: CI 通过

---

## Dependencies

```
Phase 1 ──▶ Phase 2 ──▶ Phase 3
```

- Phase 2 的各模块可并行
- Phase 3 依赖 Phase 1 和 Phase 2 完成

## Parallelizable Work

- 2.1 - 2.8 所有模块注解可并行

## Estimated Effort

| Phase | 工作量 | 状态 |
|-------|--------|------|
| Phase 1 | 小 | 待开始 |
| Phase 2 | 中 | 待开始 |
| Phase 3 | 小 | 待开始 |
