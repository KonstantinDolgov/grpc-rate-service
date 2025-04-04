# grpc-rate-service

GRPC-сервис для получения курса USDT с биржи KuCoin и сохранения данных в PostgreSQL.

## Описание

Сервис предоставляет следующие возможности:
- Получение курса USDT с биржи KuCoin через метод `GetRates`
- Автоматическое сохранение курса в базе данных PostgreSQL
- Проверка работоспособности сервиса через метод `HealthCheck`
- Graceful shutdown при получении сигнала завершения

## Требования

- Go 1.22 или выше
- PostgreSQL 12 или выше
- Docker и Docker Compose
- golangci-lint (для запуска линтера)
- goose (для выполнения миграций)
- protoc (для генерации gRPC кода, опционально)

## Установка и запуск

### С использованием Docker

1. Клонируйте репозиторий:
```bash
git clone git@github.com:KonstantinDolgov/grpc-rate-service.git
cd grpc-rate-service
```

2. Запустите сервис с помощью Docker Compose:
```bash
make docker-up
```

## Команды Makefile

- `make build` - сборка приложения
- `make test` - запуск unit-тестов
- `make docker-build` - сборка Docker-образа
- `make run` - запуск приложения
- `make lint` - запуск линтера
- `make docker-up` - запуск приложения через Docker Compose
- `make docker-down` - остановка приложения в Docker Compose
- `make clean` - Очистка сборки
## Конфигурация

Сервис можно настроить с помощью переменных окружения или флагов командной строки:

| Переменная окружения | Флаг командной строки | Описание                   | Значение по умолчанию  |
|----------------------|----------------------|----------------------------|------------------------|
| GRPC_PORT            | --grpc-port          | Порт GRPC сервера          | 50051                  |
| DB_HOST              | --db-host            | Хост базы данных           | localhost              |
| DB_PORT              | --db-port            | Порт базы данных           | 5432                   |
| DB_USER              | --db-user            | Пользователь базы данных   | postgres               |
| DB_PASSWORD          | --db-password        | Пароль базы данных         | postgres               |
| DB_NAME              | --db-name            | Имя базы данных            | rateDB                 |
| DB_SSLMODE           | --db-sslmode         | Режим SSL базы данных      | disable                |
| KUCOIN_BASE_URL      | --kucoin-base-url    | Базовый URL API KuCoin     | https://api.kucoin.com |

## Использование gRPC-клиента

Для работы с сервисом можно использовать любой gRPC-клиент, например, [grpcurl](https://github.com/fullstorydev/grpcurl):

```bash
# Получение курса USDT
grpcurl -plaintext -d '{"symbol": "BTC-USDT"}' localhost:50051 rate_service.v1.RateService/GetRates

# Проверка работоспособности сервиса
grpcurl -plaintext localhost:50051 rate_service.v1.RateService/HealthCheck
```