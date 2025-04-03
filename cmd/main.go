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
		fmt.Printf("Failed to load config: %v\n", err)
		os.Exit(1)
	}

	// Инициализация логгера
	logger.BuildLogger(readConfig.LogLevel)
	applogger := logger.Logger().Named("main")
	defer applogger.Sync()

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
