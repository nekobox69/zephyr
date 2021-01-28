// Package zephyr Create at 2021-01-19 13:48
package zephyr

import (
	"bytes"
	"errors"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"net/http/httputil"
	"regexp"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
)

type afterCompletionFunc func(w *http.ResponseWriter, r *http.Request, cost int64)
type preHandlerFunc func(w *http.ResponseWriter, r *http.Request) bool
type wrapperFuc func(w *http.ResponseWriter) http.ResponseWriter

// Zephyr web framework
type Zephyr struct {
	cros            bool     // 是否支持跨域
	filter          []Filter // 过滤器
	handlers        []Action
	logger          *logrus.Logger
	profile         *Profile
	resp            *Resp
	preHandler      preHandlerFunc
	afterCompletion afterCompletionFunc
	wrapper         wrapperFuc
}

func NewZephyr(logger *logrus.Logger) *Zephyr {
	z := Zephyr{logger: logger}
	if nil == z.logger {
		z.logger = logrus.StandardLogger()
		z.logger.SetFormatter(&logrus.JSONFormatter{})
	}
	return &z
}

// SetCros enable or disable cros
func (z *Zephyr) SetCros(cros bool) {
	z.cros = cros
}

func (z *Zephyr) SetErrResp(resp Resp) {
	z.resp = &resp
}

func (z *Zephyr) AddFilter(filter Filter) *Zephyr {
	z.filter = append(z.filter, filter)
	return z
}

func (z *Zephyr) AddHandler(r Action) {
	z.handlers = append(z.handlers, r)
}

func (z *Zephyr) SetPreHandler(preHandler preHandlerFunc) {
	z.preHandler = preHandler
}

func (z *Zephyr) SetAfterCompletion(afterCompletion afterCompletionFunc) {
	z.afterCompletion = afterCompletion
}

func (z *Zephyr) SetWrapper(wrapper wrapperFuc) {
	z.wrapper = wrapper
}

func (z *Zephyr)SetProfile(profile *Profile)  {
	z.profile=profile
}

func (z *Zephyr) GetLogger() *logrus.Logger {
	return z.logger
}

type Resp struct {
	Code int
	Msg  []byte
}

// ServeHTTP is the HTTP Entry point for a Martini instance. Useful if you want to control your own HTTP server.
func (z *Zephyr) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	startTime := unixMillisecond()
	ip := ""
	if "" == r.Header.Get("X-Real-IP") {
		ip, _, _ = net.SplitHostPort(r.RemoteAddr)
		r.Header.Set("X-Real-IP", ip)
	}
	if z.cros {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Headers", "*")
		w.Header().Set("Access-Control-Allow-Methods", "*")
	}
	found := false
	match := false
	trialFailed := false
	for _, v := range z.handlers {
		methods := strings.Join(v.Method, ",")
		if ok, _ := regexp.MatchString("^"+v.URL+"$", r.URL.Path); ok {
			found = true
			if strings.Contains(methods, r.Method) {
				match = true
				if nil != z.profile && r.URL.Path == z.profile.URL {
					z.profile.Handler(w, r)
					goto End
				}

				// 执行全局校验
				b := z.globalFilter(w, r)
				if !b {
					trialFailed = true
					goto End
				}

				// 局部校验
				b = v.fit(w, r, z.logger, z.resp)
				if !b {
					trialFailed = true
					goto End
				}

				if nil != z.preHandler && !z.preHandler(&w, r) {
					if nil != z.resp {
						w.WriteHeader((*z.resp).Code)
						w.Write((*z.resp).Msg)
					} else {
						w.WriteHeader(http.StatusBadRequest)
					}
					goto End
				}

				if nil != v.PreHandler && !v.PreHandler(&w, r) {
					if nil != z.resp {
						w.WriteHeader((*z.resp).Code)
						w.Write((*z.resp).Msg)
					} else {
						w.WriteHeader(http.StatusBadRequest)
					}
					goto End
				}
				res := w
				if nil != v.Wrapper {
					res = v.Wrapper(&w)
				} else if nil != z.wrapper {
					res = z.wrapper(&w)
				}
				v.Handler(res, r)
				break
			}
		}
	}
End:
	endTime := unixMillisecond()
	if nil == z.afterCompletion {
		z.logger.WithFields(logrus.Fields{
			"ip":     ip,
			"url":    r.URL.Path,
			"method": r.Method,
			"cost":   fmt.Sprintf("%d ms", endTime-startTime),
		}).Info("OK")
	} else {
		z.afterCompletion(&w, r, endTime-startTime)
	}

	if trialFailed {
		return
	}
	if found && !match {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	if !found {
		w.WriteHeader(http.StatusNotFound)
		return
	}
}

func (z *Zephyr) globalFilter(w http.ResponseWriter, r *http.Request) bool {
	if nil != z.filter && len(z.filter) > 0 {
		req, err := readRequest(r)
		if nil != err {
			z.logger.Error(err.Error())
			return false
		}
		for _, f := range z.filter {
			b, err := f.Fit.DoFilter(req, f.Trials)
			if nil != err {
				z.logger.Error(err)
				if nil != f.Err {
					f.Err(&w)
				} else {
					if nil != z.resp {
						w.WriteHeader((*z.resp).Code)
						w.Write((*z.resp).Msg)
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
					if nil != z.resp {
						w.WriteHeader((*z.resp).Code)
						w.Write((*z.resp).Msg)
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

// readRequest 获取request
func readRequest(r *http.Request) (*Request, error) {
	switch r.Method {
	case http.MethodGet:
		return &Request{Header: r.Header, Query: r.URL.Query(), Method: r.Method}, nil
	case http.MethodDelete:
		fallthrough
	case http.MethodPost:
		fallthrough
	case http.MethodPut:
		exp, err := regexp.Compile("([a-zA-z/\\-_\\.]+)(;*(.+))*")
		if nil != err {
			return nil, err
		}
		param := exp.FindStringSubmatch(r.Header.Get("Content-Type"))
		if nil == param || len(param) < 2 {
			return nil, errors.New("not found content-type")
		}
		switch param[1] {
		case "multipart/form-data":
			body, err := httputil.DumpRequest(r, true)
			if nil != err {
				return nil, err
			}
			copy, err := http.NewRequest("POST", "", bytes.NewReader(body))
			if nil != err {
				return nil, err
			}
			copy.Header = r.Header
			err = copy.ParseMultipartForm(32 << 20)
			if nil != err {
				copy.Body.Close()
				return nil, err
			}
			copy.Body.Close()
			return &Request{Header: r.Header, Query: r.PostForm, Method: r.Method}, nil
		default:
			body, err := ioutil.ReadAll(r.Body)
			if nil != err {
				return nil, err
			}
			r.Body.Close()
			r.Body = ioutil.NopCloser(bytes.NewBuffer(body))
			return &Request{Header: r.Header, Query: r.URL.Query(), Method: r.Method, Body: body}, nil
		}
	}
	return nil, errors.New("method not supported")
}

// unixMillisecond unix time millisecond
func unixMillisecond() int64 {
	return time.Now().UnixNano() / 1e6
}
