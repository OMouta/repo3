package httpserver

import (
	"context"
	"errors"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"repo3/internal/config"
	"repo3/internal/s3api"
	"repo3/internal/storage"
)

type Server struct {
	cfg   config.Config
	store storage.ObjectStore
}

func New(cfg config.Config, store storage.ObjectStore) *Server {
	return &Server{cfg: cfg, store: store}
}

func (s *Server) Run(ctx context.Context) error {
	httpServer := &http.Server{
		Addr:              s.cfg.Addr,
		Handler:           logRequests(s3api.NewHandler(s.store)),
		ReadHeaderTimeout: 10 * time.Second,
	}

	errs := make(chan error, 1)
	go func() {
		log.Printf("repo3 listening on %s", s.cfg.Addr)
		errs <- httpServer.ListenAndServe()
	}()

	select {
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := httpServer.Shutdown(shutdownCtx); err != nil {
			return err
		}
		return nil
	case err := <-errs:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	}
}

type statusRecorder struct {
	http.ResponseWriter
	status int
	bytes  int64
}

func (r *statusRecorder) WriteHeader(status int) {
	if r.status != 0 {
		return
	}
	r.status = status
	r.ResponseWriter.WriteHeader(status)
}

func (r *statusRecorder) Write(data []byte) (int, error) {
	if r.status == 0 {
		r.status = http.StatusOK
	}
	n, err := r.ResponseWriter.Write(data)
	r.bytes += int64(n)
	return n, err
}

func logRequests(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rec := &statusRecorder{ResponseWriter: w}
		next.ServeHTTP(rec, r)
		if rec.status == 0 {
			rec.status = http.StatusOK
		}

		action, bucket, key := classifyAction(r)
		fields := []string{
			"action=" + action,
			"method=" + r.Method,
			"path=" + strconv.Quote(r.URL.Path),
			"status=" + strconv.Itoa(rec.status),
			"duration=" + time.Since(start).String(),
		}
		if bucket != "" {
			fields = append(fields, "bucket="+strconv.Quote(bucket))
		}
		if key != "" {
			fields = append(fields, "key="+strconv.Quote(key))
		}
		if prefix := r.URL.Query().Get("prefix"); prefix != "" {
			fields = append(fields, "prefix="+strconv.Quote(prefix))
		}
		if r.ContentLength > 0 {
			fields = append(fields, "request_bytes="+strconv.FormatInt(r.ContentLength, 10))
		}
		if rec.bytes > 0 {
			fields = append(fields, "response_bytes="+strconv.FormatInt(rec.bytes, 10))
		}
		if rec.status >= 400 {
			fields = append(fields, "level=error")
		}
		log.Print("repo3 request " + strings.Join(fields, " "))
	})
}

func classifyAction(r *http.Request) (string, string, string) {
	bucket, key := splitPath(r.URL.Path)
	switch {
	case bucket == "" && r.Method == http.MethodGet:
		return "list_buckets", "", ""
	case bucket != "" && key == "" && r.Method == http.MethodPut:
		return "create_bucket", bucket, ""
	case bucket != "" && key == "" && r.Method == http.MethodDelete:
		return "delete_bucket", bucket, ""
	case bucket != "" && key == "" && r.Method == http.MethodGet:
		return "list_objects", bucket, ""
	case bucket != "" && key != "" && r.Method == http.MethodPut:
		return "put_object", bucket, key
	case bucket != "" && key != "" && r.Method == http.MethodGet:
		return "get_object", bucket, key
	case bucket != "" && key != "" && r.Method == http.MethodHead:
		return "head_object", bucket, key
	case bucket != "" && key != "" && r.Method == http.MethodDelete:
		return "delete_object", bucket, key
	default:
		return "unknown", bucket, key
	}
}

func splitPath(path string) (string, string) {
	path = strings.TrimPrefix(path, "/")
	if path == "" {
		return "", ""
	}
	bucket, key, _ := strings.Cut(path, "/")
	return bucket, key
}
