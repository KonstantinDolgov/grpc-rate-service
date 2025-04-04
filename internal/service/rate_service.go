package service

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"go.uber.org/zap"
	"studentgit.kata.academy/KonstantinDolgov/grpc-rate-service/internal/exchange/kucoin"
	"studentgit.kata.academy/KonstantinDolgov/grpc-rate-service/internal/model"
	"studentgit.kata.academy/KonstantinDolgov/grpc-rate-service/internal/repository"
)

type RateService struct {
	logger       *zap.Logger
	repo         repository.RateRepository
	kuCoinClient *kucoin.KuCoinClient
}

func NewRateService(logger *zap.Logger, repo repository.RateRepository, kuCoinClient *kucoin.KuCoinClient) *RateService {
	return &RateService{
		logger:       logger,
		repo:         repo,
		kuCoinClient: kuCoinClient,
	}
}

func (s *RateService) GetRates(ctx context.Context, symbol string) (float64, float64, time.Time, error) {
	ask, bid, timestamp, err := s.kuCoinClient.GetOrderBook(ctx, symbol)
	if err != nil {
		s.logger.Error("Failed to get order book", zap.Error(err), zap.String("symbol", symbol))
		return 0, 0, time.Time{}, err
	}

	// Сохраняем данные о курсе в БД
	rate := model.Rate{
		Symbol:    symbol,
		Ask:       ask,
		Bid:       bid,
		Timestamp: timestamp,
		CreatedAt: time.Now(),
	}

	if err := s.repo.SaveRate(ctx, rate); err != nil {
		s.logger.Error("Failed to save rate",
			zap.Error(err),
			zap.String("symbol", symbol),
			zap.Float64("ask", ask),
			zap.Float64("bid", bid),
			zap.Time("timestamp", timestamp))

		// Не возвращаем ошибку, чтобы клиент все равно получил данные о курсе
		// Можно добавить метрику для мониторинга таких ситуаций
	} else {
		s.logger.Info("Successfully saved rate to database",
			zap.String("symbol", symbol),
			zap.Float64("ask", ask),
			zap.Float64("bid", bid))
	}

	return ask, bid, timestamp, nil
}

func (s *RateService) HealthCheck(ctx context.Context) bool {
	// Проверяем соединение с базой данных
	_, err := s.repo.GetLatestRate(ctx, "BTC-USDT")
	if err != nil {
		// Пустая база не считается ошибкой, только проверяем, есть ли соединение с БД
		if !errors.Is(err, sql.ErrNoRows) {
			s.logger.Error("Health check failed: database connection error", zap.Error(err))
			return false
		}
	}

	// Проверяем соединение с API KuCoin
	_, _, _, err = s.kuCoinClient.GetOrderBook(ctx, "BTC-USDT")
	if err != nil {
		s.logger.Error("Health check failed: KuCoin API connection error", zap.Error(err))
		return false
	}

	return true
}
