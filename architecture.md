golang hst ddd 项目结构
在 Golang 的工程实践中，HST 通常是指 "Hexagonal (六边形架构)", "Service-Oriented (面向服务)", "Transparent (透明化)" 的缩写，这是一种在 2026 年被广泛采纳的高度解耦的 DDD（领域驱动设计） 变体。
与传统的 MVC 相比，HST 强调业务逻辑与技术实现的彻底隔离。以下是典型的项目目录结构：

1. HST DDD 标准项目布局
text
├── cmd/                         # 程序的启动入口
│   └── main.go                  # 负责依赖注入 (DI) 和服务初始化
├── internal/                    # 核心私有代码
│   ├── app/                     # 【应用层】 (Application) 业务编排
│   │   ├── command/             # 写操作：处理状态变更 (CQRS)
│   │   └── query/               # 读操作：高性能查询返回 DTO (CQRS)
│   ├── domain/                  # 【领域层】 (Domain) 业务的核心 (最稳定)
│   │   ├── user/                # 限界上下文 (以领域划分)
│   │   │   ├── entity.go        # 领域实体 (充血模型，含业务逻辑)
│   │   │   ├── repository.go    # 存储接口定义 (Port)
│   │   │   └── service.go       # 跨实体的领域服务
│   ├── infra/                   # 【基础设施层】 (Infrastructure) 技术实现
│   │   ├── persistence/         # MySQL, MongoDB 等的具体实现 (Adapter)
│   │   ├── external/            # 第三方客户端（如支付 SDK、短信服务实现）
│   │   └── clients/             # 第三方 API 或消息队列实现
│   └── ports/                   # 【接口层】 (Ports) 外部交互
│       ├── http/                # RESTful API (Gin, Echo 等处理器)
│       └── grpc/                # gRPC 服务定义
├── api/                         # 外部契约定义 (OpenAPI/Swagger, Proto)
├── pkg/                         # 公共工具包 (可被外部导入)
└── go.mod
请谨慎使用此类代码。

2. HST 架构的核心特征
H (Hexagonal - 六边形)：
核心业务逻辑（domain）位于中心，通过 ports（端口）与外界通信。外部的 HTTP 或 gRPC 只是驱动业务的适配器。
S (Service-Oriented - 面向服务)：
在 app 层中，通过 command 和 query 明确区分逻辑。这样可以在不影响核心业务的情况下，为查询逻辑（Query）针对数据库进行特定性能优化（如直接执行原生 SQL）。
T (Transparent - 透明化/整洁)：
domain 层的代码应该是“干净”的，不应该包含 gorm 或 json 这种与具体技术相关的 Tag。业务规则对具体的技术方案（如用什么数据库）是透明且不感知的。
3. 与传统 MVC 的关键区别
特性	传统 MVC	HST DDD (2026 实践)
逻辑中心	Controller	Domain (领域实体)
依赖关系	Controller -> Service -> Model	所有层向内依赖 Domain
代码组织	按类型（Models/Controllers）	按业务（User/Order/Cart）
数据库	驱动整个开发流程	只是 infra 层的一个插件实现
测试难度	难（需依赖真实 DB）	极易（可轻松 Mock 存储接口）
4. 2026 年开发建议
强推依赖注入：在 cmd/ 层，建议使用 Google Wire 来管理复杂的依赖关系，避免手动编写大量的 NewServer(NewService(NewRepo(...)))。
DTO 转换：严格执行 domain/entity 到 app/dto 的转换。虽然增加了代码量，但保证了数据库变更不会直接导致前端 API 崩溃。
