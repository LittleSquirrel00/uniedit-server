package model

// PaginationRequest defines pagination parameters.
type PaginationRequest struct {
	Page     int `json:"page" form:"page"`
	PageSize int `json:"page_size" form:"page_size"`
}

// DefaultPagination applies default pagination values.
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

// SortRequest defines sorting parameters.
type SortRequest struct {
	SortBy    string `json:"sort_by" form:"sort_by"`
	SortOrder string `json:"sort_order" form:"sort_order"` // asc or desc
}

// DefaultSort applies default sorting values.
func (s *SortRequest) DefaultSort(defaultField string) {
	if s.SortBy == "" {
		s.SortBy = defaultField
	}
	if s.SortOrder == "" {
		s.SortOrder = "desc"
	}
}

// IsAsc returns true if sort order is ascending.
func (s *SortRequest) IsAsc() bool {
	return s.SortOrder == "asc"
}
