package repository

const (
	DefaultLimit = 20
	MaxLimit     = 100
)

type Page struct {
	Limit  int
	Offset int
}

func NewPage(page, pageSize int) Page {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = DefaultLimit
	}
	if pageSize > MaxLimit {
		pageSize = MaxLimit
	}
	return Page{
		Limit:  pageSize,
		Offset: (page - 1) * pageSize,
	}
}
