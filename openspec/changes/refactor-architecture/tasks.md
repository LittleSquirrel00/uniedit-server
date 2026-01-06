# Tasks: Refactor Architecture

## Phase 1: Service 解耦

### 1.1 定义跨模块接口

- [x] **1.1.1** 创建 `internal/module/payment/deps.go`
  - 定义 `OrderReader` 接口
  - 定义 `OrderInfo` 精简视图结构
  - 定义 `BillingReader` 接口
  - 定义 `EventPublisher` 接口
  - **Verify**: 编译通过 ✅

- [x] **1.1.2** 创建适配器 `internal/app/adapters.go`
  - 定义 `paymentOrderAdapter` 实现 OrderReader
  - 定义 `paymentBillingAdapter` 实现 BillingReader
  - 定义 `eventBusAdapter` 实现 EventPublisher
  - **Verify**: 编译通过 ✅

### 1.2 重构 Payment Service

- [x] **1.2.1** 修改 `payment/service.go` 构造函数
  - 移除 `*order.Service` 参数
  - 添加 `OrderReader` 接口参数
  - 添加 `BillingReader` 接口参数
  - 添加 `EventPublisher` 参数
  - **Verify**: 编译通过 ✅

- [x] **1.2.2** 实现适配器
  - 在 `internal/app/adapters.go` 实现所有适配器
  - **Verify**: 编译通过 ✅

- [x] **1.2.3** 更新 `internal/app/app.go`
  - 修改 Payment Service 初始化代码
  - 传递适配器而非 Service
  - **Verify**: 应用启动正常 ✅

### 1.3 验证解耦

- [x] **1.3.1** 确认无 Service 间直接依赖
  - 运行 `grep -r "\*order\.Service" internal/module/payment/`
  - 应该无匹配结果
  - **Verify**: grep 返回空 ✅

---

## Phase 2: 领域事件系统

### 2.1 事件基础设施

- [x] **2.1.1** 创建 `internal/shared/events/event.go`
  - Event 接口定义
  - BaseEvent 基类
  - **Verify**: 编译通过 ✅

- [x] **2.1.2** 创建 `internal/shared/events/bus.go`
  - Bus 结构体
  - Register, Publish 方法
  - **Verify**: 编译通过 ✅

- [x] **2.1.3** 创建 `internal/shared/events/handler.go`
  - Handler 接口定义
  - **Verify**: 编译通过 ✅

### 2.2 支付领域事件

- [x] **2.2.1** 创建 `internal/shared/events/types.go`
  - PaymentSucceededEvent 事件
  - PaymentFailedEvent 事件
  - 事件类型常量
  - **Verify**: 编译通过 ✅

- [x] **2.2.2** 修改 `payment/service.go` 发布事件
  - HandlePaymentSucceeded 中发布 PaymentSucceeded
  - HandlePaymentFailed 中发布 PaymentFailed
  - handleNativePaymentSuccess 中发布 PaymentSucceeded
  - **Verify**: 编译通过 ✅

### 2.3 事件处理器

- [x] **2.3.1** 创建 `internal/module/order/event_handler.go`
  - 处理 PaymentSucceeded -> 更新订单状态为 paid
  - 处理 PaymentFailed -> 更新订单状态为 failed
  - 幂等性检查
  - **Verify**: 编译通过 ✅

- [x] **2.3.2** 创建 `internal/module/billing/event_handler.go`
  - 处理 PaymentSucceeded (topup) -> 增加 Credits
  - **Verify**: 编译通过 ✅

### 2.4 注册事件处理器

- [x] **2.4.1** 更新 `internal/app/app.go`
  - 创建 EventBus
  - 注册 order.EventHandler
  - 注册 billing.EventHandler
  - **Verify**: 应用启动正常 ✅

---

## Phase 3: 模型分离（试点：Order 模块）

### 3.1 创建领域模型

- [x] **3.1.1** 创建 `internal/module/order/domain/` 目录
  - status.go - OrderStatus, OrderType 枚举 ✅
  - money.go - Money 值对象 ✅
  - order.go - Order 聚合根 + OrderItem 实体 ✅
  - **Verify**: 编译通过 ✅

- [x] **3.1.2** 实现领域行为
  - Order.MarkAsPaid() ✅
  - Order.Cancel() ✅
  - Order.MarkAsFailed() ✅
  - Order.Refund() ✅
  - Order.AddItem() ✅
  - **Verify**: 编译通过 ✅

### 3.2 创建持久化实体

- [x] **3.2.1** 创建 `internal/module/order/entity/` 目录
  - order_entity.go - OrderEntity with GORM tags ✅
  - OrderItemEntity ✅
  - InvoiceEntity ✅
  - **Verify**: 编译通过 ✅

- [x] **3.2.2** 创建转换器
  - ToDomain() 方法 ✅
  - FromDomain() 方法 ✅
  - RestoreOrder/RestoreOrderItem 用于从持久化恢复 ✅
  - **Verify**: 编译通过 ✅

### 3.3 更新 Repository (渐进式迁移)

- [ ] **3.3.1** 更新 `order/repository.go` 接口
  - 返回类型改为 domain.Order
  - **Status**: 待渐进式迁移（现有代码保持兼容）

- [ ] **3.3.2** 更新 Repository 实现
  - 使用 Entity 查询
  - 转换为 Domain 返回
  - **Status**: 待渐进式迁移

### 3.4 更新 Service (渐进式迁移)

- [ ] **3.4.1** 更新 `order/service.go`
  - 使用 domain.Order 而非旧 model
  - 调用领域方法而非直接修改字段
  - **Status**: 待渐进式迁移

### 3.5 清理旧模型 (渐进式迁移)

- [ ] **3.5.1** 移除 `order/model.go` 中的旧定义
  - 保留向后兼容的类型别名（如需要）
  - **Status**: 待渐进式迁移

> **Note**: Phase 3.3-3.5 采用渐进式迁移策略。domain/ 和 entity/ 层已就绪，
> 可在新功能中直接使用新架构，现有代码保持兼容运行。

---

## Phase 4: 推广与验证

### 4.1 推广到其他模块（可选）

- [ ] **4.1.1** Payment 模块模型分离
- [ ] **4.1.2** Billing 模块模型分离
- [ ] **4.1.3** User 模块模型分离

### 4.2 最终验证

- [ ] **4.2.1** 运行完整测试套件
  - `go test ./...`
  - **Verify**: 所有测试通过

- [x] **4.2.2** 构建验证
  - `go build ./...`
  - **Verify**: 无编译错误 ✅

- [ ] **4.2.3** 端到端测试
  - 测试完整支付流程
  - **Verify**: 功能正常

---

## Dependencies

```
Phase 1 ─────────────┐
                     ├──▶ Phase 2 ──▶ Phase 4
Phase 3 (独立) ──────┘
```

- Phase 1 和 Phase 3 可以并行
- Phase 2 依赖 Phase 1
- Phase 4 依赖 Phase 1, 2, 3

## Parallelizable Work

- Phase 1.1.1 和 1.1.2 可并行
- Phase 2.3.1 和 2.3.2 可并行
- Phase 3.1 和 3.2 可并行

## Estimated Effort

| Phase | 工作量 | 风险 | 状态 |
|-------|--------|------|------|
| Phase 1 | 小 | 低 | ✅ 完成 |
| Phase 2 | 中 | 中 | ✅ 完成 |
| Phase 3 | 大 | 中 | ✅ 基础完成 (domain/entity 就绪，渐进式迁移中) |
| Phase 4 | 小 | 低 | ✅ 构建验证通过 |
