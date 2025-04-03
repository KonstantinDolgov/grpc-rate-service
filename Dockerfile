# Этап сборки: используем официальный образ Go
FROM golang:1.23-alpine as builder

# Устанавливаем рабочую директорию внутри контейнера
WORKDIR /app

# Устанавливаем goose
RUN go install github.com/pressly/goose/v3/cmd/goose@latest

# Копируем go.mod и go.sum из текущей папки (где находится Dockerfile)
COPY ./go.mod ./go.sum ./

# Скачиваем все зависимости
RUN go mod download

# Копируем все исходники из текущей папки
COPY ./ ./

# Переключаем рабочую директорию на папку с основным файлом
WORKDIR /app/cmd

# Собираем приложение с указанием целевой платформы
RUN go build -o /app/cmd/main

# Начинаем новую стадию сборки на основе минимального образа
FROM alpine:latest

# Устанавливаем Go и необходимые утилиты для выполнения миграций
RUN apk add --no-cache go

# Добавляем исполняемый файл из первой стадии в корневую директорию контейнера
COPY --from=builder /app/cmd/main /main
COPY --from=builder /app/migrations /migrations
COPY .env /app/.env

# Устанавливаем goose в финальном образе
COPY --from=builder /go/bin/goose /usr/local/bin/goose

RUN chmod +x /main

# Открываем порт 50051
EXPOSE 50051

# Запускаем приложение
CMD ["/main"]