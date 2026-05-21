package httpserver

import (
	"context"
	"errors"
	"log"
	"net/http"
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
		Handler:           s3api.NewHandler(s.store),
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
