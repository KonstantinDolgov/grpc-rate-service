package kucoin

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
)

type KuCoinClient struct {
	baseURL    string
	httpClient *http.Client
	logger     *zap.Logger
	tracer     trace.Tracer
}

type OrderBookResponse struct {
	Code string `json:"code"`
	Data struct {
		Sequence string     `json:"sequence"`
		Time     int64      `json:"time"`
		Bids     [][]string `json:"bids"`
		Asks     [][]string `json:"asks"`
	}
}

func NewKucoinClient(baseUrl string, logger *zap.Logger) *KuCoinClient {
	return &KuCoinClient{
		baseURL: baseUrl,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
		logger: logger,
		tracer: otel.Tracer("kucoin-client"),
	}
}

func (c *KuCoinClient) GetOrderBook(ctx context.Context, symbol string) (float64, float64, time.Time, error) {
	// Создаем спан для трассировки
	ctx, span := c.tracer.Start(ctx, "KuCoin.GetOrderBook",
		trace.WithAttributes(attribute.String("symbol", symbol)))
	defer span.End()

	url := fmt.Sprintf("%s/api/v1/market/orderbook/level2_20?symbol=%s", c.baseURL, symbol)

	c.logger.Debug("Requesting order book from KuCoin",
		zap.String("url", url),
		zap.String("symbol", symbol))

	span.SetAttributes(attribute.String("url", url))

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		c.logger.Error("Failed to create request", zap.Error(err), zap.String("url", url))
		span.SetStatus(codes.Error, "Failed to create request")
		span.RecordError(err)
		return 0, 0, time.Time{}, fmt.Errorf("failed to create request: %w", err)
	}

	// Создаем вложенный спан для HTTP запроса
	ctx, reqSpan := c.tracer.Start(ctx, "KuCoin.HTTPRequest")
	resp, err := c.httpClient.Do(req)
	if err != nil {
		c.logger.Error("Failed to make request", zap.Error(err), zap.String("url", url))
		reqSpan.SetStatus(codes.Error, "Failed to make HTTP request")
		reqSpan.RecordError(err)
		reqSpan.End()

		span.SetStatus(codes.Error, "Failed to make HTTP request")
		span.RecordError(err)
		return 0, 0, time.Time{}, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	reqSpan.SetAttributes(
		attribute.Int("http.status_code", resp.StatusCode),
		attribute.String("http.method", "GET"),
	)
	reqSpan.End()

	if resp.StatusCode != http.StatusOK {
		errMsg := fmt.Sprintf("unexpected status code: %d", resp.StatusCode)
		c.logger.Error("Unexpected status code",
			zap.Int("status_code", resp.StatusCode),
			zap.String("url", url))

		span.SetStatus(codes.Error, errMsg)
		return 0, 0, time.Time{}, fmt.Errorf("%s", errMsg)
	}

	// Создаем вложенный спан для декодирования ответа
	ctx, decodeSpan := c.tracer.Start(ctx, "KuCoin.DecodeResponse")
	var response OrderBookResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		c.logger.Error("Failed to decode response", zap.Error(err))
		decodeSpan.SetStatus(codes.Error, "Failed to decode response")
		decodeSpan.RecordError(err)
		decodeSpan.End()

		span.SetStatus(codes.Error, "Failed to decode response")
		span.RecordError(err)
		return 0, 0, time.Time{}, fmt.Errorf("failed to decode response: %w", err)
	}
	decodeSpan.End()

	if len(response.Data.Asks) == 0 || len(response.Data.Bids) == 0 {
		errMsg := "empty order book data"
		c.logger.Error(errMsg,
			zap.String("symbol", symbol),
			zap.Int("asks_length", len(response.Data.Asks)),
			zap.Int("bids_length", len(response.Data.Bids)))

		span.SetStatus(codes.Error, errMsg)
		return 0, 0, time.Time{}, fmt.Errorf("%s", errMsg)
	}

	// Создаем вложенный спан для парсинга цен
	_, parseSpan := c.tracer.Start(ctx, "KuCoin.ParsePrices")
	// Получаем первые ask и bid цены
	askPrice, err := strconv.ParseFloat(response.Data.Asks[0][0], 64)
	if err != nil {
		c.logger.Error("Failed to parse ask price",
			zap.Error(err),
			zap.String("raw_value", response.Data.Asks[0][0]))

		parseSpan.SetStatus(codes.Error, "Failed to parse ask price")
		parseSpan.RecordError(err)
		parseSpan.End()

		span.SetStatus(codes.Error, "Failed to parse ask price")
		span.RecordError(err)
		return 0, 0, time.Time{}, fmt.Errorf("failed to parse ask price: %w", err)
	}

	bidPrice, err := strconv.ParseFloat(response.Data.Bids[0][0], 64)
	if err != nil {
		c.logger.Error("Failed to parse bid price",
			zap.Error(err),
			zap.String("raw_value", response.Data.Bids[0][0]))

		parseSpan.SetStatus(codes.Error, "Failed to parse bid price")
		parseSpan.RecordError(err)
		parseSpan.End()

		span.SetStatus(codes.Error, "Failed to parse bid price")
		span.RecordError(err)
		return 0, 0, time.Time{}, fmt.Errorf("failed to parse bid price: %w", err)
	}
	parseSpan.End()

	// Преобразуем timestamp из миллисекунд в time.Time в UTC
	timestamp := time.Unix(0, response.Data.Time*int64(time.Millisecond)).UTC()

	c.logger.Debug("Successfully received order book data",
		zap.String("symbol", symbol),
		zap.Float64("ask", askPrice),
		zap.Float64("bid", bidPrice),
		zap.Time("timestamp", timestamp))

	// Добавляем результаты в спан
	span.SetAttributes(
		attribute.Float64("ask", askPrice),
		attribute.Float64("bid", bidPrice),
		attribute.String("timestamp", timestamp.Format(time.RFC3339)),
	)
	span.SetStatus(codes.Ok, "Successfully received order book data")

	return askPrice, bidPrice, timestamp, nil
}
