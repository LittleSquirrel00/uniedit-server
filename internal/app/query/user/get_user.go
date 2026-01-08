package user

import (
	"context"

	"github.com/google/uuid"
	"github.com/uniedit/server/internal/domain/user"
)

// GetUserQuery represents a query to get a user by ID.
type GetUserQuery struct {
	UserID uuid.UUID
}

// GetUserResult is the result of getting a user.
type GetUserResult struct {
	User *user.User
}

// GetUserHandler handles GetUserQuery.
type GetUserHandler struct {
	repo user.Repository
}

// NewGetUserHandler creates a new handler.
func NewGetUserHandler(repo user.Repository) *GetUserHandler {
	return &GetUserHandler{repo: repo}
}

// Handle executes the query.
func (h *GetUserHandler) Handle(ctx context.Context, query GetUserQuery) (*GetUserResult, error) {
	u, err := h.repo.GetByID(ctx, query.UserID)
	if err != nil {
		return nil, err
	}
	return &GetUserResult{User: u}, nil
}

// GetUserByEmailQuery represents a query to get a user by email.
type GetUserByEmailQuery struct {
	Email string
}

// GetUserByEmailResult is the result of getting a user by email.
type GetUserByEmailResult struct {
	User *user.User
}

// GetUserByEmailHandler handles GetUserByEmailQuery.
type GetUserByEmailHandler struct {
	repo user.Repository
}

// NewGetUserByEmailHandler creates a new handler.
func NewGetUserByEmailHandler(repo user.Repository) *GetUserByEmailHandler {
	return &GetUserByEmailHandler{repo: repo}
}

// Handle executes the query.
func (h *GetUserByEmailHandler) Handle(ctx context.Context, query GetUserByEmailQuery) (*GetUserByEmailResult, error) {
	u, err := h.repo.GetByEmail(ctx, query.Email)
	if err != nil {
		return nil, err
	}
	return &GetUserByEmailResult{User: u}, nil
}

// ListUsersQuery represents a query to list users.
type ListUsersQuery struct {
	Status   *user.UserStatus
	Email    *string
	IsAdmin  *bool
	Page     int
	PageSize int
}

// ListUsersResult is the result of listing users.
type ListUsersResult struct {
	Users []*user.User
	Total int64
}

// ListUsersHandler handles ListUsersQuery.
type ListUsersHandler struct {
	repo user.Repository
}

// NewListUsersHandler creates a new handler.
func NewListUsersHandler(repo user.Repository) *ListUsersHandler {
	return &ListUsersHandler{repo: repo}
}

// Handle executes the query.
func (h *ListUsersHandler) Handle(ctx context.Context, query ListUsersQuery) (*ListUsersResult, error) {
	filter := &user.UserFilter{
		Status:  query.Status,
		Email:   query.Email,
		IsAdmin: query.IsAdmin,
	}

	pagination := user.NewPagination(query.Page, query.PageSize)

	users, total, err := h.repo.List(ctx, filter, pagination)
	if err != nil {
		return nil, err
	}

	return &ListUsersResult{Users: users, Total: total}, nil
}
