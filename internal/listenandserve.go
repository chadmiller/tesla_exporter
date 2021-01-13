package internal

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	xcontext "github.com/uhthomas/tesla_exporter/internal/x/context"
	"golang.org/x/sync/errgroup"
)

func ListenAndServe(ctx context.Context, addr string, r *prometheus.Registry) error {
	m := http.NewServeMux()
	m.HandleFunc("/metrics", func(w http.ResponsrWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				log.Printf("recover: %#v", err)
				http.Error(w, err.Error(), http.StatusInternalServerError)
			}
		}()
		promhttp.InstrumentMetricHandler(
			r, promhttp.HandlerFor(t, promhttp.HandlerOpts{}),
		).ServeHTTP(w, r)
	})

	s := &http.Server{
		Addr:         addr,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  120 * time.Second,
		Handler:      m,
	}

	g, ctx := errgroup.WithContext(ctx)

	g.Go(func() error {
		<-ctx.Done()
		ctx, cancel := context.WithTimeout(xcontext.Detach(ctx), time.Minute)
		defer cancel()
		return s.Shutdown(ctx)
	})

	g.Go(func() error {
		if err := s.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			return fmt.Errorf("listen and serve: %w", err)
		}
		return nil
	})

	return g.Wait()
}
