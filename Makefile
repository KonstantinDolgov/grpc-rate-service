# Имя приложения и основные переменные
APP_NAME := app
MAIN_PATH := ./cmd
BUILD_DIR := .

# Переменная для линтера
LINTER := golangci-lint

.PHONY: build test docker-build run lint clean docker-up docker-down

# Сборка приложения
build:
	@echo "Сборка приложения $(APP_NAME)..."
	@go build -o $(BUILD_DIR)/$(APP_NAME) $(MAIN_PATH)

# Запуск юнит-тестов
test:
	@echo "Запуск тестов..."
	@go test -v ./...

# Сборка Docker-образа с приложением
docker-build:
	@echo "Сборка Docker-образа..."
	@docker build -t $(APP_NAME) .

# Запуск приложения
run:
	@echo "Запуск приложения..."
	@go run $(MAIN_PATH)/main.go

# Запуск линтера
lint:
	@echo "Запуск линтера..."
	@$(LINTER) run

# Запуск приложения через Docker Compose
docker-up:
	@echo "Запуск приложения в Docker..."
	@docker-compose up -d

# Остановка приложения в Docker Compose
docker-down:
	@echo "Остановка приложения в Docker..."
	@docker-compose down

# Очистка сборки
clean:
	@echo "Очистка сборки..."
	@rm -f $(BUILD_DIR)/$(APP_NAME)