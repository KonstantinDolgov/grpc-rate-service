package grpc

import (
	"context"
	"time"

	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
	pb "studentgit.kata.academy/KonstantinDolgov/grpc-rate-service/pkg/grpc/rate_service_v1"
)

// RateServiceInterface - интерфейс для сервиса ставок, для облегчения тестирования
type RateServiceInterface interface {
	GetRates(ctx context.Context, symbol string) (float64, float64, time.Time, error)
	HealthCheck(ctx context.Context) bool
}
type RateServiceServer struct {
	pb.UnimplementedRateServiceServer
	logger      *zap.Logger
	rateService RateServiceInterface
}

func NewRateServiceServer(logger *zap.Logger, rateService RateServiceInterface) *RateServiceServer {
	return &RateServiceServer{
		logger:      logger,
		rateService: rateService,
	}
}

func (s *RateServiceServer) GetRates(ctx context.Context, req *pb.GetRatesRequest) (*pb.GetRatesResponse, error) {
	if req.Symbol == "" {
		return nil, status.Error(codes.InvalidArgument, "symbol is required")
	}

	ask, bid, timestamp, err := s.rateService.GetRates(ctx, req.Symbol)
	if err != nil {
		s.logger.Error("Failed to get rates", zap.Error(err), zap.String("symbol", req.Symbol))
		return nil, status.Error(codes.Internal, "failed to get rates")
	}

	return &pb.GetRatesResponse{
		Ask:       ask,
		Bid:       bid,
		Timestamp: timestamppb.New(timestamp),
	}, nil
}

func (s *RateServiceServer) HealthCheck(ctx context.Context, req *pb.HealthCheckRequest) (*pb.HealthCheckResponse, error) {
	healthy := s.rateService.HealthCheck(ctx)
	return &pb.HealthCheckResponse{
		Healthy: healthy,
	}, nil
}
