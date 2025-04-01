package repository

import (
	"context"
	"studentgit.kata.academy/KonstantinDolgov/grpc-rate-service/internal/model"
)

type RateRepository interface {
	SaveRate(ctx context.Context, rate model.Rate) error
	GetLatestRate(ctx context.Context, symbol string) (*model.Rate, error)
	Close() error
}
