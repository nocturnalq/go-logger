package gologger

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"runtime"

	"github.com/gorilla/mux"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/rs/zerolog/pkgerrors"
)

func InitLogger(config LoggerConfig) error {
	logLevel, found := LogLevelMapping[config.LogLevel]
	if !found {
		return fmt.Errorf("loglevel %s not exist", config.LogLevel)
	}

	zerolog.SetGlobalLevel(logLevel)
	zerolog.ErrorStackMarshaler = pkgerrors.MarshalStack

	if err := InitSentry(config.SentryDSN, config.SentryDebugMode); err != nil {
		return err
	}

	return nil
}

type LoggingResponseWriter struct {
	http.ResponseWriter
	StatusCode int
	isCalled   bool
	err        error
}

// implements http.ResponseWriter interface
func (r *LoggingResponseWriter) Header() http.Header {
	return r.ResponseWriter.Header()
}

// implements http.ResponseWriter interface and set 'already called flag'
func (r *LoggingResponseWriter) Write(p []byte) (int, error) {
	r.isCalled = true
	return r.ResponseWriter.Write(p)
}

// implements http.ResponseWriter interface and save response status code
func (lrw *LoggingResponseWriter) WriteHeader(code int) {
	lrw.StatusCode = code
	lrw.ResponseWriter.WriteHeader(code)
}

type handlerFunc func(w http.ResponseWriter, r *http.Request) error

func (f handlerFunc) ServeHTTP(w *LoggingResponseWriter, r *http.Request) {
	// save body for logging
	var (
		body []byte
		err  error
	)
	if r.Method == "PUT" || r.Method == "POST" || r.Method == "PATCH" {
		body, err = io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		}
		r.Body = io.NopCloser(bytes.NewBuffer(body))
	}

	// log panic and call http internal error
	defer func() {
		if err := recover(); err != nil {
			buf := make([]byte, 2048)
			n := runtime.Stack(buf, false)
			buf = buf[:n]

			log.Error().
				Err(fmt.Errorf("%v", err)).
				Bytes("stack", buf).
				Str("method", r.Method).
				Str("url", r.URL.RequestURI()).
				Str("user_agent", r.UserAgent()).
				Str("IP", r.RemoteAddr).
				RawJSON("body", body).
				Interface("params", r.URL.Query()).
				Send()

			http.Error(w, "server got panic", http.StatusInternalServerError)
		}
	}()

	if err := f(w, r); err != nil {
		// log error anyway
		log.Error().
			Stack().
			Err(err).
			Str("method", r.Method).
			Str("url", r.URL.RequestURI()).
			Str("user_agent", r.UserAgent()).
			Str("IP", r.RemoteAddr).
			RawJSON("body", body).
			Interface("params", r.URL.Query()).
			Send()

		if !w.isCalled {
			CaptureErrorWithSentry(err)
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		}
	}
}

func adaptHandlerToError(f handlerFunc) http.HandlerFunc {
	return http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		lrw := &LoggingResponseWriter{
			ResponseWriter: rw,
		}

		f.ServeHTTP(lrw, req)
	})
}

type ErrorRouter struct {
	*mux.Router
}

func NewErrorRouter() *ErrorRouter {
	return &ErrorRouter{mux.NewRouter()}
}

func (er *ErrorRouter) HandleFunc(path string, f handlerFunc) *mux.Route {
	return er.NewRoute().Path(path).HandlerFunc(adaptHandlerToError(f))
}

func (er *ErrorRouter) PathPrefix(tpl string) *ErrorRoute {
	return &ErrorRoute{er.Router.PathPrefix(tpl)}
}

type ErrorRoute struct {
	*mux.Route
}

func (er *ErrorRoute) Subrouter() *ErrorRouter {
	return &ErrorRouter{er.Route.Subrouter()}
}
