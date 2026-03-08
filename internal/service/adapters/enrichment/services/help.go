package api

import "net/http"

type APIClient struct {
	baseURL    string
	httpClient HTTPClient
}

type HTTPClient interface {
	Do(req *http.Request) (*http.Response, error)
}
