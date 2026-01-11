package client

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/Daniil-Podruchny/gofermart-loyalty/internal/models"
)

type AccrualClient struct {
	baseURL    string
	httpClient *http.Client
}

type AccrualResponse struct {
	Order   string             `json:"order"`
	Status  models.OrderStatus `json:"status"`
	Accrual *float64           `json:"accrual,omitempty"`
}

func NewAccrualClient(baseURL string) *AccrualClient {
	return &AccrualClient{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

func (c *AccrualClient) GetOrderAccrual(ctx context.Context, orderNumber string) (*AccrualResponse, int, error) {
	url := fmt.Sprintf("%s/api/orders/%s", c.baseURL, orderNumber)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	// Обработка rate limit (429)
	if resp.StatusCode == http.StatusTooManyRequests {
		retryAfter := 60 // по умолчанию 60 секунд
		if retryHeader := resp.Header.Get("Retry-After"); retryHeader != "" {
			if seconds, err := strconv.Atoi(retryHeader); err == nil {
				retryAfter = seconds
			}
		}
		return nil, retryAfter, fmt.Errorf("rate limit exceeded")
	}

	// 204 - заказ не найден в системе начисления
	if resp.StatusCode == http.StatusNoContent {
		return nil, 0, nil
	}

	if resp.StatusCode != http.StatusOK {
		return nil, 0, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var accrualResp AccrualResponse
	if err := json.NewDecoder(resp.Body).Decode(&accrualResp); err != nil {
		return nil, 0, fmt.Errorf("failed to decode response: %w", err)
	}

	return &accrualResp, 0, nil
}
