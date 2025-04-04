package kucoin

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

// Вспомогательная функция для создания тестового сервера и клиента
func setupTestServerAndClient(t *testing.T, handler func(w http.ResponseWriter, r *http.Request)) (*KuCoinClient, *httptest.Server) {
	server := httptest.NewServer(http.HandlerFunc(handler))
	logger := zap.NewNop()
	client := NewKucoinClient(server.URL, logger)
	return client, server
}

// Вспомогательная функция для безопасной записи в ResponseWriter
func writeResponse(t *testing.T, w http.ResponseWriter, data []byte) {
	_, err := w.Write(data)
	assert.NoError(t, err, "Failed to write response")
}

func TestGetOrderBook_Success(t *testing.T) {
	// Создаем тестовый сервер
	client, server := setupTestServerAndClient(t, func(w http.ResponseWriter, r *http.Request) {
		// Проверяем, что метод запроса верный
		assert.Equal(t, http.MethodGet, r.Method)
		// Проверяем, что путь запроса верный
		assert.Contains(t, r.URL.String(), "/api/v1/market/orderbook/level2_20")
		// Проверяем, что параметр symbol передан верно
		assert.Contains(t, r.URL.String(), "symbol=BTC-USDT")

		// Возвращаем тестовый ответ
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		writeResponse(t, w, []byte(`{
			"code": "200000",
			"data": {
				"sequence": "1234567890",
				"time": 1617267321123,
				"bids": [
					["40000.0", "1.0", "123456"],
					["39999.0", "0.5", "123457"]
				],
				"asks": [
					["40001.0", "0.8", "123458"],
					["40002.0", "0.3", "123459"]
				]
			}
		}`))
	})
	defer server.Close()

	// Выполняем запрос
	ctx := context.Background()
	ask, bid, timestamp, err := client.GetOrderBook(ctx, "BTC-USDT")

	// Проверяем результаты
	assert.NoError(t, err)
	assert.Equal(t, 40001.0, ask)
	assert.Equal(t, 40000.0, bid)
	// Проверяем что timestamp был преобразован корректно
	expectedTime := time.Unix(0, 1617267321123*int64(time.Millisecond)).UTC()
	assert.Equal(t, expectedTime, timestamp)
}

func TestGetOrderBook_InvalidResponse(t *testing.T) {
	// Создаем тестовый сервер с неправильным форматом JSON
	client, server := setupTestServerAndClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		writeResponse(t, w, []byte(`{ invalid json }`))
	})
	defer server.Close()

	// Выполняем запрос
	ctx := context.Background()
	_, _, _, err := client.GetOrderBook(ctx, "BTC-USDT")

	// Проверяем, что получили ошибку
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to decode response")
}

func TestGetOrderBook_EmptyData(t *testing.T) {
	// Создаем тестовый сервер с пустыми данными
	client, server := setupTestServerAndClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		writeResponse(t, w, []byte(`{
			"code": "200000",
			"data": {
				"sequence": "1234567890",
				"time": 1617267321123,
				"bids": [],
				"asks": []
			}
		}`))
	})
	defer server.Close()

	// Выполняем запрос
	ctx := context.Background()
	_, _, _, err := client.GetOrderBook(ctx, "BTC-USDT")

	// Проверяем, что получили ошибку
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "empty order book data")
}

func TestGetOrderBook_InvalidPrices(t *testing.T) {
	// Создаем тестовый сервер с некорректными ценами
	client, server := setupTestServerAndClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		writeResponse(t, w, []byte(`{
			"code": "200000",
			"data": {
				"sequence": "1234567890",
				"time": 1617267321123,
				"bids": [
					["not-a-number", "1.0", "123456"]
				],
				"asks": [
					["40001.0", "0.8", "123458"]
				]
			}
		}`))
	})
	defer server.Close()

	// Выполняем запрос
	ctx := context.Background()
	_, _, _, err := client.GetOrderBook(ctx, "BTC-USDT")

	// Проверяем, что получили ошибку
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to parse bid price")
}

func TestGetOrderBook_ServerError(t *testing.T) {
	// Создаем тестовый сервер, возвращающий ошибку
	client, server := setupTestServerAndClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		writeResponse(t, w, []byte(`{"code":"500","msg":"Internal Server Error"}`))
	})
	defer server.Close()

	// Выполняем запрос
	ctx := context.Background()
	_, _, _, err := client.GetOrderBook(ctx, "BTC-USDT")

	// Проверяем, что получили ошибку
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unexpected status code: 500")
}

func TestGetOrderBook_NetworkError(t *testing.T) {
	// Создаем клиент с несуществующим URL
	logger := zap.NewNop()
	client := NewKucoinClient("http://localhost:1", logger)
	client.httpClient.Timeout = 100 * time.Millisecond // Уменьшим таймаут для быстрого тестирования

	// Выполняем запрос
	ctx := context.Background()
	_, _, _, err := client.GetOrderBook(ctx, "BTC-USDT")

	// Проверяем, что получили ошибку
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to make request")
}
