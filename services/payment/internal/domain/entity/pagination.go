package entity

// PaginationParams represents pagination request parameters
type PaginationParams struct {
	Page  int `json:"page" query:"page"`
	Limit int `json:"limit" query:"limit"`
}

// PaginationMeta represents pagination metadata in responses
type PaginationMeta struct {
	CurrentPage int   `json:"current_page"`
	PerPage     int   `json:"per_page"`
	Total       int64 `json:"total"`
	TotalPages  int   `json:"total_pages"`
}

// PaginatedPaymentsResponse represents paginated payment response
type PaginatedPaymentsResponse struct {
	Data       []*Payment     `json:"data"`
	Pagination PaginationMeta `json:"pagination"`
}

// Pagination constants
const (
	DefaultPageSize = 20
	MaxPageSize     = 100
	MinPageSize     = 1
	DefaultPage     = 1
)

// Validate validates and normalizes pagination parameters
func (p *PaginationParams) Validate() {
	if p.Page < 1 {
		p.Page = DefaultPage
	}
	
	if p.Limit < MinPageSize {
		p.Limit = DefaultPageSize
	} else if p.Limit > MaxPageSize {
		p.Limit = MaxPageSize
	}
}

// CalculateOffset calculates the database offset from page and limit
func (p *PaginationParams) CalculateOffset() int {
	return (p.Page - 1) * p.Limit
}

// NewPaginationMeta creates pagination metadata from parameters and total count
func NewPaginationMeta(page, limit int, total int64) PaginationMeta {
	totalPages := int(total) / limit
	if int(total)%limit > 0 {
		totalPages++
	}
	
	return PaginationMeta{
		CurrentPage: page,
		PerPage:     limit,
		Total:       total,
		TotalPages:  totalPages,
	}
}