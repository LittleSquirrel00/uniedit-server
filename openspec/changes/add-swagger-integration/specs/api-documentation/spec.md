# api-documentation Specification

## Purpose

提供自动生成的 API 文档，使开发者能够快速了解和测试 API 接口。

## ADDED Requirements

### Requirement: Swagger Documentation Generation

API 文档 SHALL 从代码注解自动生成。

#### Scenario: Generate OpenAPI spec

- **WHEN** 运行 `swag init` 命令
- **THEN** 在 `docs/` 目录生成 `swagger.json` 和 `swagger.yaml`
- **AND** 生成 `docs.go` 供 Go 代码导入
- **AND** 文档包含所有公开 API 端点

#### Scenario: Handler annotation format

- **GIVEN** 一个 HTTP handler 函数
- **WHEN** 添加 Swagger 注解
- **THEN** 注解包含 @Summary（简要描述）
- **AND** 注解包含 @Description（详细描述，可选）
- **AND** 注解包含 @Tags（API 分组）
- **AND** 注解包含 @Accept 和 @Produce（内容类型）
- **AND** 注解包含 @Param（请求参数）
- **AND** 注解包含 @Success 和 @Failure（响应）
- **AND** 注解包含 @Router（路由路径和方法）
- **AND** 需要认证的端点包含 @Security

### Requirement: Swagger UI Integration

Swagger UI SHALL 提供交互式 API 文档界面。

#### Scenario: Access Swagger UI

- **GIVEN** 应用正在运行
- **WHEN** 访问 `/swagger/index.html`
- **THEN** 显示 Swagger UI 界面
- **AND** 界面展示所有 API 端点
- **AND** 端点按 Tags 分组

#### Scenario: Try out API

- **GIVEN** Swagger UI 已加载
- **WHEN** 用户点击某个端点的 "Try it out"
- **THEN** 可以输入请求参数
- **AND** 可以发送实际请求
- **AND** 显示响应结果

#### Scenario: Authentication in Swagger UI

- **GIVEN** API 需要 Bearer Token 认证
- **WHEN** 用户点击 "Authorize" 按钮
- **THEN** 可以输入 Bearer Token
- **AND** 后续请求自动携带 Authorization header

### Requirement: API Grouping

API 端点 SHALL 按功能模块分组。

#### Scenario: Tag-based grouping

- **WHEN** 查看 Swagger 文档
- **THEN** 端点按以下 Tags 分组：
  - Auth（认证）
  - User（用户）
  - Billing（计费）
  - Order（订单）
  - Payment（支付）
  - AI（AI 服务）
  - Git（Git 托管）
  - Collaboration（协作）

### Requirement: Request/Response Schema

API 文档 SHALL 包含完整的请求和响应 Schema。

#### Scenario: Request body schema

- **GIVEN** 一个 POST 端点
- **WHEN** 查看文档
- **THEN** 显示请求体的 JSON Schema
- **AND** 包含字段类型、是否必填
- **AND** 包含字段描述（如有）

#### Scenario: Response schema

- **GIVEN** 任意 API 端点
- **WHEN** 查看文档
- **THEN** 显示成功响应的 JSON Schema
- **AND** 显示错误响应的格式
- **AND** 包含 HTTP 状态码说明

### Requirement: Documentation Synchronization

API 文档 SHALL 与代码保持同步。

#### Scenario: CI validation

- **GIVEN** CI 流水线运行
- **WHEN** 执行文档验证步骤
- **THEN** 运行 `swag init` 重新生成文档
- **AND** 检查 `docs/` 目录无变更
- **AND** 有变更时 CI 失败并提示更新文档

## MODIFIED Requirements

（无）

## REMOVED Requirements

（无）
