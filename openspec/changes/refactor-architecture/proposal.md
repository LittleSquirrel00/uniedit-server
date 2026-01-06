# Proposal: Refactor Architecture

## Change ID
`refactor-architecture`

## Summary

重构项目架构以提升模块解耦和领域模型清晰度，包含三个核心改进：

1. **Service 间依赖改为 Repository 依赖** - 消除 Service 层的直接跨模块依赖
2. **引入领域事件系统** - 通过事件驱动解耦模块间交互
3. **分离领域模型与持久化模型** - 消除 Model 中的 GORM tags 污染

## Motivation

### 当前问题

1. **Service 间直接依赖**
   ```go
   // payment/service.go - 违反依赖倒置原则
   type Service struct {
       orderService   *order.Service      // ❌ 直接依赖具体 Service
       billingService billing.ServiceInterface
   }
   ```

2. **模块耦合度高**
   - 支付成功后，payment.Service 直接调用 order.MarkAsPaid() 和 billing.AddCredits()
   - 无法独立测试单个模块
   - 添加新功能需要修改多个模块

3. **领域模型与持久化混杂**
   ```go
   // order/model.go - 领域逻辑与 GORM tags 混合
   type Order struct {
       ID     uuid.UUID   `json:"id" gorm:"type:uuid;primaryKey"`  // ❌ 持久化细节
       Status OrderStatus `json:"status" gorm:"not null;default:pending"`
   }
   ```

### 改进目标

- **依赖方向**：上层模块只依赖下层的 Repository 接口，不依赖 Service
- **事件驱动**：模块间通过领域事件通信，实现松耦合
- **模型分离**：领域模型（Domain Model）与持久化模型（Entity）分离

## Scope

### In Scope

1. **Phase 1: Service 解耦**
   - 定义跨模块 Repository 接口
   - 重构 payment.Service 依赖
   - 移除 Service 间的直接调用

2. **Phase 2: 领域事件系统**
   - 创建事件基础设施 (EventBus, EventHandler)
   - 定义核心领域事件 (PaymentSucceeded, OrderPaid 等)
   - 实现事件发布/订阅

3. **Phase 3: 模型分离**
   - 创建纯领域模型 (无 GORM tags)
   - 创建持久化实体 (Entity with GORM tags)
   - Repository 负责 Domain <-> Entity 转换

### Out of Scope

- 数据库 schema 变更
- API 接口变更
- 前端/客户端修改

## Dependencies

- 不依赖其他 OpenSpec changes
- 需要在现有功能稳定后进行

## Risks

| 风险 | 影响 | 缓解措施 |
|------|------|---------|
| 重构范围大 | 可能引入 bug | 分阶段实施，每阶段完整测试 |
| 事件系统复杂度 | 调试困难 | 先用同步事件，后续可改异步 |
| 模型转换开销 | 性能下降 | 使用高效转换，热路径优化 |

## Success Criteria

1. 所有 Service 不再直接依赖其他模块的 Service
2. 跨模块交互通过事件或 Repository 完成
3. 领域模型不包含任何 GORM/持久化 tags
4. 现有测试全部通过
5. 构建成功，无编译错误

## Related Specs

- [service-decoupling](specs/service-decoupling/spec.md) - Service 解耦规范
- [domain-events](specs/domain-events/spec.md) - 领域事件规范
- [model-separation](specs/model-separation/spec.md) - 模型分离规范
