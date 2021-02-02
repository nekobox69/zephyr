// Package zephyr Create at 2021-01-19 14:04
package zephyr

import (
	"net/http"
	"net/url"
)

// TrialItem 预处理选项
type TrialItem struct {
	Condition map[string]string
	Rules     []string
}

// Request req data
type Request struct {
	Query  url.Values
	Body   []byte
	Header http.Header
	Method string
}

// Filter 过滤器.
type Filter struct {
	Trials TrialItem
	Fit    Fit
	Err    func(w *http.ResponseWriter)
}

// Fit doFilter
type Fit interface {
	DoFilter(request *Request, prerequisite TrialItem) (bool, error)
}
