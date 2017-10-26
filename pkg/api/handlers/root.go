package handlers

import "net/http"

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
			rw.Write([]byte(RootContent))
		} else {
			next.ServeHTTP(rw, r)
		}
	})
}
