package server

import (
	"net/http"

	"kratos-demo/internal/conf"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// NewMetricsServer creates a new metrics server for Prometheus
func NewMetricsServer(c *conf.Bootstrap, logger log.Logger) *http.Server {
	helper := log.NewHelper(logger)
	
	mux := http.NewServeMux()
	mux.Handle(c.Telemetry.Metrics.Path, promhttp.Handler())
	
	srv := &http.Server{
		Addr:    c.Telemetry.Metrics.Addr,
		Handler: mux,
	}
	
	go func() {
		helper.Infof("[Metrics] server listening on: %s", srv.Addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			helper.Errorf("[Metrics] server error: %v", err)
		}
	}()
	
	return srv
}