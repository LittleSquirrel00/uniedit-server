# model-separation Specification

## Purpose

分离领域模型与持久化模型，使领域模型保持纯净，不受数据库技术细节污染。

## ADDED Requirements

### Requirement: Domain Model Purity

领域模型 SHALL 不包含任何持久化框架的标记。

#### Scenario: No GORM tags in domain

- **WHEN** 定义领域模型（如 Order, Payment）
- **THEN** struct 不包含 `gorm:` tags
- **AND** 不包含 `json:` tags
- **AND** 字段使用私有访问（小写开头）
- **AND** 通过 Getter 方法暴露数据

#### Scenario: Factory method for creation

- **WHEN** 创建新的聚合根
- **THEN** 使用工厂方法（如 NewOrder）
- **AND** 工厂方法执行业务验证
- **AND** 返回创建的实例或错误

#### Scenario: Domain behavior methods

- **WHEN** 需要修改聚合状态
- **THEN** 通过领域方法（如 MarkAsPaid）
- **AND** 方法内执行状态转换验证
- **AND** 方法返回错误表示无效操作

### Requirement: Persistence Entity

持久化实体 SHALL 专门用于数据库交互。

#### Scenario: Entity with GORM tags

- **WHEN** 定义持久化实体
- **THEN** struct 包含完整的 `gorm:` tags
- **AND** 字段使用公开访问（大写开头）
- **AND** 包含 TableName() 方法

#### Scenario: Entity in separate package

- **WHEN** 组织代码结构
- **THEN** 持久化实体放在 `entity/` 子包
- **AND** 领域模型放在 `domain/` 子包

### Requirement: Model Conversion

Repository SHALL 负责领域模型与持久化实体的转换。

#### Scenario: Entity to Domain

- **GIVEN** 从数据库查询得到 Entity
- **WHEN** Repository 返回结果
- **THEN** 调用 entity.ToDomain() 转换
- **AND** 返回领域模型实例

#### Scenario: Domain to Entity

- **GIVEN** 需要保存领域模型
- **WHEN** Repository 执行保存
- **THEN** 调用 entity.FromDomain(domain) 转换
- **AND** 保存 Entity 到数据库

#### Scenario: Restore method for hydration

- **WHEN** 从持久化数据恢复领域模型
- **THEN** 使用 RestoreXxx 方法（非 NewXxx）
- **AND** Restore 方法跳过业务验证
- **AND** 直接设置所有字段值

### Requirement: Value Objects

通用业务概念 SHALL 使用值对象封装。

#### Scenario: Money value object

- **WHEN** 表示金额
- **THEN** 使用 Money 值对象
- **AND** Money 包含 amount（分）和 currency
- **AND** Money 是不可变的
- **AND** 提供 Add, Equals 等方法

#### Scenario: Value object equality

- **GIVEN** 两个 Money 实例
- **WHEN** 比较相等性
- **THEN** 基于所有字段值比较
- **AND** 不基于引用比较

## MODIFIED Requirements

### Requirement: Repository Interface

Repository 接口 SHALL 使用领域模型作为参数和返回值。

#### Scenario: Repository methods signature

- **WHEN** 定义 Repository 接口
- **THEN** 方法参数使用领域模型类型
- **AND** 方法返回值使用领域模型类型
- **AND** 不暴露 Entity 类型

### Requirement: Service Layer

Service 层 SHALL 只操作领域模型。

#### Scenario: Service uses domain model

- **WHEN** Service 执行业务逻辑
- **THEN** 操作领域模型实例
- **AND** 调用领域模型的行为方法
- **AND** 不直接操作 Entity
