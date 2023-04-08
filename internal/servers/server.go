package servers

import (
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/aaarkadev/collectalertagent/internal/configs"
	"github.com/aaarkadev/collectalertagent/internal/repositories"
	"github.com/aaarkadev/collectalertagent/internal/types"
)

type ServerHandlerData struct {
	Repo            repositories.Repo
	Config          configs.ServerConfig
	IsHeadersWriten bool
	Writer          gzip.Writer
	http.ResponseWriter
}

func (w *ServerHandlerData) WriteHeader(code int) {

	w.IsHeadersWriten = true
	w.ResponseWriter.WriteHeader(code)
}

func (w *ServerHandlerData) Write(b []byte) (int, error) {
	isValidContentType := true

	if w.IsHeadersWriten {
		isValidContentType = false
	}

	if isValidContentType {
		ct := w.Header().Get("Content-Type")
		if len(ct) <= 0 {
			ct = http.DetectContentType(b)
		}

		var contentTypesToGzip = []string{
			"text/html",
			"text/plain",
			"application/json",
		}

		isValidContentType = false
		for _, v := range contentTypesToGzip {
			if strings.Contains(strings.ToLower(ct), v) {
				isValidContentType = true
				break
			}
		}
	}
	if !isValidContentType {
		return w.ResponseWriter.Write(b)
	}

	w.Header().Set("Content-Encoding", "gzip")
	w.Header().Set("Vary", "Accept-Encoding")
	defer w.Writer.Close()
	return w.Writer.Write(b)
}

func UnGzipMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		if !strings.Contains(strings.ToLower(r.Header.Get("Content-Encoding")), "gzip") {
			next.ServeHTTP(w, r)
			return
		}

		gz, err := gzip.NewReader(r.Body)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			log.Panicln(types.NewTimeError(fmt.Errorf("server.UnGzipMiddleware(): fail: %w", err)))
			return
		}
		defer gz.Close()

		r.Body = io.NopCloser(gz)
		r.ContentLength = -1

		next.ServeHTTP(w, r)
	})
}

func GzipMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		if !strings.Contains(strings.ToLower(r.Header.Get("Accept-Encoding")), "gzip") {
			next.ServeHTTP(w, r)
			return
		}
		gz, err := gzip.NewWriterLevel(w, gzip.BestSpeed)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			log.Panicln(types.NewTimeError(fmt.Errorf("server.GzipMiddleware(): fail: %w", err)))
			return
		}

		next.ServeHTTP(&ServerHandlerData{ResponseWriter: w, Writer: *gz}, r)
	})
}

func StartServer(mainCtx context.Context, config configs.ServerConfig, router http.Handler) *http.Server {

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan,
		syscall.SIGHUP,
		syscall.SIGINT,
		syscall.SIGTERM,
		syscall.SIGQUIT)

	server := &http.Server{Addr: config.ListenAddress, Handler: router}

	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Panicln(types.NewTimeError(fmt.Errorf("server.StartServer(): fail: %w", err)))
		}
	}()

	<-sigChan

	shutdownCtx, shutdownCtxCancel := context.WithTimeout(mainCtx, configs.GlobalDefaultTimeout)
	defer shutdownCtxCancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Panicln(types.NewTimeError(fmt.Errorf("server.StartServer(): fail: %w", err)))
	}

	return server
}

func BindServerToHandler(s *ServerHandlerData, f func(http.ResponseWriter, *http.Request, *ServerHandlerData)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		f(w, r, s)
	}
}
