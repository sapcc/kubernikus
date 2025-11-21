package handlers

import (
	"net/http"
	"strings"
)

var RootContent = `{
  "paths": [
    "/api",
    "/api/v1",
    "/auth/login",
    "/docs",
    "/swagger"
  ]
}`

func RootHandler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/" {
			for _, accept := range strings.Split(r.Header.Get("Accept"), ",") {
				switch strings.TrimSpace(accept) {
				case "text/html":
					http.Redirect(rw, r, "/docs", http.StatusMovedPermanently)
					return
				}
			}
			rw.Write([]byte(RootContent))
		} else {
			next.ServeHTTP(rw, r)
		}
	})
}
