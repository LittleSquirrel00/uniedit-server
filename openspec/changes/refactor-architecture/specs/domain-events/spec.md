# domain-events Specification

## Purpose

引入领域事件系统，通过事件驱动解耦模块间的交互。

## ADDED Requirements

### Requirement: Event Infrastructure

系统 SHALL 提供领域事件基础设施。

#### Scenario: Event interface

- **WHEN** 定义领域事件
- **THEN** 所有事件实现 `Event` 接口
- **AND** 接口包含 EventID, EventType, OccurredAt, AggregateID, AggregateType 方法

#### Scenario: Event bus

- **WHEN** 发布领域事件
- **THEN** 通过 EventBus 发布
- **AND** 已注册的 Handler 同步接收事件
- **AND** 单个 Handler 失败不影响其他 Handler

#### Scenario: Handler registration

- **WHEN** 应用启动
- **THEN** 各模块的 EventHandler 注册到 EventBus
- **AND** Handler 通过 Handles() 声明处理的事件类型

### Requirement: Payment Domain Events

Payment 模块 SHALL 发布支付相关领域事件。

#### Scenario: PaymentSucceeded event

- **GIVEN** 支付成功
- **WHEN** 处理支付回调
- **THEN** 发布 PaymentSucceeded 事件
- **AND** 事件包含 PaymentID, OrderID, UserID, Amount, Currency

#### Scenario: PaymentFailed event

- **GIVEN** 支付失败
- **WHEN** 处理支付失败回调
- **THEN** 发布 PaymentFailed 事件
- **AND** 事件包含 PaymentID, OrderID, FailureCode, FailureMessage

### Requirement: Order Event Handler

Order 模块 SHALL 处理支付相关事件。

#### Scenario: Handle PaymentSucceeded

- **GIVEN** PaymentSucceeded 事件发布
- **WHEN** Order EventHandler 接收事件
- **THEN** 将对应订单状态更新为 Paid
- **AND** 设置 PaidAt 时间戳

#### Scenario: Handle PaymentFailed

- **GIVEN** PaymentFailed 事件发布
- **WHEN** Order EventHandler 接收事件
- **THEN** 将对应订单状态更新为 Failed

### Requirement: Billing Event Handler

Billing 模块 SHALL 处理需要充值的事件。

#### Scenario: Handle PaymentSucceeded for topup

- **GIVEN** PaymentSucceeded 事件发布
- **AND** 订单类型为 topup
- **WHEN** Billing EventHandler 接收事件
- **THEN** 为用户增加对应金额的 Credits

### Requirement: Event Error Handling

事件处理 SHALL 有健壮的错误处理。

#### Scenario: Handler error isolation

- **GIVEN** 多个 Handler 处理同一事件
- **WHEN** 其中一个 Handler 失败
- **THEN** 其他 Handler 继续执行
- **AND** 错误被记录到日志

#### Scenario: Idempotent handling

- **GIVEN** 同一事件可能重复投递
- **WHEN** Handler 处理事件
- **THEN** 处理逻辑应该是幂等的
- **AND** 重复处理不产生副作用
