package telemetry

import (
	"context"
	"fmt"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.20.0"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type TracingConfig struct {
	ServiceName    string
	ServiceVersion string
	Environment    string
	OTLPEndpoint   string
}

// InitTracing инициализирует трассировку с использованием OpenTelemetry
func InitTracing(ctx context.Context, config TracingConfig, logger *zap.Logger) (func(context.Context) error, error) {
	logger.Info("Инициализация трассировки OpenTelemetry",
		zap.String("service", config.ServiceName),
		zap.String("endpoint", config.OTLPEndpoint))

	// Создаем ресурс с информацией о сервисе
	res, err := resource.New(ctx,
		resource.WithAttributes(
			semconv.ServiceName(config.ServiceName),
			semconv.ServiceVersion(config.ServiceVersion),
			semconv.DeploymentEnvironment(config.Environment),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("не удалось создать ресурс OpenTelemetry: %w", err)
	}

	// Настраиваем коннект к коллектору OTLP (например, Jaeger или Collector)
	opts := []otlptracegrpc.Option{
		otlptracegrpc.WithEndpoint(config.OTLPEndpoint),
		otlptracegrpc.WithInsecure(),
		otlptracegrpc.WithDialOption(grpc.WithTransportCredentials(insecure.NewCredentials())),
		otlptracegrpc.WithDialOption(grpc.WithTransportCredentials(insecure.NewCredentials())),
	}

	client := otlptracegrpc.NewClient(opts...)
	exporter, err := otlptrace.New(ctx, client)
	if err != nil {
		return nil, fmt.Errorf("не удалось создать OTLP экспортер: %w", err)
	}

	// Создаем трасировщик
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(res),
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
	)

	// Устанавливаем глобальный трасировщик
	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	logger.Info("Трассировка OpenTelemetry успешно инициализирована")

	// Возвращаем функцию для закрытия трасировщика
	return func(ctx context.Context) error {
		logger.Info("Завершение работы трасировщика OpenTelemetry")
		return tp.Shutdown(ctx)
	}, nil
}
