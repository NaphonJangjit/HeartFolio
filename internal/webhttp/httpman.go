package webhttp

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

type Middleware func(http.Handler) http.Handler

type Router struct {
	mux         *http.ServeMux
	prefix      string
	parent      *Router
	middlewares []Middleware
}

func New() *Router {
	return &Router{
		mux:    http.NewServeMux(),
		prefix: "",
		parent: nil,
	}
}

func (r *Router) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	r.mux.ServeHTTP(w, req)
}

func (r *Router) Use(mw ...Middleware) *Router {
	r.middlewares = append(r.middlewares, mw...)
	return r
}

func (r *Router) Group(prefix string) *Router {
	return &Router{
		mux:         r.mux,
		prefix:      r.prefix + prefix,
		parent:      r,
		middlewares: []Middleware{},
	}
}

func (r *Router) Mount(prefix string, handler http.Handler) {
	fullPrefix := r.prefix + prefix
	if fullPrefix != "" && !strings.HasSuffix(fullPrefix, "/") {
		fullPrefix = fullPrefix + "/"
	}
	wrapped := http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if !strings.HasPrefix(req.URL.Path, fullPrefix) {
			http.NotFound(w, req)
			return
		}
		strippedPath := strings.TrimPrefix(req.URL.Path, fullPrefix)
		if strippedPath == "" || strippedPath[0] != '/' {
			strippedPath = "/" + strippedPath
		}
		r2 := new(http.Request)
		*r2 = *req
		r2.URL = new(url.URL)
		*r2.URL = *req.URL
		r2.URL.Path = strippedPath
		handler.ServeHTTP(w, r2)
	})
	final := r.wrapHandler(wrapped)
	r.mux.Handle(fullPrefix, final)
}

func (r *Router) Register(prefix string, handler http.Handler) {
	r.Mount(prefix, handler)
}

func (r *Router) Handle(method, path string, handler http.Handler) {
	fullPath := r.prefix + path
	pattern := method + " " + fullPath
	wrapped := r.wrapHandler(handler)
	r.mux.Handle(pattern, wrapped)
}

func (r *Router) HandleFunc(method, path string, handler http.HandlerFunc) {
	r.Handle(method, path, handler)
}

func (r *Router) Get(path string, handler http.HandlerFunc) {
	r.Handle(http.MethodGet, path, handler)
}

func (r *Router) Post(path string, handler http.HandlerFunc) {
	r.Handle(http.MethodPost, path, handler)
}

func (r *Router) Put(path string, handler http.HandlerFunc) {
	r.Handle(http.MethodPut, path, handler)
}

func (r *Router) Delete(path string, handler http.HandlerFunc) {
	r.Handle(http.MethodDelete, path, handler)
}

func (r *Router) Patch(path string, handler http.HandlerFunc) {
	r.Handle(http.MethodPatch, path, handler)
}

func (r *Router) Options(path string, handler http.HandlerFunc) {
	r.Handle(http.MethodOptions, path, handler)
}

func (r *Router) Head(path string, handler http.HandlerFunc) {
	r.Handle(http.MethodHead, path, handler)
}

func (r *Router) wrapHandler(handler http.Handler) http.Handler {
	var chain []Middleware
	for cur := r; cur != nil; cur = cur.parent {
		chain = append(cur.middlewares, chain...)
	}
	result := handler
	for i := len(chain) - 1; i >= 0; i-- {
		result = chain[i](result)
	}
	return result
}

func JSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := encodeJSON(w, data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func encodeJSON(w http.ResponseWriter, data interface{}) error {
	return json.NewEncoder(w).Encode(data)
}

func ReadJSON(r *http.Request, data interface{}) error {
	defer r.Body.Close()
	return json.NewDecoder(r.Body).Decode(data)
}

func ErrorJSON(w http.ResponseWriter, status int, message string) {
	JSON(w, status, map[string]string{"error": message})
}

func Param(r *http.Request, name string) string {
	return r.PathValue(name)
}

func ParamInt(r *http.Request, name string) (int, error) {
	val := Param(r, name)
	if val == "" {
		return 0, fmt.Errorf("parameter %s is missing", name)
	}
	return strconv.Atoi(val)
}