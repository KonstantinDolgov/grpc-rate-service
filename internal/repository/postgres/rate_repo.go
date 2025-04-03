package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
	"go.uber.org/zap"
	"studentgit.kata.academy/KonstantinDolgov/grpc-rate-service/internal/model"
	"studentgit.kata.academy/KonstantinDolgov/grpc-rate-service/internal/repository"
)

type Repository struct {
	db     *sql.DB
	logger *zap.Logger
}

func NewRepository(connString string, logger *zap.Logger) (repository.RateRepository, error) {
	logger.Debug("Initializing database repository")

	db, err := sql.Open("pgx", connString)
	if err != nil {
		logger.Error("Failed to open database connection", zap.Error(err))
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	logger.Debug("Checking database connection")
	if err := db.Ping(); err != nil {
		logger.Error("Failed to ping database", zap.Error(err))
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	logger.Info("Database repository initialized successfully")
	return &Repository{
		db:     db,
		logger: logger,
	}, nil
}

func (r *Repository) SaveRate(ctx context.Context, rate model.Rate) error {
	query := `
		INSERT INTO rates (symbol, ask, bid, timestamp, created_at)
		VALUES ($1, $2, $3, $4, $5)
	`
	_, err := r.db.ExecContext(
		ctx,
		query,
		rate.Symbol,
		rate.Ask,
		rate.Bid,
		rate.Timestamp,
		time.Now(),
	)

	if err != nil {
		r.logger.Error("Failed to save rate",
			zap.String("symbol", rate.Symbol),
			zap.Float64("ask", rate.Ask),
			zap.Float64("bid", rate.Bid),
			zap.Error(err))
		return fmt.Errorf("failed to execute insert query: %w", err)
	}

	r.logger.Debug("Rate saved successfully",
		zap.String("symbol", rate.Symbol),
		zap.Float64("ask", rate.Ask),
		zap.Float64("bid", rate.Bid))

	return nil
}

func (r *Repository) GetLatestRate(ctx context.Context, symbol string) (*model.Rate, error) {
	query := `
		SELECT id, symbol, ask, bid, timestamp, created_at
		FROM rates
		WHERE symbol = $1
		ORDER BY timestamp DESC
		LIMIT 1
	`
	var rate model.Rate
	err := r.db.QueryRowContext(ctx, query, symbol).Scan(
		&rate.ID,
		&rate.Symbol,
		&rate.Ask,
		&rate.Bid,
		&rate.Timestamp,
		&rate.CreatedAt,
	)

	if err != nil {
		r.logger.Error("Failed to get latest rate",
			zap.String("symbol", symbol),
			zap.Error(err))
		return nil, fmt.Errorf("failed to get latest rate: %w", err)
	}

	r.logger.Debug("Retrieved latest rate",
		zap.String("symbol", rate.Symbol),
		zap.Float64("ask", rate.Ask),
		zap.Float64("bid", rate.Bid),
		zap.Time("timestamp", rate.Timestamp))

	return &rate, nil
}

func (r *Repository) Close() error {
	r.logger.Info("Closing database connection")
	if err := r.db.Close(); err != nil {
		r.logger.Error("Failed to close database connection", zap.Error(err))
		return fmt.Errorf("failed to close database connection: %w", err)
	}
	r.logger.Info("Database connection closed successfully")
	return nil
}
