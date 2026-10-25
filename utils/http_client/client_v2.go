package http_client

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
)

const AuthHeader = "Authorization"

type HttpClient struct {
	BaseUrl string
	Client  *http.Client
	// Token 完整的 Authorization 头内容，如 "Bearer xxx"；为空则不发送
	// 从配置或环境变量注入，禁止写死在代码中
	Token string
}

func NewHttpClient(baseUrl string) *HttpClient {
	return &HttpClient{
		BaseUrl: baseUrl,
		Client:  &http.Client{Timeout: DefaultTimeout},
	}
}

// WithToken 设置 Authorization 头
func (c *HttpClient) WithToken(token string) *HttpClient {
	c.Token = token
	return c
}

func (c *HttpClient) Get(url string) (*http.Response, error) {
	return c.GetCtx(context.Background(), url)
}

func (c *HttpClient) GetCtx(ctx context.Context, url string) (*http.Response, error) {
	return c.do(ctx, http.MethodGet, url, nil)
}

func (c *HttpClient) PostJSON(url string, body interface{}) (*http.Response, error) {
	return c.PostJSONCtx(context.Background(), url, body)
}

func (c *HttpClient) PostJSONCtx(ctx context.Context, url string, body interface{}) (*http.Response, error) {
	return c.doJSON(ctx, http.MethodPost, url, body)
}

func (c *HttpClient) PutJSON(url string, body interface{}) (*http.Response, error) {
	return c.PutJSONCtx(context.Background(), url, body)
}

func (c *HttpClient) PutJSONCtx(ctx context.Context, url string, body interface{}) (*http.Response, error) {
	return c.doJSON(ctx, http.MethodPut, url, body)
}

func (c *HttpClient) Delete(url string) (*http.Response, error) {
	return c.DeleteCtx(context.Background(), url)
}

func (c *HttpClient) DeleteCtx(ctx context.Context, url string) (*http.Response, error) {
	return c.do(ctx, http.MethodDelete, url, nil)
}

func (c *HttpClient) doJSON(ctx context.Context, method, url string, body interface{}) (*http.Response, error) {
	jsonData, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}
	return c.do(ctx, method, url, bytes.NewReader(jsonData))
}

func (c *HttpClient) do(ctx context.Context, method, url string, body io.Reader) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, method, c.BaseUrl+url, body)
	if err != nil {
		return nil, err
	}
	if c.Token != "" {
		req.Header.Set(AuthHeader, c.Token)
	}
	if body != nil {
		req.Header.Set(ContentType, Json)
	}
	return c.Client.Do(req)
}

// HandleResponse 读取并关闭响应体；非 200 时返回错误
func HandleResponse(resp *http.Response) ([]byte, error) {
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		_, _ = io.Copy(io.Discard, resp.Body)
		return nil, errors.New("HTTP request failed with status code: " + resp.Status)
	}
	return io.ReadAll(resp.Body)
}
