package main

import (
	"context"
	"fmt"
	"os"

	"go.uber.org/zap"
	"studentgit.kata.academy/KonstantinDolgov/grpc-rate-service/config"
	"studentgit.kata.academy/KonstantinDolgov/grpc-rate-service/internal/app"
	"studentgit.kata.academy/KonstantinDolgov/grpc-rate-service/pkg/logger"
)

func main() {
	// Загрузка конфигурации
	readConfig, err := config.ReadConfig()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load config: %v\n", err)
		os.Exit(1)
	}

	// Инициализация логгера
	logger.BuildLogger(readConfig.LogLevel)
	applogger := logger.Logger().Named("main")
	defer func() {
		if err := applogger.Sync(); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to build logger: %v\n", err)
			os.Exit(1)
		}
	}()

	// Логирование информации о конфигурации телеметрии
	if readConfig.EnableTracing {
		applogger.Info("OpenTelemetry tracing enabled",
			zap.String("otlp_endpoint", readConfig.OTLPEndpoint))
	} else {
		applogger.Info("OpenTelemetry tracing disabled")
	}

	if readConfig.EnableMetrics {
		applogger.Info("Prometheus metrics enabled",
			zap.String("metrics_http_addr", readConfig.MetricsHTTPAddr))
	} else {
		applogger.Info("Prometheus metrics disabled")
	}

	// Создание и инициализация приложения
	application, err := app.NewApp(readConfig, applogger)
	if err != nil {
		applogger.Fatal("Failed to initialize application", zap.Error(err))
		os.Exit(1)
	}

	// Запуск приложения
	ctx := context.Background()
	if err := application.Run(ctx); err != nil {
		applogger.Fatal("Application error", zap.Error(err))
		os.Exit(1)
	}
}
