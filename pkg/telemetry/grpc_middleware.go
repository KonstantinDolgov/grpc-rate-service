package telemetry

import (
	"context"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

// TracingUnaryServerInterceptor создает перехватчик для унарных запросов
// который добавляет трассировку и метрики
func TracingUnaryServerInterceptor(serviceName string) grpc.UnaryServerInterceptor {
	tracer := otel.Tracer(serviceName)

	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		// Извлекаем метаданные
		md, ok := metadata.FromIncomingContext(ctx)
		if !ok {
			md = metadata.New(nil)
		}

		// Начинаем трассировку
		ctx, span := tracer.Start(ctx, info.FullMethod, trace.WithAttributes(
			attribute.String("grpc.method", info.FullMethod),
		))
		defer span.End()

		// Добавляем метаданные запроса в span
		for k, vs := range md {
			if len(vs) > 0 {
				span.SetAttributes(attribute.String("grpc.metadata."+k, vs[0]))
			}
		}

		// Начинаем отсчет времени для метрики длительности запроса
		startTime := time.Now()

		// Обрабатываем запрос
		resp, err := handler(ctx, req)

		// Фиксируем метрики
		duration := time.Since(startTime).Seconds()
		statusCode := "ok"
		if err != nil {
			st, _ := status.FromError(err)
			statusCode = st.Code().String()
			span.SetStatus(codes.Error, err.Error())
			span.RecordError(err)
		}

		// Обновляем метрики
		RequestCounter.WithLabelValues(info.FullMethod, statusCode).Inc()
		RequestDuration.WithLabelValues(info.FullMethod).Observe(duration)

		return resp, err
	}
}

// MetricsUnaryServerInterceptor создает перехватчик для унарных запросов
// который добавляет только метрики (без трассировки)
func MetricsUnaryServerInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		// Начинаем отсчет времени для метрики длительности запроса
		startTime := time.Now()

		// Обрабатываем запрос
		resp, err := handler(ctx, req)

		// Фиксируем метрики
		duration := time.Since(startTime).Seconds()
		statusCode := "ok"
		if err != nil {
			st, _ := status.FromError(err)
			statusCode = st.Code().String()
		}

		// Обновляем метрики
		RequestCounter.WithLabelValues(info.FullMethod, statusCode).Inc()
		RequestDuration.WithLabelValues(info.FullMethod).Observe(duration)

		return resp, err
	}
}
