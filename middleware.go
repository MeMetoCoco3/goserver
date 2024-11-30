package main

import (
	"fmt"
	"net/http"
)

func middlewareLog(next interface{}) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Printf("Log: %s %s %s\n", r.Method, r.URL.Path, r.Host)

		switch h := next.(type) {
		case http.Handler:
			h.ServeHTTP(w, r)
		case http.HandlerFunc:
			h(w, r)
		case func(http.ResponseWriter, *http.Request):
			h(w, r)
		default:
			fmt.Printf("Unsupported handler type")
		}
	})
}
