package servers

import (
	"compress/gzip"
	"context"

	"io"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/aaarkadev/collectalertagent/internal/configs"
	"github.com/aaarkadev/collectalertagent/internal/repositories"
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
	/*w.Header().Del("Content-Length")
	w.Header()["Content-Length"] = nil
	w.Header().Set("Transfer-Encoding", "chunked")
	w.Header().Set("Connection", "Close")
	*/
	w.ResponseWriter.WriteHeader(code)

	//if  w.isCompressable() { w.Header().Del("Content-Length")}
}

func (w *ServerHandlerData) Write(b []byte) (int, error) {
	isValidContentType := true

	if w.IsHeadersWriten {
		isValidContentType = false
	}
	/*if len(b) <= 1400 ||   {
		isValidContentType = false
	}*/

	if isValidContentType {
		//mime.ParseMediaType
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
			return
		}
		//defer gz.Close()

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
			panic(err)
		}
	}()

	<-sigChan

	shutdownCtx, shutdownCtxCancel := context.WithTimeout(mainCtx, configs.GlobalDefaultTimeout)
	defer shutdownCtxCancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		panic(err)
	}

	return server
}

func BindServerToHandler(s *ServerHandlerData, f func(http.ResponseWriter, *http.Request, *ServerHandlerData)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		f(w, r, s)
	}
}
