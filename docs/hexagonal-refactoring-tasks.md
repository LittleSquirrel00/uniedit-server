# UniEdit Server 六边形架构重构任务清单

## 概述

本文档定义了将 UniEdit Server 从当前模块化架构重构为六边形架构的详细任务清单。重构采用渐进式策略，确保系统在重构过程中保持可用。

### 重构目标

```
当前结构                              目标结构
internal/                            internal/
├── app/                             ├── app.go
├── module/                          ├── config.go
│   ├── auth/                        ├── domain/
│   │   ├── handler.go               │   ├── registry.go
│   │   ├── service.go               │   ├── auth/
│   │   ├── repository.go            │   ├── user/
│   │   └── ...                      │   ├── billing/
│   ├── billing/                     │   └── ...
│   ├── ai/                          ├── port/
│   └── ...                          │   ├── inbound/
├── shared/                          │   └── outbound/
│   ├── config/                      ├── adapter/
│   ├── database/                    │   ├── inbound/
│   └── ...                          │   └── outbound/
└── ...                              ├── model/
                                     └── migration/
```

### 重构原则

1. **渐进式迁移** - 一次迁移一个实体，保持系统可用
2. **测试先行** - 迁移前确保测试覆盖，迁移后验证通过
3. **依赖向内** - 严格遵循依赖方向规则
4. **接口优先** - 先定义 Port 接口，再实现 Adapter

---

## Phase 0: 准备工作

### P0.1 创建目录结构骨架

**任务**: 创建新的目录结构，不删除现有代码

```bash
# 执行脚本
mkdir -p internal/domain
mkdir -p internal/port/inbound
mkdir -p internal/port/outbound
mkdir -p internal/adapter/inbound/gin
mkdir -p internal/adapter/inbound/grpc
mkdir -p internal/adapter/inbound/command
mkdir -p internal/adapter/outbound/postgres
mkdir -p internal/adapter/outbound/redis
mkdir -p internal/adapter/outbound/r2
mkdir -p internal/adapter/outbound/http
mkdir -p internal/adapter/outbound/oauth
mkdir -p internal/adapter/outbound/rabbitmq
mkdir -p internal/model
```

**产出文件**:
- [ ] `internal/domain/registry.go` - Domain 注册表骨架
- [ ] `internal/port/inbound/registry.go` - Inbound Port 注册表骨架
- [ ] `internal/port/outbound/registry.go` - Outbound Port 注册表骨架

**验证**:
```bash
tree internal/ -L 3 -d
```

---

### P0.2 创建通用 Port 定义

**任务**: 定义通用的基础设施端口接口

**产出文件**:

#### `internal/port/outbound/database.go`
```go
package outbound

import (
    "context"
    "gorm.io/gorm"
)

// DatabaseExecutor defines generic database operations.
type DatabaseExecutor interface {
    DB() *gorm.DB
    Transaction(ctx context.Context, fn func(tx *gorm.DB) error) error
}
```

#### `internal/port/outbound/cache.go`
```go
package outbound

import (
    "context"
    "time"
)

// CachePort defines generic cache operations.
type CachePort interface {
    Get(ctx context.Context, key string) ([]byte, error)
    Set(ctx context.Context, key string, value []byte, ttl time.Duration) error
    Delete(ctx context.Context, key string) error
    Exists(ctx context.Context, key string) (bool, error)
}
```

#### `internal/port/outbound/storage.go`
```go
package outbound

import (
    "context"
    "io"
    "time"
)

// StoragePort defines object storage operations.
type StoragePort interface {
    Put(ctx context.Context, key string, reader io.Reader, size int64) error
    Get(ctx context.Context, key string) (io.ReadCloser, error)
    Delete(ctx context.Context, key string) error
    GetPresignedURL(ctx context.Context, key string, duration time.Duration) (string, error)
}
```

#### `internal/port/outbound/message.go`
```go
package outbound

import "context"

// EventPublisherPort defines event publishing operations.
type EventPublisherPort interface {
    Publish(ctx context.Context, event interface{}) error
}

// MessagePort defines message queue operations.
type MessagePort interface {
    Publish(ctx context.Context, topic string, message []byte) error
    Subscribe(ctx context.Context, topic string, handler func([]byte) error) error
}
```

---

### P0.3 创建通用 Model 定义

**任务**: 创建共享的请求/响应模型

**产出文件**:

#### `internal/model/request.go`
```go
package model

// PaginationRequest defines pagination parameters.
type PaginationRequest struct {
    Page     int `json:"page" form:"page"`
    PageSize int `json:"page_size" form:"page_size"`
}

// DefaultPagination returns default pagination values.
func (p *PaginationRequest) DefaultPagination() {
    if p.Page <= 0 {
        p.Page = 1
    }
    if p.PageSize <= 0 || p.PageSize > 100 {
        p.PageSize = 20
    }
}

// Offset returns the offset for database queries.
func (p *PaginationRequest) Offset() int {
    return (p.Page - 1) * p.PageSize
}
```

#### `internal/model/response.go`
```go
package model

// PaginatedResponse defines paginated response structure.
type PaginatedResponse[T any] struct {
    Data       []T   `json:"data"`
    Total      int64 `json:"total"`
    Page       int   `json:"page"`
    PageSize   int   `json:"page_size"`
    TotalPages int   `json:"total_pages"`
}

// ErrorResponse defines error response structure.
type ErrorResponse struct {
    Code    string `json:"code"`
    Message string `json:"message"`
    Details any    `json:"details,omitempty"`
}
```

---

## Phase 1: User 实体迁移（示范）

选择 User 作为第一个迁移实体，因为它相对简单且依赖较少。

### P1.1 定义 User Model

**任务**: 在 `internal/model/` 创建 User 相关数据结构

**产出文件**: `internal/model/user.go`

```go
package model

import (
    "time"
    "github.com/google/uuid"
)

// User represents user data structure.
type User struct {
    ID        uuid.UUID  `json:"id" gorm:"type:uuid;primaryKey"`
    Email     string     `json:"email" gorm:"uniqueIndex;not null"`
    Name      string     `json:"name"`
    AvatarURL string     `json:"avatar_url"`
    CreatedAt time.Time  `json:"created_at"`
    UpdatedAt time.Time  `json:"updated_at"`
    DeletedAt *time.Time `json:"deleted_at,omitempty" gorm:"index"`
}

// TableName returns the table name for GORM.
func (User) TableName() string {
    return "users"
}

// UserInput represents user creation/update input.
type UserInput struct {
    Email     string `json:"email" binding:"required,email"`
    Name      string `json:"name" binding:"required"`
    AvatarURL string `json:"avatar_url"`
}

// UserFilter represents user query filters.
type UserFilter struct {
    IDs    []uuid.UUID `json:"ids"`
    Email  string      `json:"email"`
    Search string      `json:"search"`
    PaginationRequest
}

// Profile represents user profile.
type Profile struct {
    UserID      uuid.UUID `json:"user_id"`
    DisplayName string    `json:"display_name"`
    Bio         string    `json:"bio"`
    AvatarURL   string    `json:"avatar_url"`
    UpdatedAt   time.Time `json:"updated_at"`
}

// Preferences represents user preferences.
type Preferences struct {
    UserID   uuid.UUID `json:"user_id"`
    Theme    string    `json:"theme"`
    Language string    `json:"language"`
    Timezone string    `json:"timezone"`
}
```

---

### P1.2 定义 User Outbound Port

**任务**: 定义 User 相关的出站端口接口

**产出文件**: `internal/port/outbound/user.go`

```go
package outbound

import (
    "context"
    "github.com/google/uuid"
    "github.com/uniedit/server/internal/model"
)

// UserDatabasePort defines user persistence operations.
type UserDatabasePort interface {
    // Create creates a new user.
    Create(ctx context.Context, user *model.User) error

    // FindByID finds a user by ID.
    FindByID(ctx context.Context, id uuid.UUID) (*model.User, error)

    // FindByEmail finds a user by email.
    FindByEmail(ctx context.Context, email string) (*model.User, error)

    // FindByFilter finds users by filter.
    FindByFilter(ctx context.Context, filter model.UserFilter) ([]*model.User, int64, error)

    // Update updates a user.
    Update(ctx context.Context, user *model.User) error

    // Delete soft deletes a user.
    Delete(ctx context.Context, id uuid.UUID) error
}

// ProfileDatabasePort defines profile persistence operations.
type ProfileDatabasePort interface {
    // GetProfile gets user profile.
    GetProfile(ctx context.Context, userID uuid.UUID) (*model.Profile, error)

    // UpdateProfile updates user profile.
    UpdateProfile(ctx context.Context, profile *model.Profile) error
}

// PreferencesDatabasePort defines preferences persistence operations.
type PreferencesDatabasePort interface {
    // GetPreferences gets user preferences.
    GetPreferences(ctx context.Context, userID uuid.UUID) (*model.Preferences, error)

    // UpdatePreferences updates user preferences.
    UpdatePreferences(ctx context.Context, prefs *model.Preferences) error
}

// AvatarStoragePort defines avatar storage operations.
type AvatarStoragePort interface {
    // Upload uploads user avatar.
    Upload(ctx context.Context, userID uuid.UUID, data []byte, contentType string) (string, error)

    // Delete deletes user avatar.
    Delete(ctx context.Context, userID uuid.UUID) error

    // GetURL gets avatar URL.
    GetURL(ctx context.Context, userID uuid.UUID) (string, error)
}
```

---

### P1.3 定义 User Inbound Port

**任务**: 定义 User 相关的入站端口接口

**产出文件**: `internal/port/inbound/user.go`

```go
package inbound

import "github.com/gin-gonic/gin"

// UserHttpPort defines HTTP handler interface for user operations.
type UserHttpPort interface {
    // GetProfile handles GET /users/me/profile
    GetProfile(c *gin.Context)

    // UpdateProfile handles PUT /users/me/profile
    UpdateProfile(c *gin.Context)

    // GetPreferences handles GET /users/me/preferences
    GetPreferences(c *gin.Context)

    // UpdatePreferences handles PUT /users/me/preferences
    UpdatePreferences(c *gin.Context)

    // UploadAvatar handles POST /users/me/avatar
    UploadAvatar(c *gin.Context)
}

// UserAdminPort defines admin interface for user management.
type UserAdminPort interface {
    // ListUsers handles GET /admin/users
    ListUsers(c *gin.Context)

    // GetUser handles GET /admin/users/:id
    GetUser(c *gin.Context)

    // SuspendUser handles POST /admin/users/:id/suspend
    SuspendUser(c *gin.Context)

    // DeleteUser handles DELETE /admin/users/:id
    DeleteUser(c *gin.Context)
}
```

---

### P1.4 创建 User Domain

**任务**: 创建 User 领域服务

**产出文件**: `internal/domain/user/domain.go`

```go
package user

import (
    "context"
    "errors"
    "github.com/google/uuid"
    "github.com/uniedit/server/internal/model"
    "github.com/uniedit/server/internal/port/outbound"
)

var (
    ErrUserNotFound = errors.New("user not found")
    ErrUserExists   = errors.New("user already exists")
)

// UserDomain defines user domain service interface.
type UserDomain interface {
    // GetProfile gets user profile.
    GetProfile(ctx context.Context, userID uuid.UUID) (*model.Profile, error)

    // UpdateProfile updates user profile.
    UpdateProfile(ctx context.Context, userID uuid.UUID, input *model.Profile) error

    // GetPreferences gets user preferences.
    GetPreferences(ctx context.Context, userID uuid.UUID) (*model.Preferences, error)

    // UpdatePreferences updates user preferences.
    UpdatePreferences(ctx context.Context, userID uuid.UUID, prefs *model.Preferences) error

    // UploadAvatar uploads user avatar.
    UploadAvatar(ctx context.Context, userID uuid.UUID, data []byte, contentType string) (string, error)

    // ListUsers lists users (admin).
    ListUsers(ctx context.Context, filter model.UserFilter) ([]*model.User, int64, error)

    // GetUser gets user by ID (admin).
    GetUser(ctx context.Context, id uuid.UUID) (*model.User, error)

    // SuspendUser suspends a user (admin).
    SuspendUser(ctx context.Context, id uuid.UUID, reason string) error

    // DeleteUser deletes a user (admin).
    DeleteUser(ctx context.Context, id uuid.UUID) error
}

// userDomain implements UserDomain.
type userDomain struct {
    userDB    outbound.UserDatabasePort
    profileDB outbound.ProfileDatabasePort
    prefsDB   outbound.PreferencesDatabasePort
    avatarSt  outbound.AvatarStoragePort
}

// NewUserDomain creates a new user domain service.
func NewUserDomain(
    userDB outbound.UserDatabasePort,
    profileDB outbound.ProfileDatabasePort,
    prefsDB outbound.PreferencesDatabasePort,
    avatarSt outbound.AvatarStoragePort,
) UserDomain {
    return &userDomain{
        userDB:    userDB,
        profileDB: profileDB,
        prefsDB:   prefsDB,
        avatarSt:  avatarSt,
    }
}

func (d *userDomain) GetProfile(ctx context.Context, userID uuid.UUID) (*model.Profile, error) {
    profile, err := d.profileDB.GetProfile(ctx, userID)
    if err != nil {
        return nil, err
    }
    if profile == nil {
        return nil, ErrUserNotFound
    }
    return profile, nil
}

func (d *userDomain) UpdateProfile(ctx context.Context, userID uuid.UUID, input *model.Profile) error {
    input.UserID = userID
    return d.profileDB.UpdateProfile(ctx, input)
}

func (d *userDomain) GetPreferences(ctx context.Context, userID uuid.UUID) (*model.Preferences, error) {
    return d.prefsDB.GetPreferences(ctx, userID)
}

func (d *userDomain) UpdatePreferences(ctx context.Context, userID uuid.UUID, prefs *model.Preferences) error {
    prefs.UserID = userID
    return d.prefsDB.UpdatePreferences(ctx, prefs)
}

func (d *userDomain) UploadAvatar(ctx context.Context, userID uuid.UUID, data []byte, contentType string) (string, error) {
    return d.avatarSt.Upload(ctx, userID, data, contentType)
}

func (d *userDomain) ListUsers(ctx context.Context, filter model.UserFilter) ([]*model.User, int64, error) {
    return d.userDB.FindByFilter(ctx, filter)
}

func (d *userDomain) GetUser(ctx context.Context, id uuid.UUID) (*model.User, error) {
    user, err := d.userDB.FindByID(ctx, id)
    if err != nil {
        return nil, err
    }
    if user == nil {
        return nil, ErrUserNotFound
    }
    return user, nil
}

func (d *userDomain) SuspendUser(ctx context.Context, id uuid.UUID, reason string) error {
    // Business logic for suspension
    // Could involve audit logging, notification, etc.
    return d.userDB.Delete(ctx, id)
}

func (d *userDomain) DeleteUser(ctx context.Context, id uuid.UUID) error {
    return d.userDB.Delete(ctx, id)
}
```

---

### P1.5 创建 User Postgres Adapter

**任务**: 实现 User 数据库适配器

**产出文件**: `internal/adapter/outbound/postgres/user.go`

```go
package postgres

import (
    "context"
    "errors"
    "github.com/google/uuid"
    "github.com/uniedit/server/internal/model"
    "github.com/uniedit/server/internal/port/outbound"
    "gorm.io/gorm"
)

// userAdapter implements outbound.UserDatabasePort.
type userAdapter struct {
    db *gorm.DB
}

// NewUserAdapter creates a new user database adapter.
func NewUserAdapter(db *gorm.DB) outbound.UserDatabasePort {
    return &userAdapter{db: db}
}

func (a *userAdapter) Create(ctx context.Context, user *model.User) error {
    return a.db.WithContext(ctx).Create(user).Error
}

func (a *userAdapter) FindByID(ctx context.Context, id uuid.UUID) (*model.User, error) {
    var user model.User
    err := a.db.WithContext(ctx).Where("id = ?", id).First(&user).Error
    if errors.Is(err, gorm.ErrRecordNotFound) {
        return nil, nil
    }
    if err != nil {
        return nil, err
    }
    return &user, nil
}

func (a *userAdapter) FindByEmail(ctx context.Context, email string) (*model.User, error) {
    var user model.User
    err := a.db.WithContext(ctx).Where("email = ?", email).First(&user).Error
    if errors.Is(err, gorm.ErrRecordNotFound) {
        return nil, nil
    }
    if err != nil {
        return nil, err
    }
    return &user, nil
}

func (a *userAdapter) FindByFilter(ctx context.Context, filter model.UserFilter) ([]*model.User, int64, error) {
    var users []*model.User
    var total int64

    query := a.db.WithContext(ctx).Model(&model.User{})

    if len(filter.IDs) > 0 {
        query = query.Where("id IN ?", filter.IDs)
    }
    if filter.Email != "" {
        query = query.Where("email = ?", filter.Email)
    }
    if filter.Search != "" {
        query = query.Where("name ILIKE ? OR email ILIKE ?", "%"+filter.Search+"%", "%"+filter.Search+"%")
    }

    if err := query.Count(&total).Error; err != nil {
        return nil, 0, err
    }

    filter.DefaultPagination()
    if err := query.Offset(filter.Offset()).Limit(filter.PageSize).Find(&users).Error; err != nil {
        return nil, 0, err
    }

    return users, total, nil
}

func (a *userAdapter) Update(ctx context.Context, user *model.User) error {
    return a.db.WithContext(ctx).Save(user).Error
}

func (a *userAdapter) Delete(ctx context.Context, id uuid.UUID) error {
    return a.db.WithContext(ctx).Delete(&model.User{}, "id = ?", id).Error
}

// Compile-time check
var _ outbound.UserDatabasePort = (*userAdapter)(nil)
```

---

### P1.6 创建 User HTTP Adapter

**任务**: 实现 User HTTP 处理适配器

**产出文件**: `internal/adapter/inbound/gin/user.go`

```go
package gin

import (
    "net/http"
    "github.com/gin-gonic/gin"
    "github.com/google/uuid"
    "github.com/uniedit/server/internal/domain/user"
    "github.com/uniedit/server/internal/model"
    "github.com/uniedit/server/internal/port/inbound"
)

// userAdapter implements inbound.UserHttpPort.
type userAdapter struct {
    domain user.UserDomain
}

// NewUserAdapter creates a new user HTTP adapter.
func NewUserAdapter(domain user.UserDomain) inbound.UserHttpPort {
    return &userAdapter{domain: domain}
}

// RegisterRoutes registers user routes.
func (a *userAdapter) RegisterRoutes(r *gin.RouterGroup) {
    users := r.Group("/users")
    {
        users.GET("/me/profile", a.GetProfile)
        users.PUT("/me/profile", a.UpdateProfile)
        users.GET("/me/preferences", a.GetPreferences)
        users.PUT("/me/preferences", a.UpdatePreferences)
        users.POST("/me/avatar", a.UploadAvatar)
    }
}

func (a *userAdapter) GetProfile(c *gin.Context) {
    userID := c.MustGet("user_id").(uuid.UUID)

    profile, err := a.domain.GetProfile(c.Request.Context(), userID)
    if err != nil {
        handleError(c, err)
        return
    }

    c.JSON(http.StatusOK, profile)
}

func (a *userAdapter) UpdateProfile(c *gin.Context) {
    userID := c.MustGet("user_id").(uuid.UUID)

    var input model.Profile
    if err := c.ShouldBindJSON(&input); err != nil {
        c.JSON(http.StatusBadRequest, model.ErrorResponse{
            Code:    "invalid_input",
            Message: err.Error(),
        })
        return
    }

    if err := a.domain.UpdateProfile(c.Request.Context(), userID, &input); err != nil {
        handleError(c, err)
        return
    }

    c.JSON(http.StatusOK, gin.H{"message": "profile updated"})
}

func (a *userAdapter) GetPreferences(c *gin.Context) {
    userID := c.MustGet("user_id").(uuid.UUID)

    prefs, err := a.domain.GetPreferences(c.Request.Context(), userID)
    if err != nil {
        handleError(c, err)
        return
    }

    c.JSON(http.StatusOK, prefs)
}

func (a *userAdapter) UpdatePreferences(c *gin.Context) {
    userID := c.MustGet("user_id").(uuid.UUID)

    var input model.Preferences
    if err := c.ShouldBindJSON(&input); err != nil {
        c.JSON(http.StatusBadRequest, model.ErrorResponse{
            Code:    "invalid_input",
            Message: err.Error(),
        })
        return
    }

    if err := a.domain.UpdatePreferences(c.Request.Context(), userID, &input); err != nil {
        handleError(c, err)
        return
    }

    c.JSON(http.StatusOK, gin.H{"message": "preferences updated"})
}

func (a *userAdapter) UploadAvatar(c *gin.Context) {
    userID := c.MustGet("user_id").(uuid.UUID)

    file, err := c.FormFile("avatar")
    if err != nil {
        c.JSON(http.StatusBadRequest, model.ErrorResponse{
            Code:    "invalid_file",
            Message: "avatar file is required",
        })
        return
    }

    f, err := file.Open()
    if err != nil {
        handleError(c, err)
        return
    }
    defer f.Close()

    data := make([]byte, file.Size)
    if _, err := f.Read(data); err != nil {
        handleError(c, err)
        return
    }

    url, err := a.domain.UploadAvatar(c.Request.Context(), userID, data, file.Header.Get("Content-Type"))
    if err != nil {
        handleError(c, err)
        return
    }

    c.JSON(http.StatusOK, gin.H{"url": url})
}

// Compile-time check
var _ inbound.UserHttpPort = (*userAdapter)(nil)
```

---

### P1.7 更新 Domain Registry

**任务**: 在 Domain Registry 中注册 User Domain

**产出文件**: `internal/domain/registry.go`

```go
package domain

import (
    "github.com/uniedit/server/internal/domain/user"
)

// Domain holds all domain services.
type Domain struct {
    User user.UserDomain
    // Auth    auth.AuthDomain      // Phase 2
    // Billing billing.BillingDomain // Phase 3
    // ...
}

// NewDomain creates domain services with dependencies.
func NewDomain(ports *OutboundPorts) *Domain {
    return &Domain{
        User: user.NewUserDomain(
            ports.UserDB,
            ports.ProfileDB,
            ports.PrefsDB,
            ports.AvatarStorage,
        ),
    }
}

// OutboundPorts holds all outbound port implementations.
type OutboundPorts struct {
    // User ports
    UserDB        outbound.UserDatabasePort
    ProfileDB     outbound.ProfileDatabasePort
    PrefsDB       outbound.PreferencesDatabasePort
    AvatarStorage outbound.AvatarStoragePort

    // Generic ports
    Database DatabaseExecutor
    Cache    CachePort
    Storage  StoragePort
}
```

---

### P1.8 编写 User Domain 测试

**任务**: 为 User Domain 编写单元测试

**产出文件**: `internal/domain/user/domain_test.go`

```go
package user

import (
    "context"
    "testing"
    "github.com/google/uuid"
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/mock"
    "github.com/uniedit/server/internal/model"
)

// MockUserDatabasePort is a mock implementation.
type MockUserDatabasePort struct {
    mock.Mock
}

func (m *MockUserDatabasePort) FindByID(ctx context.Context, id uuid.UUID) (*model.User, error) {
    args := m.Called(ctx, id)
    if args.Get(0) == nil {
        return nil, args.Error(1)
    }
    return args.Get(0).(*model.User), args.Error(1)
}

// ... other mock methods

func TestUserDomain_GetUser(t *testing.T) {
    t.Run("success", func(t *testing.T) {
        mockDB := new(MockUserDatabasePort)
        domain := NewUserDomain(mockDB, nil, nil, nil)

        userID := uuid.New()
        expectedUser := &model.User{
            ID:    userID,
            Email: "test@example.com",
            Name:  "Test User",
        }

        mockDB.On("FindByID", mock.Anything, userID).Return(expectedUser, nil)

        user, err := domain.GetUser(context.Background(), userID)

        assert.NoError(t, err)
        assert.Equal(t, expectedUser, user)
        mockDB.AssertExpectations(t)
    })

    t.Run("not found", func(t *testing.T) {
        mockDB := new(MockUserDatabasePort)
        domain := NewUserDomain(mockDB, nil, nil, nil)

        userID := uuid.New()
        mockDB.On("FindByID", mock.Anything, userID).Return(nil, nil)

        user, err := domain.GetUser(context.Background(), userID)

        assert.ErrorIs(t, err, ErrUserNotFound)
        assert.Nil(t, user)
    })
}
```

---

### P1.9 集成测试与验证

**任务**: 验证 User 模块迁移完成

**验证步骤**:

```bash
# 1. 运行单元测试
go test ./internal/domain/user/...

# 2. 检查依赖方向
grep -r "adapter" internal/domain/
# 应该无输出

# 3. 编译检查
go build ./...

# 4. 运行集成测试
go test ./tests/integration/user_test.go
```

---

## Phase 2: Auth 实体迁移

### P2.1 定义 Auth Model

**产出文件**: `internal/model/auth.go`

| 模型 | 说明 |
|------|------|
| `User` | 复用 P1 定义 |
| `RefreshToken` | 刷新令牌 |
| `APIKey` | API 密钥 |
| `Session` | 会话信息 |
| `AuditLog` | 审计日志 |

### P2.2 定义 Auth Ports

**产出文件**:
- `internal/port/outbound/auth.go` - 数据库、缓存、OAuth 端口
- `internal/port/inbound/auth.go` - HTTP、API Key 端口

### P2.3 创建 Auth Domain

**产出文件**:
- `internal/domain/auth/domain.go` - 领域服务
- `internal/domain/auth/user.go` - User 聚合根
- `internal/domain/auth/token.go` - Token 实体
- `internal/domain/auth/apikey.go` - API Key 实体
- `internal/domain/auth/event.go` - 领域事件

### P2.4 创建 Auth Adapters

**产出文件**:
- `internal/adapter/outbound/postgres/auth.go`
- `internal/adapter/outbound/redis/session.go`
- `internal/adapter/outbound/oauth/google.go`
- `internal/adapter/outbound/oauth/github.go`
- `internal/adapter/inbound/gin/auth.go`

### P2.5 测试与验证

---

## Phase 3: Billing 实体迁移

### P3.1 定义 Billing Model

**产出文件**: `internal/model/billing.go`

| 模型 | 说明 |
|------|------|
| `Subscription` | 订阅信息 |
| `Plan` | 套餐计划 |
| `Balance` | 余额 |
| `Usage` | 使用记录 |
| `Quota` | 配额 |

### P3.2 - P3.5 同上模式

---

## Phase 4: Order 实体迁移

### P4.1 - P4.5 同上模式

---

## Phase 5: Payment 实体迁移

### P5.1 - P5.5 同上模式

**特殊注意**: Payment 依赖 Order 和 Billing，需要定义跨域端口。

---

## Phase 6: AI 实体迁移

### P6.1 定义 AI Model

**产出文件**: `internal/model/ai.go`

| 模型 | 说明 |
|------|------|
| `Provider` | AI 提供商 |
| `AIModel` | AI 模型 |
| `AIRequest` | 请求 |
| `AIResponse` | 响应 |
| `AIStreamChunk` | 流式块 |

### P6.2 - P6.5 同上模式

**特殊注意**: AI 模块复杂度高，需要保留现有的 routing、pool 等子模块逻辑。

---

## Phase 7: Git 实体迁移

### P7.1 - P7.5 同上模式

---

## Phase 8: Media 实体迁移

### P8.1 - P8.5 同上模式

---

## Phase 9: Collaboration 实体迁移

### P9.1 - P9.5 同上模式

---

## Phase 10: 清理与优化

### P10.1 删除旧代码

**任务**: 删除 `internal/module/` 目录

```bash
# 确保所有测试通过后
rm -rf internal/module/
```

### P10.2 更新 shared 目录

**任务**: 将 `internal/shared/` 中的工具移动到合适位置

| 原位置 | 新位置 |
|--------|--------|
| `shared/config/` | `internal/config.go` |
| `shared/database/` | `internal/adapter/outbound/postgres/database.go` |
| `shared/cache/` | `internal/adapter/outbound/redis/cache.go` |
| `shared/middleware/` | `internal/adapter/inbound/gin/middleware.go` |
| `shared/errors/` | `internal/model/errors.go` |

### P10.3 更新 app.go

**任务**: 重写应用组装层

**产出文件**: `internal/app.go`

### P10.4 更新文档

**任务**: 更新项目文档

- [ ] 更新 `README.md`
- [ ] 更新 `CLAUDE.md`
- [ ] 生成 API 文档

---

## 任务追踪表

| Phase | 任务 | 状态 | 负责人 | 备注 |
|-------|------|------|--------|------|
| P0.1 | 创建目录结构 | ✅ | Claude | 完成 |
| P0.2 | 通用 Port 定义 | ✅ | Claude | database/cache/storage/message |
| P0.3 | 通用 Model 定义 | ✅ | Claude | request/response |
| P1.1 | User Model | ✅ | Claude | 完成 |
| P1.2 | User Outbound Port | ✅ | Claude | 完成 |
| P1.3 | User Inbound Port | ✅ | Claude | 完成 |
| P1.4 | User Domain | ✅ | Claude | 完成 |
| P1.5 | User Postgres Adapter | ✅ | Claude | 完成 |
| P1.6 | User HTTP Adapter | ✅ | Claude | 完成 |
| P1.7 | Domain Registry | ✅ | Claude | 完成 |
| P1.8 | User Domain 测试 | ✅ | Claude | 6 个测试用例通过 |
| P1.9 | 集成验证 | ✅ | Claude | 编译通过，测试通过 |
| P2.1 | Auth Model | ✅ | Claude | RefreshToken, UserAPIKey, SystemAPIKey, TokenPair |
| P2.2 | Auth Outbound Port | ✅ | Claude | RefreshTokenDB, UserAPIKeyDB, SystemAPIKeyDB, OAuth, JWT, Crypto |
| P2.3 | Auth Inbound Port | ✅ | Claude | AuthHttpPort, APIKeyHttpPort, SystemAPIKeyHttpPort |
| P2.4 | Auth Domain | ✅ | Claude | OAuth登录, Token刷新, API Key管理 |
| P2.5 | Auth Postgres Adapter | ✅ | Claude | refresh_token, user_api_key, system_api_key |
| P2.6 | Auth Redis Adapter | ✅ | Claude | oauth_state, rate_limiter |
| P2.7 | OAuth Adapter | ✅ | Claude | GitHub, Google providers, Registry |
| P2.8 | Auth HTTP Adapter | ✅ | Claude | auth, api_key, system_api_key handlers |
| P2.9 | Domain Registry | ✅ | Claude | 添加 Auth Domain 到注册表 |
| P2.10 | Auth Domain 测试 | ✅ | Claude | 15 个测试用例全部通过 |
| P2.11 | 集成验证 | ✅ | Claude | 编译通过，测试通过，依赖方向正确 |
| P3.1 | Billing Model | ✅ | Claude | Plan, Subscription, UsageRecord, Quota, Balance, Credits |
| P3.2 | Billing Outbound Port | ✅ | Claude | PlanDB, SubscriptionDB, UsageDB, QuotaCache |
| P3.3 | Billing Inbound Port | ✅ | Claude | BillingHttpPort, UsageHttpPort, CreditsHttpPort |
| P3.4 | Billing Domain | ✅ | Claude | 订阅管理, 用量记录, 配额检查, 余额管理 |
| P3.5 | Billing Postgres Adapter | ✅ | Claude | plan, subscription, usage_record |
| P3.6 | Billing Redis Adapter | ✅ | Claude | quota_cache |
| P3.7 | Billing HTTP Adapter | ✅ | Claude | billing, usage, credits handlers |
| P3.8 | Billing Domain 测试 | ✅ | Claude | 覆盖率 41.3% |
| P3.9 | 集成验证 | ✅ | Claude | 编译通过，测试通过 |
| P4.1 | Order Model | ✅ | Claude | Order, OrderItem, Invoice, OrderStatus |
| P4.2 | Order Outbound Port | ✅ | Claude | OrderDB, OrderItemDB, InvoiceDB |
| P4.3 | Order Inbound Port | ✅ | Claude | OrderHttpPort, InvoiceHttpPort |
| P4.4 | Order Domain | ✅ | Claude | 订单创建, 状态流转, 发票管理 |
| P4.5 | Order Postgres Adapter | ✅ | Claude | order, order_item, invoice |
| P4.6 | Order HTTP Adapter | ✅ | Claude | order, invoice handlers |
| P4.7 | Order Domain 测试 | ✅ | Claude | 覆盖率 57.3% |
| P4.8 | 集成验证 | ✅ | Claude | 编译通过，测试通过 |
| P5.1 | Payment Model | ✅ | Claude | Payment, PaymentStatus, PaymentMethod, WebhookEvent |
| P5.2 | Payment Outbound Port | ✅ | Claude | PaymentDB, WebhookEventDB, ProviderRegistry, OrderReader, BillingReader |
| P5.3 | Payment Inbound Port | ✅ | Claude | PaymentHttpPort, RefundHttpPort, WebhookHttpPort |
| P5.4 | Payment Domain | ✅ | Claude | Stripe支付, 原生支付(支付宝/微信), 退款, Webhook处理 |
| P5.5 | Payment Postgres Adapter | ✅ | Claude | payment, webhook_event |
| P5.6 | Payment HTTP Adapter | ✅ | Claude | payment, refund, webhook handlers |
| P5.7 | Domain Registry | ✅ | Claude | 添加 Payment Domain 到注册表 |
| P5.8 | Payment Domain 测试 | ✅ | Claude | 17 个测试用例通过，覆盖率 29.0% |
| P5.9 | 集成验证 | ✅ | Claude | 编译通过，测试通过，依赖方向正确 |
| P6.1 | AI Model | ⬜ | | Provider, Model, Account, Group |
| P6.2 | AI Outbound Port | ⬜ | | ProviderDB, ModelDB, AccountDB, HealthCache, VendorAdapter |
| P6.3 | AI Inbound Port | ⬜ | | ChatHttp, EmbeddingHttp, AdminHttp, PoolHttp |
| P6.4 | AI Domain | ⬜ | | Chat, Embed, Route, HealthMonitor |
| P6.5 | 迁移路由策略 | ⬜ | | 6个策略: UserPref, Health, Capability, Context, Cost, LoadBalance |
| P6.6 | AI Postgres Adapter | ⬜ | | provider, model, account, group |
| P6.7 | AI Redis Adapter | ⬜ | | health_cache, embedding_cache |
| P6.8 | 迁移厂商适配器 | ⬜ | | OpenAI, Anthropic, Generic |
| P6.9 | AI HTTP Adapter | ⬜ | | chat, embedding, admin, pool handlers |
| P6.10 | AI Domain Registry | ⬜ | | 添加 AI Domain |
| P6.11 | AI Domain 测试 | ⬜ | | 目标覆盖率 >80% |
| P6.12 | AI 集成验证 | ⬜ | | 编译通过，测试通过 |
| P7.1 | Git Model | ⬜ | | GitRepo, Collaborator, PullRequest, LFSObject, LFSLock |
| P7.2 | Git Outbound Port | ⬜ | | RepoDB, CollabDB, PRDB, LFSDB, Storage, Authenticator |
| P7.3 | Git Inbound Port | ⬜ | | RepoHttp, CollabHttp, PRHttp, ProtocolHttp, LFSHttp |
| P7.4 | Git Domain | ⬜ | | 仓库管理, 访问控制, PR, 存储统计 |
| P7.5 | 迁移 Git 协议 | ⬜ | | Smart HTTP: InfoRefs, UploadPack, ReceivePack |
| P7.6 | LFS Domain | ⬜ | | Batch, Lock, Verify |
| P7.7 | Git Postgres Adapter | ⬜ | | repo, collaborator, pull_request, lfs |
| P7.8 | 迁移 R2 存储适配器 | ⬜ | | R2Client, BillyFilesystem |
| P7.9 | Git REST HTTP Adapter | ⬜ | | repo, collab, pr handlers |
| P7.10 | Git Protocol HTTP Adapter | ⬜ | | info_refs, upload_pack, receive_pack |
| P7.11 | Git LFS HTTP Adapter | ⬜ | | batch, lock, verify handlers |
| P7.12 | Git Domain Registry | ⬜ | | 添加 Git Domain |
| P7.13 | Git Domain 测试 | ⬜ | | 目标覆盖率 >80% |
| P7.14 | Git 集成验证 | ⬜ | | 编译通过，测试通过，Git协议兼容 |
| P8.1 | Media Model | ⬜ | | Provider, Model, Task |
| P8.2 | Media Outbound Port | ⬜ | | ProviderRegistry, HealthChecker, TaskManager, VendorAdapter |
| P8.3 | Media Inbound Port | ⬜ | | ImageHttp, VideoHttp, TaskHttp |
| P8.4 | Media Domain | ⬜ | | GenerateImage, GenerateVideo, GetStatus |
| P8.5 | 迁移 Media 适配器 | ⬜ | | OpenAI media adapter |
| P8.6 | Media HTTP Adapter | ⬜ | | image, video, task handlers |
| P8.7 | Media Domain Registry | ⬜ | | 添加 Media Domain |
| P8.8 | Media Domain 测试 | ⬜ | | 目标覆盖率 >80% |
| P8.9 | Media 集成验证 | ⬜ | | 编译通过，测试通过 |
| P9.1 | Collaboration Model | ⬜ | | Team, TeamMember, TeamInvitation |
| P9.2 | Collaboration Outbound Port | ⬜ | | TeamDB, MemberDB, InvitationDB, UserReader |
| P9.3 | Collaboration Inbound Port | ⬜ | | TeamHttp, MemberHttp, InvitationHttp |
| P9.4 | Collaboration Domain | ⬜ | | 团队管理, 成员管理, 邀请流程 |
| P9.5 | 迁移角色权限 | ⬜ | | Role, Permission 定义 |
| P9.6 | Collaboration Postgres Adapter | ⬜ | | team, member, invitation |
| P9.7 | Collaboration HTTP Adapter | ⬜ | | team, member, invitation handlers |
| P9.8 | Collaboration Domain Registry | ⬜ | | 添加 Collaboration Domain |
| P9.9 | Collaboration Domain 测试 | ⬜ | | 目标覆盖率 >80% |
| P9.10 | Collaboration 集成验证 | ⬜ | | 编译通过，测试通过 |
| P10.1 | 删除旧 module 代码 | ⬜ | | 删除 internal/module/ |
| P10.2 | 更新 shared 目录 | ⬜ | | 迁移到对应 adapter |
| P10.3 | 更新 app.go | ⬜ | | 重写应用组装层 |
| P10.4 | 更新文档 | ⬜ | | README, CLAUDE.md, API文档 |

---

## 风险与缓解

| 风险 | 影响 | 缓解措施 |
|------|------|---------|
| 循环依赖 | 编译失败 | 严格遵循依赖方向，使用接口隔离 |
| 测试覆盖不足 | 迁移后 Bug | 迁移前补充测试，迁移后验证 |
| 业务中断 | 用户影响 | 渐进式迁移，保持旧代码可用 |
| AI 模块复杂 | 迁移困难 | 保留核心逻辑，仅重组结构 |

---

## 验收标准

1. **编译通过**: `go build ./...` 无错误 ✅
2. **测试通过**: `go test ./...` 全部通过 ✅
3. **覆盖率达标**: Domain 层 > 80%，Adapter 层 > 60% ⚠️ (见下表)
4. **依赖正确**: Domain 不依赖 Adapter ✅
5. **文档完整**: 所有新接口有注释 ⬜

---

## 当前覆盖率统计

| Domain | 覆盖率 | 状态 | 备注 |
|--------|--------|------|------|
| user | 80.4% | ✅ | 达标 |
| auth | 80.5% | ✅ | 达标 |
| billing | 81.0% | ✅ | 达标 |
| order | 88.5% | ✅ | 达标（从 78.3% 提升） |
| payment | 83.4% | ✅ | 达标（从 59.9% 提升） |

**所有 Domain 模块测试覆盖率已达标 (>80%)**

---

## 迁移进度汇总

```
Phase 0: 准备工作        [███████████████████████] 100%
Phase 1: User 迁移       [███████████████████████] 100%
Phase 2: Auth 迁移       [███████████████████████] 100%
Phase 3: Billing 迁移    [███████████████████████] 100%
Phase 4: Order 迁移      [███████████████████████] 100%
Phase 5: Payment 迁移    [███████████████████████] 100%
Phase 6: AI 迁移         [░░░░░░░░░░░░░░░░░░░░░░░]   0%
Phase 7: Git 迁移        [░░░░░░░░░░░░░░░░░░░░░░░]   0%
Phase 8: Media 迁移      [░░░░░░░░░░░░░░░░░░░░░░░]   0%
Phase 9: Collaboration   [░░░░░░░░░░░░░░░░░░░░░░░]   0%
Phase 10: 清理优化       [░░░░░░░░░░░░░░░░░░░░░░░]   0%

总体进度: 5/10 phases (50%)
```

**下一步**: Phase 6 AI 迁移（复杂模块，包含 routing/provider/pool 等子模块）

**详细方案**: 见 [Phase 6-10 详细迁移方案](./phase6-10-migration-plan.md)
