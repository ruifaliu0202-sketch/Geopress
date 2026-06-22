package service

const (
	DefaultPageSize = 20
	MaxPageSize     = 100
)

type PageRequest struct {
	Page     int
	PageSize int
}

func (req PageRequest) Normalize() PageRequest {
	if req.Page < 1 {
		req.Page = 1
	}
	if req.PageSize < 1 {
		req.PageSize = DefaultPageSize
	}
	if req.PageSize > MaxPageSize {
		req.PageSize = MaxPageSize
	}
	return req
}

func (req PageRequest) Limit() int {
	return req.Normalize().PageSize
}

func (req PageRequest) Offset() int {
	normalized := req.Normalize()
	return (normalized.Page - 1) * normalized.PageSize
}

type ListResponse[T any] struct {
	Items    []T `json:"items"`
	Total    int `json:"total"`
	Page     int `json:"page"`
	PageSize int `json:"pageSize"`
}

func NewListResponse[T any](items []T, total int, req PageRequest) ListResponse[T] {
	normalized := req.Normalize()
	if items == nil {
		items = []T{}
	}
	return ListResponse[T]{
		Items:    items,
		Total:    total,
		Page:     normalized.Page,
		PageSize: normalized.PageSize,
	}
}
