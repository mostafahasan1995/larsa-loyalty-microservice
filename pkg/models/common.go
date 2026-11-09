package models

type Pagination struct {
	TotalPages float64 `json:"totalPages"`
	PerPage    int64   `json:"perPage"`
	TotalCount int64   `json:"totalCount"`
}
