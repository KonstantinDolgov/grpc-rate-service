package kucoin

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"go.uber.org/zap"
)

type KuCoinClient struct {
	baseURL    string
	httpClient *http.Client
	logger     *zap.Logger
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
	}
}

func (c *KuCoinClient) GetOrderBook(ctx context.Context, symbol string) (float64, float64, time.Time, error) {
	url := fmt.Sprintf("%s/api/v1/market/orderbook/level2_20?symbol=%s", c.baseURL, symbol)

	c.logger.Debug("Requesting order book from KuCoin",
		zap.String("url", url),
		zap.String("symbol", symbol))

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		c.logger.Error("Failed to create request", zap.Error(err), zap.String("url", url))
		return 0, 0, time.Time{}, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		c.logger.Error("Failed to make request", zap.Error(err), zap.String("url", url))
		return 0, 0, time.Time{}, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		c.logger.Error("Unexpected status code",
			zap.Int("status_code", resp.StatusCode),
			zap.String("url", url))
		return 0, 0, time.Time{}, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var response OrderBookResponse

	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		c.logger.Error("Failed to decode response", zap.Error(err))
		return 0, 0, time.Time{}, fmt.Errorf("failed to decode response: %w", err)
	}

	if len(response.Data.Asks) == 0 || len(response.Data.Bids) == 0 {
		c.logger.Error("Empty order book data",
			zap.String("symbol", symbol),
			zap.Int("asks_length", len(response.Data.Asks)),
			zap.Int("bids_length", len(response.Data.Bids)))
		return 0, 0, time.Time{}, fmt.Errorf("empty order book data")
	}

	// Получаем первые ask и bid цены
	askPrice, err := strconv.ParseFloat(response.Data.Asks[0][0], 64)
	if err != nil {
		c.logger.Error("Failed to parse ask price",
			zap.Error(err),
			zap.String("raw_value", response.Data.Asks[0][0]))
		return 0, 0, time.Time{}, fmt.Errorf("failed to parse ask price: %w", err)
	}

	bidPrice, err := strconv.ParseFloat(response.Data.Bids[0][0], 64)
	if err != nil {
		c.logger.Error("Failed to parse bid price",
			zap.Error(err),
			zap.String("raw_value", response.Data.Bids[0][0]))
		return 0, 0, time.Time{}, fmt.Errorf("failed to parse bid price: %w", err)
	}

	// Преобразуем timestamp из миллисекунд в time.Time в UTC
	timestamp := time.Unix(0, response.Data.Time*int64(time.Millisecond)).UTC()

	c.logger.Debug("Successfully received order book data",
		zap.String("symbol", symbol),
		zap.Float64("ask", askPrice),
		zap.Float64("bid", bidPrice),
		zap.Time("timestamp", timestamp))

	return askPrice, bidPrice, timestamp, nil
}
