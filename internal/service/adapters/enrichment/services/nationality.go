package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"github.com/flexer2006/case-person-enrichment-go/internal/service/domain"
	logger "github.com/flexer2006/case-person-enrichment-go/internal/utilities"
	"go.uber.org/zap"
)

func NewNationalityAPIClient(client HTTPClient) *APIClient {
	if client == nil {
		client = &http.Client{}
	}
	return &APIClient{
		baseURL:    "https://api.nationalize.io",
		httpClient: client,
	}
}

func (c *APIClient) GetNationalityByName(ctx context.Context, name string) (string, float64, error) {
	logger.Debug(ctx, "getting nationality for name", zap.String("name", name))
	if name == "" {
		logger.Error(ctx, "empty name provided for nationality prediction")
		return "", 0, domain.ErrEmptyName
	}
	reqURL, err := url.Parse(c.baseURL)
	if err != nil {
		logger.Error(ctx, "failed to parse base URL", zap.Error(err))
		return "", 0, fmt.Errorf("failed to parse base URL: %w", err)
	}
	q := reqURL.Query()
	q.Add("name", name)
	reqURL.RawQuery = q.Encode()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL.String(), nil)
	if err != nil {
		logger.Error(ctx, "failed to create request", zap.Error(err))
		return "", 0, fmt.Errorf("failed to create request: %w", err)
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		logger.Error(ctx, "failed to execute request", zap.Error(err))
		return "", 0, fmt.Errorf("failed to execute request: %w", err)
	}
	defer func() {
		if closeErr := resp.Body.Close(); closeErr != nil {
			logger.Warn(ctx, "failed to close response body", zap.Error(closeErr))
		}
	}()
	if resp.StatusCode != http.StatusOK {
		logger.Error(ctx, "API returned non-200 status code",
			zap.Int("status_code", resp.StatusCode))
		return "", 0, fmt.Errorf("%w: status %d", domain.ErrNon200Response, resp.StatusCode)
	}
	var nationalityResp domain.NationalityResponse
	if err := json.NewDecoder(resp.Body).Decode(&nationalityResp); err != nil {
		logger.Error(ctx, "failed to decode API response", zap.Error(err))
		return "", 0, fmt.Errorf("failed to decode API response: %w", err)
	}
	if len(nationalityResp.Countries) == 0 {
		logger.Debug(ctx, "no nationality data found for name", zap.String("name", name))
		return "", 0, nil
	}
	mostProbableCountry := nationalityResp.Countries[0]
	for _, country := range nationalityResp.Countries {
		if country.Probability > mostProbableCountry.Probability {
			mostProbableCountry = country
		}
	}
	logger.Debug(ctx, "received nationality from API",
		zap.String("name", name),
		zap.String("country_id", mostProbableCountry.CountryID),
		zap.Float64("probability", mostProbableCountry.Probability))
	return mostProbableCountry.CountryID, mostProbableCountry.Probability, nil
}
