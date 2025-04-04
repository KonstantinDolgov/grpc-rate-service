package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
	"studentgit.kata.academy/KonstantinDolgov/grpc-rate-service/internal/model"
	"studentgit.kata.academy/KonstantinDolgov/grpc-rate-service/internal/repository"
)

type Repository struct {
	db     *sql.DB
	logger *zap.Logger
	tracer trace.Tracer
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
		tracer: otel.Tracer("db-repository"),
	}, nil
}

func (r *Repository) SaveRate(ctx context.Context, rate model.Rate) error {
	// Создаем спан для трассировки только если трассировщик инициализирован
	var span trace.Span
	if r.tracer != nil {
		ctx, span = r.tracer.Start(ctx, "Repository.SaveRate",
			trace.WithAttributes(
				attribute.String("symbol", rate.Symbol),
				attribute.Float64("ask", rate.Ask),
				attribute.Float64("bid", rate.Bid),
			))
		defer span.End()
	}

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

		if span != nil {
			span.SetStatus(codes.Error, "Failed to save rate to database")
			span.RecordError(err)
		}

		return fmt.Errorf("failed to execute insert query: %w", err)
	}

	r.logger.Debug("Rate saved successfully",
		zap.String("symbol", rate.Symbol),
		zap.Float64("ask", rate.Ask),
		zap.Float64("bid", rate.Bid))

	if span != nil {
		span.SetStatus(codes.Ok, "Rate saved successfully")
	}
	return nil
}

func (r *Repository) GetLatestRate(ctx context.Context, symbol string) (*model.Rate, error) {
	// Создаем спан для трассировки только если трассировщик инициализирован
	var span trace.Span
	if r.tracer != nil {
		ctx, span = r.tracer.Start(ctx, "Repository.GetLatestRate",
			trace.WithAttributes(attribute.String("symbol", symbol)))
		defer span.End()
	}

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

		if span != nil {
			span.SetStatus(codes.Error, "Failed to get latest rate from database")
			span.RecordError(err)
		}
		return nil, fmt.Errorf("failed to get latest rate: %w", err)
	}

	r.logger.Debug("Retrieved latest rate",
		zap.String("symbol", rate.Symbol),
		zap.Float64("ask", rate.Ask),
		zap.Float64("bid", rate.Bid),
		zap.Time("timestamp", rate.Timestamp))

	if span != nil {
		span.SetAttributes(
			attribute.Float64("ask", rate.Ask),
			attribute.Float64("bid", rate.Bid),
			attribute.String("timestamp", rate.Timestamp.Format(time.RFC3339)),
		)
		span.SetStatus(codes.Ok, "Latest rate retrieved successfully")
	}

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
