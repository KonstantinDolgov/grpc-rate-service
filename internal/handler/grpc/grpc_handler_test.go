package grpc

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
	pb "studentgit.kata.academy/KonstantinDolgov/grpc-rate-service/pkg/grpc/rate_service_v1"
)

type MockRateService struct {
	mock.Mock
}

func (m *MockRateService) GetRates(ctx context.Context, symbol string) (float64, float64, time.Time, error) {
	args := m.Called(ctx, symbol)
	return args.Get(0).(float64), args.Get(1).(float64), args.Get(2).(time.Time), args.Error(3)
}

func (m *MockRateService) HealthCheck(ctx context.Context) bool {
	args := m.Called(ctx)
	return args.Bool(0)
}

func TestGetRates_Success(t *testing.T) {
	// Arrange
	mockService := new(MockRateService)
	logger := zap.NewNop()
	server := NewRateServiceServer(logger, mockService)

	ctx := context.Background()
	symbol := "BTC-USDT"
	ask := 40000.5
	bid := 39999.5
	timestamp := time.Now().UTC()

	// Настраиваем мок
	mockService.On("GetRates", ctx, symbol).Return(ask, bid, timestamp, nil)

	// Act
	resp, err := server.GetRates(ctx, &pb.GetRatesRequest{Symbol: symbol})

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, ask, resp.Ask)
	assert.Equal(t, bid, resp.Bid)
	assert.Equal(t, timestamppb.New(timestamp).AsTime(), resp.Timestamp.AsTime())
	mockService.AssertExpectations(t)
}

func TestGetRates_EmptySymbol(t *testing.T) {
	// Arrange
	mockService := new(MockRateService)
	logger := zap.NewNop()
	server := NewRateServiceServer(logger, mockService)

	ctx := context.Background()

	// Act
	resp, err := server.GetRates(ctx, &pb.GetRatesRequest{Symbol: ""})

	// Assert
	assert.Error(t, err)
	assert.Nil(t, resp)

	// Проверяем, что ошибка имеет правильный gRPC код
	st, ok := status.FromError(err)
	assert.True(t, ok)
	assert.Equal(t, codes.InvalidArgument, st.Code())
	assert.Contains(t, st.Message(), "symbol is required")

	// Проверяем, что сервис не был вызван
	mockService.AssertNotCalled(t, "GetRates")
}

func TestGetRates_ServiceError(t *testing.T) {
	// Arrange
	mockService := new(MockRateService)
	logger := zap.NewNop()
	server := NewRateServiceServer(logger, mockService)

	ctx := context.Background()
	symbol := "BTC-USDT"
	expectedErr := errors.New("service error")

	// Настраиваем мок с ошибкой
	mockService.On("GetRates", ctx, symbol).Return(0.0, 0.0, time.Time{}, expectedErr)

	// Act
	resp, err := server.GetRates(ctx, &pb.GetRatesRequest{Symbol: symbol})

	// Assert
	assert.Error(t, err)
	assert.Nil(t, resp)

	// Проверяем, что ошибка имеет правильный gRPC код
	st, ok := status.FromError(err)
	assert.True(t, ok)
	assert.Equal(t, codes.Internal, st.Code())
	assert.Contains(t, st.Message(), "failed to get rates")

	mockService.AssertExpectations(t)
}

func TestHealthCheck_Healthy(t *testing.T) {
	// Arrange
	mockService := new(MockRateService)
	logger := zap.NewNop()
	server := NewRateServiceServer(logger, mockService)

	ctx := context.Background()

	// Настраиваем мок
	mockService.On("HealthCheck", ctx).Return(true)

	// Act
	resp, err := server.HealthCheck(ctx, &pb.HealthCheckRequest{})

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.True(t, resp.Healthy)
	mockService.AssertExpectations(t)
}

func TestHealthCheck_Unhealthy(t *testing.T) {
	// Arrange
	mockService := new(MockRateService)
	logger := zap.NewNop()
	server := NewRateServiceServer(logger, mockService)

	ctx := context.Background()

	// Настраиваем мок
	mockService.On("HealthCheck", ctx).Return(false)

	// Act
	resp, err := server.HealthCheck(ctx, &pb.HealthCheckRequest{})

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.False(t, resp.Healthy)
	mockService.AssertExpectations(t)
}
