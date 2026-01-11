package pagination

// Pagination represents pagination parameters.
type Pagination struct {
	Page     int `form:"page" binding:"min=1"`
	PageSize int `form:"page_size" binding:"min=1,max=100"`
}

// Default values.
const (
	DefaultPage     = 1
	DefaultPageSize = 20
	MaxPageSize     = 100
)

// New creates pagination with default values.
func New() *Pagination {
	return &Pagination{
		Page:     DefaultPage,
		PageSize: DefaultPageSize,
	}
}

// NewWithSize creates pagination with a specific page size.
func NewWithSize(pageSize int) *Pagination {
	if pageSize < 1 {
		pageSize = DefaultPageSize
	}
	if pageSize > MaxPageSize {
		pageSize = MaxPageSize
	}
	return &Pagination{
		Page:     DefaultPage,
		PageSize: pageSize,
	}
}

// Offset returns the offset for database queries.
func (p *Pagination) Offset() int {
	if p.Page < 1 {
		p.Page = DefaultPage
	}
	return (p.Page - 1) * p.PageSize
}

// Limit returns the limit for database queries.
func (p *Pagination) Limit() int {
	if p.PageSize < 1 {
		return DefaultPageSize
	}
	if p.PageSize > MaxPageSize {
		return MaxPageSize
	}
	return p.PageSize
}

// TotalPages calculates the total number of pages.
func (p *Pagination) TotalPages(total int64) int {
	if total == 0 {
		return 0
	}
	pageSize := p.Limit()
	pages := int(total) / pageSize
	if int(total)%pageSize > 0 {
		pages++
	}
	return pages
}

// PageInfo represents pagination info in API responses.
type PageInfo struct {
	Page       int   `json:"page"`
	PageSize   int   `json:"page_size"`
	Total      int64 `json:"total"`
	TotalPages int   `json:"total_pages"`
}

// Info returns pagination info for API responses.
func (p *Pagination) Info(total int64) PageInfo {
	return PageInfo{
		Page:       p.Page,
		PageSize:   p.Limit(),
		Total:      total,
		TotalPages: p.TotalPages(total),
	}
}
