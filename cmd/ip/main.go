package main

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"os"
	"time"

	"alpineworks.io/ootel"
	"github.com/alpineworks/ip/internal/config"
	h "github.com/alpineworks/ip/internal/handlers"
	"github.com/alpineworks/ip/internal/logging"
	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"go.opentelemetry.io/contrib/instrumentation/host"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/contrib/instrumentation/runtime"
)

func main() {
	logLevel := os.Getenv("LOG_LEVEL")
	if logLevel == "" {
		logLevel = "error"
	}

	slogLevel, err := logging.LogLevelToSlogLevel(logLevel)
	if err != nil {
		log.Fatalf("could not convert log level: %s", err)
	}

	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slogLevel,
	})))
	c, err := config.NewConfig()
	if err != nil {
		slog.Error("could not create config", slog.String("error", err.Error()))
		os.Exit(1)
	}

	ctx := context.Background()

	exporterType := ootel.ExporterTypePrometheus
	if c.Local {
		exporterType = ootel.ExporterTypeOTLPGRPC
	}

	ootelClient := ootel.NewOotelClient(
		ootel.WithMetricConfig(
			ootel.NewMetricConfig(
				c.MetricsEnabled,
				exporterType,
				c.MetricsPort,
			),
		),
		ootel.WithTraceConfig(
			ootel.NewTraceConfig(
				c.TracingEnabled,
				c.TracingSampleRate,
				c.TracingService,
				c.TracingVersion,
			),
		),
	)

	shutdown, err := ootelClient.Init(ctx)
	if err != nil {
		slog.Error("could not create ootel client", slog.String("error", err.Error()))
		os.Exit(1)
	}

	err = runtime.Start(runtime.WithMinimumReadMemStatsInterval(5 * time.Second))
	if err != nil {
		slog.Error("could not create runtime metrics", slog.String("error", err.Error()))
		os.Exit(1)
	}

	err = host.Start()
	if err != nil {
		slog.Error("could not create host metrics", slog.String("error", err.Error()))
		os.Exit(1)
	}

	defer func() {
		_ = shutdown(ctx)
	}()

	r := mux.NewRouter()
	r.Handle("/", otelhttp.WithRouteTag("/", h.IPHandler())).Methods("GET")
	r.Handle("/raw", otelhttp.WithRouteTag("/raw", h.RawIPHandler())).Methods("GET")
	r.Use(otelhttp.NewMiddleware("otelhttp"))

	http.Handle("/", handlers.CORS(handlers.AllowedOrigins([]string{"*"}))(r))

	slog.Info("starting server", slog.Int("port", c.Port))
	_ = http.ListenAndServe(fmt.Sprintf(":%d", c.Port), nil)
}
