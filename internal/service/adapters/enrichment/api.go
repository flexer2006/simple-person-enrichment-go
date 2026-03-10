package enrichment

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"github.com/flexer2006/pes-api/internal/service/domain"
	"github.com/flexer2006/pes-api/internal/service/logger"
	"github.com/flexer2006/pes-api/internal/service/ports"

	"go.uber.org/zap"
)

const (
	ageBaseURL         = "https://api.agify.io"
	genderBaseURL      = "https://api.genderize.io"
	nationalityBaseURL = "https://api.nationalize.io"
)

type api struct {
	httpCli                           *http.Client
	ageURL, genderURL, nationalityURL string
}

func NewAPI() ports.API {
	return new(api{
		httpCli:        http.DefaultClient,
		ageURL:         ageBaseURL,
		genderURL:      genderBaseURL,
		nationalityURL: nationalityBaseURL,
	})
}

func (a *api) GetAgeByName(ctx context.Context, name string) (int, float64, error) {
	return predict(a, ctx, name, "age", a.ageURL, func(resp *domain.AgeResponse) (int, float64, error) {
		return resp.Age, min(float64(resp.Count)/1000.0, 1.0), nil
	})
}

func (a *api) GetGenderByName(ctx context.Context, name string) (string, float64, error) {
	return predict(a, ctx, name, "gender", a.genderURL, func(resp *domain.GenderResponse) (string, float64, error) {
		return resp.Gender, resp.Probability, nil
	})
}

func (a *api) GetNationalityByName(ctx context.Context, name string) (string, float64, error) {
	return predict(a, ctx, name, "nationality", a.nationalityURL, func(resp *domain.NationalityResponse) (string, float64, error) {
		if len(resp.Countries) == 0 {
			return "", 0, nil
		}
		mostProbable := resp.Countries[0]
		for _, c := range resp.Countries[1:] {
			if c.Probability > mostProbable.Probability {
				mostProbable = c
			}
		}
		return mostProbable.CountryID, mostProbable.Probability, nil
	})
}

func predict[Resp any, Res any](
	a *api,
	ctx context.Context,
	name, kind, baseURL string,
	mapper func(*Resp) (Res, float64, error),
) (res Res, prob float64, err error) {
	logger.Debug(ctx, "getting "+kind+" for name", zap.String("name", name))
	if name == "" {
		logger.Error(ctx, "empty name provided for prediction")
		err = domain.ErrEmptyName
		return
	}
	reqURL, errp := url.Parse(baseURL)
	if errp != nil {
		logger.Error(ctx, "failed to parse base URL", zap.Error(errp))
		err = fmt.Errorf("failed to parse base URL: %w", errp)
		return
	}
	q := reqURL.Query()
	q.Add("name", name)
	reqURL.RawQuery = q.Encode()
	req, errp := http.NewRequestWithContext(ctx, http.MethodGet, reqURL.String(), nil)
	if errp != nil {
		logger.Error(ctx, "failed to create request", zap.Error(errp))
		err = fmt.Errorf("failed to create request: %w", errp)
		return
	}
	//nolint:gosec
	respHTTP, errp := a.httpCli.Do(req)
	if errp != nil {
		logger.Error(ctx, "request execution failed", zap.Error(errp))
		err = fmt.Errorf("failed to execute request: %w", errp)
		return
	}
	defer func() {
		if cerr := respHTTP.Body.Close(); cerr != nil {
			logger.Warn(ctx, "failed to close response body", zap.Error(cerr))
		}
	}()
	if respHTTP.StatusCode != http.StatusOK {
		logger.Error(ctx, "API returned non-200 status code", zap.Int("status_code", respHTTP.StatusCode))
		err = fmt.Errorf("%w: status %d", domain.ErrNon200Response, respHTTP.StatusCode)
		return
	}
	resp := new(Resp)
	if errp = json.NewDecoder(respHTTP.Body).Decode(resp); errp != nil {
		logger.Error(ctx, "failed to decode response body", zap.Error(errp))
		err = fmt.Errorf("failed to decode API response: %w", errp)
		return
	}
	res, prob, err = mapper(resp)
	return
}
