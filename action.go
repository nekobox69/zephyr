// Package zephyr Create at 2021-01-19 14:41
package zephyr

import (
	"net/http"

	"github.com/sirupsen/logrus"
)

// Action request mapping
type Action struct {
	URL             string                                       // 请求的URL，支持正则表达式
	Method          []string                                     // 支持的请求方法
	Filter          []Filter                                     // 过滤器
	Handler         func(w http.ResponseWriter, r *http.Request) // 处理请求的方法
	PreHandler      preHandlerFunc
	AfterCompletion afterCompletionFunc
	Wrapper         wrapperFuc
}

func (a *Action) fit(w http.ResponseWriter, r *http.Request, logger *logrus.Logger, resp *Resp) bool {
	if nil != a.Filter && len(a.Filter) > 0 {
		req, err := readRequest(r)
		if nil != err {
			logger.Error(err.Error())
			return false
		}
		for _, f := range a.Filter {
			b, err := f.Fit.DoFilter(req, f.Trials)
			if nil != err {
				logger.Error(err)
				if nil != f.Err {
					f.Err(&w)
				} else {
					if nil != resp {
						w.WriteHeader((*resp).Code)
						w.Write((*resp).Msg)
					} else {
						w.WriteHeader(http.StatusBadRequest)
					}
				}
				return false
			}
			if !b {
				if nil != f.Err {
					f.Err(&w)
				} else {
					if nil != resp {
						w.WriteHeader((*resp).Code)
						w.Write((*resp).Msg)
					} else {
						w.WriteHeader(http.StatusBadRequest)
					}
				}
				return false
			}
		}
	}
	return true
}
