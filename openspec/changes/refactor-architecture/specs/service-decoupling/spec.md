# service-decoupling Specification

## Purpose

消除 Service 层之间的直接依赖，改为依赖 Repository 接口，遵循依赖倒置原则。

## ADDED Requirements

### Requirement: Service Dependency Inversion

Service 层 SHALL 只依赖 Repository 接口，不依赖其他模块的 Service。

#### Scenario: Payment Service depends on Order Repository

- **GIVEN** Payment Service 需要查询订单信息
- **WHEN** 处理支付逻辑
- **THEN** 通过 OrderReader 接口获取订单
- **AND** 不直接调用 order.Service 方法

#### Scenario: Define interface at consumer

- **GIVEN** Payment 模块需要订单数据
- **WHEN** 定义依赖接口
- **THEN** OrderReader 接口定义在 payment 包中
- **AND** 接口只包含 Payment 需要的方法

### Requirement: Cross-Module Repository Interface

跨模块 Repository 接口 SHALL 定义在使用方模块。

#### Scenario: OrderReader interface

- **WHEN** payment 模块需要读取订单
- **THEN** 在 `payment/deps.go` 中定义 OrderReader 接口
- **AND** 接口方法返回 payment 模块需要的精简视图

#### Scenario: BillingReader interface

- **WHEN** order 模块需要读取计划信息
- **THEN** 在 `order/deps.go` 中定义 BillingReader 接口
- **AND** order.Repository 不需要了解 billing 内部实现

### Requirement: Slim View Objects

跨模块数据传输 SHALL 使用精简视图对象。

#### Scenario: OrderInfo for Payment

- **GIVEN** Payment 模块需要订单信息
- **WHEN** 定义数据结构
- **THEN** 使用 `OrderInfo` 而非完整 `Order` 模型
- **AND** 只包含 Payment 需要的字段

## MODIFIED Requirements

### Requirement: Payment Service Constructor

Payment Service 构造函数 SHALL 接受 Repository 接口而非 Service。

#### Scenario: New constructor signature

- **WHEN** 创建 Payment Service
- **THEN** 构造函数签名为:
  ```go
  func NewService(
      repo Repository,
      orderRepo OrderReader,     // Repository 接口
      eventBus *events.Bus,      // 事件总线
      registry *ProviderRegistry,
      logger *zap.Logger,
  ) *Service
  ```
- **AND** 不再接受 `*order.Service` 参数
