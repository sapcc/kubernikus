package handlers

import (
	"net/http"
	"strings"
)

var RootContent = `{
  "paths": [
    "/api",
    "/api/v1",
    "/docs",
    "/info",
    "/static",
		"/swagger"
  ]
}`

func RootHandler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/" {
			for _, accept := range strings.Split(r.Header.Get("Accept"), ",") {
				switch strings.TrimSpace(accept) {
				case "text/html":
					http.Redirect(rw, r, "/docs", 301)
					return
				}
			}
			rw.Write([]byte(RootContent))
		} else {
			next.ServeHTTP(rw, r)
		}
	})
}
