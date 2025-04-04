package service

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
	"studentgit.kata.academy/KonstantinDolgov/grpc-rate-service/internal/exchange/kucoin"
	"studentgit.kata.academy/KonstantinDolgov/grpc-rate-service/internal/model"
	"studentgit.kata.academy/KonstantinDolgov/grpc-rate-service/internal/repository"
	"studentgit.kata.academy/KonstantinDolgov/grpc-rate-service/pkg/telemetry"
)

type RateService struct {
	logger       *zap.Logger
	repo         repository.RateRepository
	kuCoinClient *kucoin.KuCoinClient
	tracer       trace.Tracer
}

func NewRateService(logger *zap.Logger, repo repository.RateRepository, kuCoinClient *kucoin.KuCoinClient) *RateService {
	return &RateService{
		logger:       logger,
		repo:         repo,
		kuCoinClient: kuCoinClient,
		tracer:       otel.Tracer("rate-service"),
	}
}

func (s *RateService) GetRates(ctx context.Context, symbol string) (float64, float64, time.Time, error) {
	// Создаем спан для трассировки
	ctx, span := s.tracer.Start(ctx, "RateService.GetRates",
		trace.WithAttributes(attribute.String("symbol", symbol)))
	defer span.End()

	ask, bid, timestamp, err := s.kuCoinClient.GetOrderBook(ctx, symbol)
	if err != nil {
		s.logger.Error("Failed to get order book", zap.Error(err), zap.String("symbol", symbol))

		// Отмечаем ошибку в трассировке
		span.SetStatus(codes.Error, "Failed to get order book")
		span.RecordError(err)

		// Обновляем метрику
		telemetry.RateFetchCounter.WithLabelValues(symbol, "error").Inc()

		return 0, 0, time.Time{}, err
	}

	// Обновляем информацию в спане
	span.SetAttributes(
		attribute.Float64("ask", ask),
		attribute.Float64("bid", bid),
		attribute.String("timestamp", timestamp.Format(time.RFC3339)),
	)

	// Обновляем метрику успешного получения курса
	telemetry.RateFetchCounter.WithLabelValues(symbol, "success").Inc()

	// Сохраняем данные о курсе в БД
	rate := model.Rate{
		Symbol:    symbol,
		Ask:       ask,
		Bid:       bid,
		Timestamp: timestamp,
		CreatedAt: time.Now(),
	}

	// Создаем вложенный спан для сохранения в БД
	ctxSave, spanSave := s.tracer.Start(ctx, "RateService.SaveRate")
	if err := s.repo.SaveRate(ctxSave, rate); err != nil {
		s.logger.Error("Failed to save rate",
			zap.Error(err),
			zap.String("symbol", symbol),
			zap.Float64("ask", ask),
			zap.Float64("bid", bid),
			zap.Time("timestamp", timestamp))

		// Отмечаем ошибку в трассировке
		spanSave.SetStatus(codes.Error, "Failed to save rate to database")
		spanSave.RecordError(err)

		// Не возвращаем ошибку, чтобы клиент все равно получил данные о курсе
	} else {
		s.logger.Info("Successfully saved rate to database",
			zap.String("symbol", symbol),
			zap.Float64("ask", ask),
			zap.Float64("bid", bid))

		spanSave.SetStatus(codes.Ok, "Successfully saved rate to database")
	}
	spanSave.End()

	return ask, bid, timestamp, nil
}

func (s *RateService) HealthCheck(ctx context.Context) bool {
	// Создаем спан для трассировки
	ctx, span := s.tracer.Start(ctx, "RateService.HealthCheck")
	defer span.End()

	// Проверяем доступность репозитория
	_, err := s.repo.GetLatestRate(ctx, "BTC-USDT")
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		s.logger.Error("Repository health check failed", zap.Error(err))
		span.SetStatus(codes.Error, "Repository health check failed")
		span.RecordError(err)
		return false
	}

	span.SetStatus(codes.Ok, "Health check successful")

	return true
}
