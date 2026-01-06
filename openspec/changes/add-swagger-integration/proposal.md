# Proposal: Add Swagger Integration

## Summary

为 uniedit-server 添加 Swagger/OpenAPI 文档支持，使用 swaggo/swag 自动从代码注解生成 API 文档，并集成 Swagger UI 提供交互式文档界面。

## Motivation

当前项目有 8 个业务模块，超过 40 个 API 端点，但没有统一的 API 文档：
- 前端开发者需要阅读代码才能了解 API 契约
- 涉及支付、计费等敏感接口，缺乏清晰文档增加对接风险
- 无法方便地测试 API

## Scope

### In Scope
- 安装 swaggo/swag 依赖
- 添加主文档注解 (cmd/server/main.go)
- 为所有 handler 添加 Swagger 注解
- 集成 Swagger UI 路由
- 添加 swag init 到构建流程

### Out of Scope
- 自动生成客户端 SDK（可后续扩展）
- API 版本管理（当前只有 v1）

## Capabilities Affected

| Capability | Change Type |
|------------|-------------|
| api-documentation | ADDED |

## Design Considerations

### 技术选型

| 方案 | 优点 | 缺点 |
|------|------|------|
| swaggo/swag | 社区成熟，Gin 原生支持，注解式 | 注解较冗长 |
| go-swagger | 功能强大，规范标准 | 学习曲线陡峭 |
| 手写 OpenAPI | 灵活 | 易与代码不同步 |

**选择 swaggo/swag**：与 Gin 框架集成良好，维护成本可控。

### 文档结构

```
docs/
├── docs.go         # 生成的 Go 代码
├── swagger.json    # OpenAPI JSON
└── swagger.yaml    # OpenAPI YAML
```

### API 分组策略

按模块分组 (Tags)：
- Auth - 认证相关
- User - 用户管理
- Billing - 计费
- Order - 订单
- Payment - 支付
- AI - AI 服务
- Git - Git 托管
- Collaboration - 协作

## Risks

| 风险 | 影响 | 缓解措施 |
|------|------|----------|
| 注解与代码不同步 | 中 | CI 中验证 swag init 无差异 |
| 敏感信息泄露 | 高 | 生产环境可禁用 Swagger UI |

## Success Criteria

- [ ] `swag init` 成功生成文档
- [ ] Swagger UI 可访问且展示所有端点
- [ ] 所有公开 API 有完整的请求/响应文档
- [ ] CI 验证文档与代码同步
