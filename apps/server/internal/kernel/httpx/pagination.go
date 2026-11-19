package httpx

import (
	"net/http"
	"strconv"
)

const (
	DefaultPageSize = 20
	MaxPageSize     = 100
)

type PageQuery struct {
	Page     int
	PageSize int
}

func (p PageQuery) Limit() int {
	return p.PageSize
}

func (p PageQuery) Offset() int {
	return (p.Page - 1) * p.PageSize
}

type PageResult[T any] struct {
	Items    []T   `json:"items"`
	Total    int64 `json:"total"`
	Page     int   `json:"page"`
	PageSize int   `json:"page_size"`
}

func ParsePageQuery(r *http.Request) PageQuery {
	q := r.URL.Query()
	page, _ := strconv.Atoi(q.Get("page"))
	if page < 1 {
		page = 1
	}
	size, _ := strconv.Atoi(q.Get("page_size"))
	if size < 1 {
		size = DefaultPageSize
	}
	if size > MaxPageSize {
		size = MaxPageSize
	}
	return PageQuery{Page: page, PageSize: size}
}
