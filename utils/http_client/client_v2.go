package http_client

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"
)

var (
	// X-PS-Flag
	AuthHeader = "Authorization"
	Token      = "Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJ0ZW5hbnRfaWQiOiJ2aWRlb3NvbGFyIiwiaWF0IjoxNTE2MjM5MDIyLCJpc3MiOiJ2aWRlb3NvbGFyIiwiZXhwIjoxNzQ1NjgzMjAwfQ.xDEvDbDTdTZbBxK2dPjNjsBAeqTr45xDCxWiO21bGZw"
)

type HttpClient struct {
	BaseUrl string
	Client  *http.Client
}

func NewHttpClient(baseUrl string) *HttpClient {
	return &HttpClient{
		BaseUrl: baseUrl,
		Client:  &http.Client{},
	}
}

func (h *HttpClient) SetHeader() *HttpClient {
	return h
}

func (c *HttpClient) Get(url string) (*http.Response, error) {
	req, err := http.NewRequest(http.MethodGet, c.BaseUrl+url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Add(AuthHeader, Token)
	return c.Client.Do(req)
}

func (c *HttpClient) PostJSON(url string, body interface{}) (*http.Response, error) {
	jsonData, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequest(http.MethodPost, c.BaseUrl+url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, err
	}
	req.Header.Add(AuthHeader, Token)
	req.Header.Set("Content-Type", "application/json")
	return c.Client.Do(req)
}

func (c *HttpClient) PutJSON(url string, body interface{}) (*http.Response, error) {
	jsonData, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequest(http.MethodPut, c.BaseUrl+url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, err
	}
	req.Header.Add(AuthHeader, Token)
	req.Header.Set("Content-Type", "application/json")
	return c.Client.Do(req)
}

func (c *HttpClient) Delete(url string) (*http.Response, error) {
	req, err := http.NewRequest(http.MethodDelete, c.BaseUrl+url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Add(AuthHeader, Token)
	return c.Client.Do(req)
}

func HandleResponse(resp *http.Response) ([]byte, error) {
	if resp.StatusCode != http.StatusOK {
		return nil, errors.New("HTTP request failed with status code: " + resp.Status)
	}
	defer resp.Body.Close()
	return io.ReadAll(resp.Body)
}
