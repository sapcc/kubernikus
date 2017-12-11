package log

import (
	"bufio"
	"context"
	"fmt"
	"net"
	"net/http"
	"strings"
	"time"

	kitlog "github.com/go-kit/kit/log"
	uuid "github.com/satori/go.uuid"
)

type key int

const (
	KubernikusRequestID key = 0
)

func RequestIDHandler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(rw http.ResponseWriter, request *http.Request) {
		if id := request.Context().Value(KubernikusRequestID); id == nil {
			request = request.WithContext(context.WithValue(request.Context(), KubernikusRequestID, uuid.NewV4()))
		}
		next.ServeHTTP(rw, request)
	})
}

func LoggingHandler(logger kitlog.Logger, next http.Handler) http.Handler {
	ingress_logger := kitlog.With(logger, "api", "ingress")
	return http.HandlerFunc(func(rw http.ResponseWriter, request *http.Request) {
		wrapper := makeWrapper(rw)

		id := ""
		if reqId := request.Context().Value(KubernikusRequestID); reqId != nil {
			id = fmt.Sprintf("%s", reqId)
			logger = kitlog.With(logger, "id", id)
		}
		request = request.WithContext(context.WithValue(request.Context(), "logger", logger))

		defer func(begin time.Time) {
			var keyvals = make([]interface{}, 0, 4)

			keyvals = append(keyvals,
				"status", wrapper.Status(),
				"size", wrapper.Size(),
				"took", time.Since(begin),
			)

			if id != "" {
				keyvals = append(keyvals, "id", id)
			}

			log(ingress_logger, request, keyvals...)
		}(time.Now())

		next.ServeHTTP(wrapper, request)
	})
}

func log(logger kitlog.Logger, request *http.Request, extra ...interface{}) {
	var keyvals []interface{}

	source_ip, _, err := net.SplitHostPort(request.RemoteAddr)
	if err != nil {
		source_ip = request.RemoteAddr
	}

	if source_ip != "" {
		keyvals = append(keyvals, "source_ip", source_ip)
	}

	keyvals = append(keyvals, "method", request.Method)

	host, host_port, err := net.SplitHostPort(request.Host)
	if err == nil {
		if host != "" {
			keyvals = append(keyvals,
				"host", host)
		}
		if host_port != "" {
			keyvals = append(keyvals,
				"port", host_port)
		}
	}

	keyvals = append(keyvals, "path", request.URL.EscapedPath())

	for i, k := range request.URL.Query() {
		keyvals = append(keyvals, i, strings.Join(k, ","))
	}

	keyvals = append(keyvals, "user_agent", request.UserAgent())
	keyvals = append(keyvals, extra...)
	logger.Log(keyvals...)
}

// this stuff is copied from gorilla

func makeWrapper(w http.ResponseWriter) loggingResponseWriter {
	var logger loggingResponseWriter = &responseLogger{w: w, status: http.StatusOK}
	if _, ok := w.(http.Hijacker); ok {
		logger = &hijackLogger{responseLogger{w: w, status: http.StatusOK}}
	}
	h, ok1 := logger.(http.Hijacker)
	c, ok2 := w.(http.CloseNotifier)
	if ok1 && ok2 {
		return hijackCloseNotifier{logger, h, c}
	}
	if ok2 {
		return &closeNotifyWriter{logger, c}
	}
	return logger
}

type hijackLogger struct {
	responseLogger
}

func (l *hijackLogger) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	h := l.responseLogger.w.(http.Hijacker)
	conn, rw, err := h.Hijack()
	if err == nil && l.responseLogger.status == 0 {
		// The status will be StatusSwitchingProtocols if there was no error and
		// WriteHeader has not been called yet
		l.responseLogger.status = http.StatusSwitchingProtocols
	}
	return conn, rw, err
}

type closeNotifyWriter struct {
	loggingResponseWriter
	http.CloseNotifier
}

type hijackCloseNotifier struct {
	loggingResponseWriter
	http.Hijacker
	http.CloseNotifier
}

type loggingResponseWriter interface {
	commonLoggingResponseWriter
	http.Pusher
}

type commonLoggingResponseWriter interface {
	http.ResponseWriter
	http.Flusher
	Status() int
	Size() int
}

type responseLogger struct {
	w      http.ResponseWriter
	status int
	size   int
}

func (l *responseLogger) Header() http.Header {
	return l.w.Header()
}

func (l *responseLogger) Write(b []byte) (int, error) {
	size, err := l.w.Write(b)
	l.size += size
	return size, err
}

func (l *responseLogger) WriteHeader(s int) {
	l.w.WriteHeader(s)
	l.status = s
}

func (l *responseLogger) Status() int {
	return l.status
}

func (l *responseLogger) Size() int {
	return l.size
}

func (l *responseLogger) Flush() {
	f, ok := l.w.(http.Flusher)
	if ok {
		f.Flush()
	}
}

func (l *responseLogger) Push(target string, opts *http.PushOptions) error {
	p, ok := l.w.(http.Pusher)
	if !ok {
		return fmt.Errorf("responseLogger does not implement http.Pusher")
	}
	return p.Push(target, opts)
}
