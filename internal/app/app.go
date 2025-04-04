package app

import (
	"context"
	"fmt"
	"net"
	"os"
	"os/signal"
	"syscall"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
	"studentgit.kata.academy/KonstantinDolgov/grpc-rate-service/config"
	"studentgit.kata.academy/KonstantinDolgov/grpc-rate-service/internal/exchange/kucoin"
	grpcServer "studentgit.kata.academy/KonstantinDolgov/grpc-rate-service/internal/handler/grpc"
	"studentgit.kata.academy/KonstantinDolgov/grpc-rate-service/internal/repository"
	"studentgit.kata.academy/KonstantinDolgov/grpc-rate-service/internal/repository/postgres"
	"studentgit.kata.academy/KonstantinDolgov/grpc-rate-service/internal/service"
	pb "studentgit.kata.academy/KonstantinDolgov/grpc-rate-service/pkg/grpc/rate_service_v1"
	"studentgit.kata.academy/KonstantinDolgov/grpc-rate-service/pkg/telemetry"
)

type App struct {
	config       *config.Config
	logger       *zap.Logger
	grpcServer   *grpc.Server
	repo         repository.RateRepository
	cleanupFuncs []func(context.Context) error
}

// Переменная для подмены в тестах
var newRepositoryFunc = postgres.NewRepository

func NewApp(config *config.Config, logger *zap.Logger) (*App, error) {
	// Создание репозитория
	repo, err := newRepositoryFunc(config.GetDBConnString(), logger)
	if err != nil {
		return nil, fmt.Errorf("failed to create repository: %w", err)
	}

	return &App{
		config:       config,
		logger:       logger,
		repo:         repo,
		cleanupFuncs: make([]func(context.Context) error, 0),
	}, nil
}

func (a *App) Run(ctx context.Context) error {
	// Инициализация трассировки
	if a.config.EnableTracing {
		tracingCleanup, err := telemetry.InitTracing(ctx, telemetry.TracingConfig{
			ServiceName:    a.config.ServiceName,
			ServiceVersion: a.config.ServiceVersion,
			Environment:    a.config.Environment,
			OTLPEndpoint:   a.config.OTLPEndpoint,
		}, a.logger)

		if err != nil {
			a.logger.Warn("Не удалось инициализировать трассировку", zap.Error(err))
		} else {
			a.cleanupFuncs = append(a.cleanupFuncs, tracingCleanup)
		}
	}

	// Инициализация метрик
	if a.config.EnableMetrics {
		metricsCleanup, err := telemetry.InitMetrics(ctx, telemetry.MetricsConfig{
			ServiceName:    a.config.ServiceName,
			ServiceVersion: a.config.ServiceVersion,
			Environment:    a.config.Environment,
			HTTPAddr:       a.config.MetricsHTTPAddr,
		}, a.logger)

		if err != nil {
			a.logger.Warn("Не удалось инициализировать метрики", zap.Error(err))
		} else {
			a.cleanupFuncs = append(a.cleanupFuncs, metricsCleanup)
		}
	}

	// Запуск миграций
	if err := a.runMigrations(); err != nil {
		return fmt.Errorf("failed to run migrations: %w", err)
	}

	// Создание клиента KuCoin
	kuCoinClient := kucoin.NewKucoinClient(a.config.KuCoinBaseURL, a.logger)

	// Создание сервиса
	rateService := service.NewRateService(a.logger, a.repo, kuCoinClient)

	// Создание GRPC-сервера
	rateServiceServer := grpcServer.NewRateServiceServer(a.logger, rateService)

	// Создание и настройка GRPC-сервера с middleware для трассировки и метрик
	var serverOptions []grpc.ServerOption

	// Добавляем перехватчики для трассировки и метрик
	if a.config.EnableTracing {
		serverOptions = append(serverOptions,
			grpc.UnaryInterceptor(telemetry.TracingUnaryServerInterceptor(a.config.ServiceName)),
			grpc.StatsHandler(otelgrpc.NewServerHandler()),
		)
	} else if a.config.EnableMetrics {
		serverOptions = append(serverOptions,
			grpc.UnaryInterceptor(telemetry.MetricsUnaryServerInterceptor()),
		)
	}

	// Создание и настройка GRPC-сервера
	a.grpcServer = grpc.NewServer(serverOptions...)
	pb.RegisterRateServiceServer(a.grpcServer, rateServiceServer)
	reflection.Register(a.grpcServer)

	// Запуск GRPC-сервера
	lis, err := net.Listen("tcp", ":"+a.config.GRPCPort)
	if err != nil {
		return fmt.Errorf("failed to listen: %w", err)
	}

	a.logger.Info("Starting GRPC server", zap.String("port", a.config.GRPCPort))

	// Канал для сигналов прерывания
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)

	// Канал для ошибок сервера
	errCh := make(chan error)

	// Запуск сервера в отдельной горутине
	go func() {
		if err := a.grpcServer.Serve(lis); err != nil {
			errCh <- fmt.Errorf("failed to serve: %w", err)
		}
	}()

	// Ожидание сигнала завершения или ошибки
	select {
	case <-quit:
		a.logger.Info("Shutting down server...")
		a.Shutdown(ctx)
	case err := <-errCh:
		a.logger.Error("Server error", zap.Error(err))
		a.Shutdown(ctx)
		return err
	case <-ctx.Done():
		a.logger.Info("Context canceled, shutting down server...")
		a.Shutdown(ctx)
	}

	return nil
}

// Shutdown корректно завершает работу приложения
func (a *App) Shutdown(ctx context.Context) {
	// Graceful shutdown GRPC сервера
	if a.grpcServer != nil {
		a.grpcServer.GracefulStop()
		a.logger.Info("GRPC server successfully shutdown")
	}

	// Закрытие соединения с базой данных
	if a.repo != nil {
		if err := a.repo.Close(); err != nil {
			a.logger.Error("Failed to close repository", zap.Error(err))
		} else {
			a.logger.Info("Database connection closed")
		}
	}

	// Закрытие провайдеров трассировки и метрик
	for _, cleanupFunc := range a.cleanupFuncs {
		if err := cleanupFunc(ctx); err != nil {
			a.logger.Error("Cleanup error", zap.Error(err))
		}
	}
}

// runMigrations запускает миграции базы данных
func (a *App) runMigrations() error {
	db, err := goose.OpenDBWithDriver("pgx", a.config.GetDBConnString())
	if err != nil {
		return err
	}
	defer db.Close()

	// Установка директории с миграциями
	if err := goose.Up(db, "../migrations"); err != nil {
		return err
	}

	a.logger.Info("Database migrations completed successfully")
	return nil
}
