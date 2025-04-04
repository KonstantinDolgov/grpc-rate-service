package app

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"go.uber.org/zap"
	"studentgit.kata.academy/KonstantinDolgov/grpc-rate-service/config"
	"studentgit.kata.academy/KonstantinDolgov/grpc-rate-service/internal/model"
	"studentgit.kata.academy/KonstantinDolgov/grpc-rate-service/internal/repository"
)

type MockRepository struct {
	mock.Mock
}

func (m *MockRepository) SaveRate(ctx context.Context, rate model.Rate) error {
	args := m.Called(ctx, rate)
	return args.Error(0)
}

func (m *MockRepository) GetLatestRate(ctx context.Context, symbol string) (*model.Rate, error) {
	args := m.Called(ctx, symbol)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.Rate), args.Error(1)
}

func (m *MockRepository) Close() error {
	args := m.Called()
	return args.Error(0)
}

func TestNewApp(t *testing.T) {
	// Arrange
	logger := zap.NewNop()
	cfg := &config.Config{
		DBHost:        "localhost",
		DBPort:        "5432",
		DBUser:        "user",
		DBPassword:    "password",
		DBName:        "dbname",
		DBSSLMode:     "disable",
		GRPCPort:      "50051",
		LogLevel:      "info",
		KuCoinBaseURL: "https://api.kucoin.com",
	}

	// Подмена функции создания репозитория для тестирования
	originalFunc := newRepositoryFunc
	defer func() { newRepositoryFunc = originalFunc }()

	t.Run("success", func(t *testing.T) {
		// Arrange
		mockRepo := new(MockRepository)

		// Подменяем функцию создания репозитория
		newRepositoryFunc = func(connString string, logger *zap.Logger) (repository.RateRepository, error) {
			return mockRepo, nil
		}

		// Act
		app, err := NewApp(cfg, logger)

		// Assert
		assert.NoError(t, err)
		assert.NotNil(t, app)
		assert.Equal(t, cfg, app.config)
		assert.Equal(t, logger, app.logger)
		assert.Equal(t, mockRepo, app.repo)
	})

	t.Run("repository creation error", func(t *testing.T) {
		// Arrange
		expectedErr := errors.New("repository creation error")

		// Подменяем функцию создания репозитория с ошибкой
		newRepositoryFunc = func(connString string, logger *zap.Logger) (repository.RateRepository, error) {
			return nil, expectedErr
		}

		// Act
		app, err := NewApp(cfg, logger)

		// Assert
		assert.Error(t, err)
		assert.Nil(t, app)
		assert.Contains(t, err.Error(), "failed to create repository")
	})
}

func TestShutdown(t *testing.T) {
	// Arrange
	logger := zap.NewNop()
	mockRepo := new(MockRepository)

	app := &App{
		logger: logger,
		repo:   mockRepo,
	}

	// Ожидаем вызов закрытия репозитория
	mockRepo.On("Close").Return(nil)

	// Act
	app.Shutdown()

	// Assert
	mockRepo.AssertExpectations(t)
}

func TestShutdown_WithError(t *testing.T) {
	// Arrange
	logger := zap.NewNop()
	mockRepo := new(MockRepository)

	app := &App{
		logger: logger,
		repo:   mockRepo,
	}

	// Ожидаем вызов закрытия репозитория с ошибкой
	expectedErr := errors.New("close error")
	mockRepo.On("Close").Return(expectedErr)

	// Act - не должно паниковать
	app.Shutdown()

	// Assert
	mockRepo.AssertExpectations(t)
	// Мы не можем проверить логгирование напрямую, так как используем zap.NewNop()
}
