package service

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"go.uber.org/zap"
	"studentgit.kata.academy/KonstantinDolgov/grpc-rate-service/internal/model"
	"studentgit.kata.academy/KonstantinDolgov/grpc-rate-service/internal/repository"
)

type MockRateRepository struct {
	mock.Mock
}

func (m *MockRateRepository) SaveRate(ctx context.Context, rate model.Rate) error {
	args := m.Called(ctx, rate)
	return args.Error(0)
}

func (m *MockRateRepository) GetLatestRate(ctx context.Context, symbol string) (*model.Rate, error) {
	args := m.Called(ctx, symbol)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.Rate), args.Error(1)
}

func (m *MockRateRepository) Close() error {
	args := m.Called()
	return args.Error(0)
}

// Мок для KuCoin клиента
type MockKuCoinClient struct {
	mock.Mock
}

func (m *MockKuCoinClient) GetOrderBook(ctx context.Context, symbol string) (float64, float64, time.Time, error) {
	args := m.Called(ctx, symbol)
	return args.Get(0).(float64), args.Get(1).(float64), args.Get(2).(time.Time), args.Error(3)
}

// Тестовая структура для внедрения мока KuCoin клиента
type testRateService struct {
	RateService
	mockKuCoin *MockKuCoinClient
}

func newTestRateService(logger *zap.Logger, repo repository.RateRepository) *testRateService {
	mockKuCoin := new(MockKuCoinClient)
	return &testRateService{
		RateService: RateService{
			logger: logger,
			repo:   repo,
			// kuCoinClient поле не инициализируем, так как переопределяем его методы
		},
		mockKuCoin: mockKuCoin,
	}
}

func (s *testRateService) GetRates(ctx context.Context, symbol string) (float64, float64, time.Time, error) {
	ask, bid, timestamp, err := s.mockKuCoin.GetOrderBook(ctx, symbol)
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
	}

	return ask, bid, timestamp, nil
}

func (s *testRateService) HealthCheck(ctx context.Context) bool {
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
	_, _, _, err = s.mockKuCoin.GetOrderBook(ctx, "BTC-USDT")
	if err != nil {
		s.logger.Error("Health check failed: KuCoin API connection error", zap.Error(err))
		return false
	}

	return true
}

func TestGetRates(t *testing.T) {
	// Arrange
	mockRepo := new(MockRateRepository)
	logger := zap.NewNop()
	service := newTestRateService(logger, mockRepo)

	ctx := context.Background()
	symbol := "BTC-USDT"
	ask := 40000.5
	bid := 39999.5
	timestamp := time.Now().UTC()

	// Настраиваем мок KuCoin клиента
	service.mockKuCoin.On("GetOrderBook", ctx, symbol).Return(ask, bid, timestamp, nil)

	// Настраиваем мок репозитория
	mockRepo.On("SaveRate", ctx, mock.MatchedBy(func(rate model.Rate) bool {
		return rate.Symbol == symbol && rate.Ask == ask && rate.Bid == bid && rate.Timestamp == timestamp
	})).Return(nil)

	// Act
	resultAsk, resultBid, resultTimestamp, err := service.GetRates(ctx, symbol)

	// Assert
	assert.NoError(t, err)
	assert.Equal(t, ask, resultAsk)
	assert.Equal(t, bid, resultBid)
	assert.Equal(t, timestamp, resultTimestamp)

	service.mockKuCoin.AssertExpectations(t)
	mockRepo.AssertExpectations(t)
}

func TestGetRates_KuCoinError(t *testing.T) {
	// Arrange
	mockRepo := new(MockRateRepository)
	logger := zap.NewNop()
	service := newTestRateService(logger, mockRepo)

	ctx := context.Background()
	symbol := "BTC-USDT"
	expectedErr := errors.New("kucoin error")

	// Настраиваем мок KuCoin клиента с ошибкой
	service.mockKuCoin.On("GetOrderBook", ctx, symbol).Return(0.0, 0.0, time.Time{}, expectedErr)

	// Act
	_, _, _, err := service.GetRates(ctx, symbol)

	// Assert
	assert.Error(t, err)
	assert.Equal(t, expectedErr, err)

	service.mockKuCoin.AssertExpectations(t)
	// Репозиторий не должен быть вызван при ошибке KuCoin
	mockRepo.AssertNotCalled(t, "SaveRate")
}

func TestGetRates_RepoError(t *testing.T) {
	// Arrange
	mockRepo := new(MockRateRepository)
	logger := zap.NewNop()
	service := newTestRateService(logger, mockRepo)

	ctx := context.Background()
	symbol := "BTC-USDT"
	ask := 40000.5
	bid := 39999.5
	timestamp := time.Now().UTC()
	repoError := errors.New("database error")

	// Настраиваем мок KuCoin клиента
	service.mockKuCoin.On("GetOrderBook", ctx, symbol).Return(ask, bid, timestamp, nil)

	// Настраиваем мок репозитория с ошибкой
	mockRepo.On("SaveRate", ctx, mock.MatchedBy(func(rate model.Rate) bool {
		return rate.Symbol == symbol && rate.Ask == ask && rate.Bid == bid && rate.Timestamp == timestamp
	})).Return(repoError)

	// Act
	resultAsk, resultBid, resultTimestamp, err := service.GetRates(ctx, symbol)

	// Assert
	assert.NoError(t, err) // Ошибка сохранения не возвращается клиенту
	assert.Equal(t, ask, resultAsk)
	assert.Equal(t, bid, resultBid)
	assert.Equal(t, timestamp, resultTimestamp)

	service.mockKuCoin.AssertExpectations(t)
	mockRepo.AssertExpectations(t)
}

func TestHealthCheck_AllOk(t *testing.T) {
	// Arrange
	mockRepo := new(MockRateRepository)
	logger := zap.NewNop()
	service := newTestRateService(logger, mockRepo)

	ctx := context.Background()
	rate := &model.Rate{Symbol: "BTC-USDT", Ask: 40000.5, Bid: 39999.5}

	// Настраиваем мок репозитория
	mockRepo.On("GetLatestRate", ctx, "BTC-USDT").Return(rate, nil)

	// Настраиваем мок KuCoin клиента
	service.mockKuCoin.On("GetOrderBook", ctx, "BTC-USDT").Return(40000.5, 39999.5, time.Now(), nil)

	// Act
	result := service.HealthCheck(ctx)

	// Assert
	assert.True(t, result)
	mockRepo.AssertExpectations(t)
	service.mockKuCoin.AssertExpectations(t)
}

func TestHealthCheck_EmptyDB(t *testing.T) {
	// Arrange
	mockRepo := new(MockRateRepository)
	logger := zap.NewNop()
	service := newTestRateService(logger, mockRepo)

	ctx := context.Background()

	// Настраиваем мок репозитория с отсутствующими данными
	mockRepo.On("GetLatestRate", ctx, "BTC-USDT").Return(nil, sql.ErrNoRows)

	// Настраиваем мок KuCoin клиента
	service.mockKuCoin.On("GetOrderBook", ctx, "BTC-USDT").Return(40000.5, 39999.5, time.Now(), nil)

	// Act
	result := service.HealthCheck(ctx)

	// Assert
	assert.True(t, result)
	mockRepo.AssertExpectations(t)
	service.mockKuCoin.AssertExpectations(t)
}

func TestHealthCheck_DBError(t *testing.T) {
	// Arrange
	mockRepo := new(MockRateRepository)
	logger := zap.NewNop()
	service := newTestRateService(logger, mockRepo)

	ctx := context.Background()
	dbError := errors.New("database connection error")

	// Настраиваем мок репозитория с ошибкой
	mockRepo.On("GetLatestRate", ctx, "BTC-USDT").Return(nil, dbError)

	// Act
	result := service.HealthCheck(ctx)

	// Assert
	assert.False(t, result)
	mockRepo.AssertExpectations(t)
	// KuCoin клиент не должен быть вызван при ошибке базы данных
	service.mockKuCoin.AssertNotCalled(t, "GetOrderBook")
}

func TestHealthCheck_KuCoinError(t *testing.T) {
	// Arrange
	mockRepo := new(MockRateRepository)
	logger := zap.NewNop()
	service := newTestRateService(logger, mockRepo)

	ctx := context.Background()
	rate := &model.Rate{Symbol: "BTC-USDT", Ask: 40000.5, Bid: 39999.5}
	kuCoinError := errors.New("kucoin api error")

	// Настраиваем мок репозитория
	mockRepo.On("GetLatestRate", ctx, "BTC-USDT").Return(rate, nil)

	// Настраиваем мок KuCoin клиента с ошибкой
	service.mockKuCoin.On("GetOrderBook", ctx, "BTC-USDT").Return(0.0, 0.0, time.Time{}, kuCoinError)

	// Act
	result := service.HealthCheck(ctx)

	// Assert
	assert.False(t, result)
	mockRepo.AssertExpectations(t)
	service.mockKuCoin.AssertExpectations(t)
}
