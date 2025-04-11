package telemetry

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.opentelemetry.io/otel"
	promexporter "go.opentelemetry.io/otel/exporters/prometheus"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.20.0"
	"go.uber.org/zap"
)

// MetricsConfig содержит настройки для инициализации метрик
type MetricsConfig struct {
	ServiceName    string
	ServiceVersion string
	Environment    string
	HTTPAddr       string
}

// Метрики gRPC сервера
var (
	RequestCounter = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "grpc_requests_total",
			Help: "Total number of gRPC requests",
		},
		[]string{"method", "status"},
	)

	RequestDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "grpc_request_duration_seconds",
			Help:    "Duration of gRPC requests in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method"},
	)

	RateFetchCounter = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "rate_fetch_total",
			Help: "Total number of currency rate fetch requests",
		},
		[]string{"symbol", "status"},
	)
)

func init() {
	// Регистрируем метрики
	prometheus.MustRegister(RequestCounter)
	prometheus.MustRegister(RequestDuration)
	prometheus.MustRegister(RateFetchCounter)
}

// InitMetrics инициализирует метрики с использованием Prometheus и OpenTelemetry
func InitMetrics(ctx context.Context, config MetricsConfig, logger *zap.Logger) (func(context.Context) error, error) {
	logger.Info("Initializing Prometheus metrics",
		zap.String("service", config.ServiceName),
		zap.String("endpoint", config.HTTPAddr))

	// Создаем ресурс с информацией о сервисе
	res, err := resource.New(ctx,
		resource.WithAttributes(
			semconv.ServiceName(config.ServiceName),
			semconv.ServiceVersion(config.ServiceVersion),
			semconv.DeploymentEnvironment(config.Environment),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create OpenTelemetry resource: %w", err)
	}

	// Создаем экспортер Prometheus
	exporter, err := promexporter.New()
	if err != nil {
		return nil, fmt.Errorf("failed to create Prometheus exporter: %w", err)
	}

	// Создаем провайдер метрик
	meterProvider := metric.NewMeterProvider(
		metric.WithReader(exporter),
		metric.WithResource(res),
	)

	// Устанавливаем глобальный провайдер метрик
	otel.SetMeterProvider(meterProvider)

	// Запускаем HTTP сервер для метрик Prometheus
	go func() {
		mux := http.NewServeMux()
		mux.Handle("/metrics", promhttp.Handler())

		server := &http.Server{
			Addr:         config.HTTPAddr,
			Handler:      mux,
			ReadTimeout:  10 * time.Second,
			WriteTimeout: 10 * time.Second,
			IdleTimeout:  30 * time.Second,
		}

		logger.Info("Starting HTTP server for Prometheus metrics", zap.String("addr", config.HTTPAddr))
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error("Failed to start metrics HTTP server", zap.Error(err))
		}
	}()

	logger.Info("Prometheus metrics successfully initialized")

	// Возвращаем функцию для закрытия провайдера метрик
	return func(ctx context.Context) error {
		logger.Info("Shutting down metrics provider")
		return meterProvider.Shutdown(ctx)
	}, nil
}
