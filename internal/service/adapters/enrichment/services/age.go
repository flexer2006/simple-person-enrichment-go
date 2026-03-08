package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"github.com/flexer2006/case-person-enrichment-go/internal/logger"
	domain "github.com/flexer2006/case-person-enrichment-go/internal/service/domain"

	"go.uber.org/zap"
)

func NewAgeAPIClient(client HTTPClient) *APIClient {
	if client == nil {
		client = &http.Client{}
	}

	return &APIClient{
		baseURL:    "https://api.agify.io",
		httpClient: client,
	}
}

func (c *APIClient) GetAgeByName(ctx context.Context, name string) (int, float64, error) {
	logger.Debug(ctx, "getting age for name", zap.String("name", name))

	if name == "" {
		logger.Error(ctx, "empty name provided for age prediction")
		return 0, 0, ErrEmptyName
	}

	reqURL, err := url.Parse(c.baseURL)
	if err != nil {
		logger.Error(ctx, "failed to parse base URL", zap.Error(err))
		return 0, 0, fmt.Errorf("failed to parse base URL: %w", err)
	}

	q := reqURL.Query()
	q.Add("name", name)
	reqURL.RawQuery = q.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL.String(), nil)
	if err != nil {
		logger.Error(ctx, "failed to create request", zap.Error(err))
		return 0, 0, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		logger.Error(ctx, "failed to execute request", zap.Error(err))
		return 0, 0, fmt.Errorf("failed to execute request: %w", err)
	}
	defer func() {
		if closeErr := resp.Body.Close(); closeErr != nil {
			logger.Warn(ctx, "failed to close response body", zap.Error(closeErr))
		}
	}()

	if resp.StatusCode != http.StatusOK {
		logger.Error(ctx, "API returned non-200 status code",
			zap.Int("status_code", resp.StatusCode))
		return 0, 0, fmt.Errorf("%w: status %d", ErrNon200Response, resp.StatusCode)
	}

	var ageResp domain.AgeResponse
	if err := json.NewDecoder(resp.Body).Decode(&ageResp); err != nil {
		logger.Error(ctx, "failed to decode API response", zap.Error(err))
		return 0, 0, fmt.Errorf("failed to decode API response: %w", err)
	}

	logger.Debug(ctx, "received age from API",
		zap.String("name", name),
		zap.Int("age", ageResp.Age),
		zap.Int("count", ageResp.Count))

	probability := min(float64(ageResp.Count)/1000.0, 1.0)

	return ageResp.Age, probability, nil
}
