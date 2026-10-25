// Package httpclient 带超时、链路透传与响应大小限制的 HTTP 客户端。
package httpclient

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
)

const AuthHeader = "Authorization"

type HTTPClient struct {
	BaseURL string
	Client  *http.Client
	// Token 完整的 Authorization 头内容，如 "Bearer xxx"；为空则不发送
	// 从配置或环境变量注入，禁止写死在代码中
	Token string
}

func NewHTTPClient(baseURL string) *HTTPClient {
	return &HTTPClient{
		BaseURL: baseURL,
		Client:  &http.Client{Timeout: DefaultTimeout},
	}
}

// WithToken 设置 Authorization 头
func (c *HTTPClient) WithToken(token string) *HTTPClient {
	c.Token = token
	return c
}

func (c *HTTPClient) Get(url string) (*http.Response, error) {
	return c.GetCtx(context.Background(), url)
}

func (c *HTTPClient) GetCtx(ctx context.Context, url string) (*http.Response, error) {
	return c.do(ctx, http.MethodGet, url, nil)
}

func (c *HTTPClient) PostJSON(url string, body interface{}) (*http.Response, error) {
	return c.PostJSONCtx(context.Background(), url, body)
}

func (c *HTTPClient) PostJSONCtx(ctx context.Context, url string, body interface{}) (*http.Response, error) {
	return c.doJSON(ctx, http.MethodPost, url, body)
}

func (c *HTTPClient) PutJSON(url string, body interface{}) (*http.Response, error) {
	return c.PutJSONCtx(context.Background(), url, body)
}

func (c *HTTPClient) PutJSONCtx(ctx context.Context, url string, body interface{}) (*http.Response, error) {
	return c.doJSON(ctx, http.MethodPut, url, body)
}

func (c *HTTPClient) Delete(url string) (*http.Response, error) {
	return c.DeleteCtx(context.Background(), url)
}

func (c *HTTPClient) DeleteCtx(ctx context.Context, url string) (*http.Response, error) {
	return c.do(ctx, http.MethodDelete, url, nil)
}

func (c *HTTPClient) doJSON(ctx context.Context, method, url string, body interface{}) (*http.Response, error) {
	jsonData, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}
	return c.do(ctx, method, url, bytes.NewReader(jsonData))
}

func (c *HTTPClient) do(ctx context.Context, method, url string, body io.Reader) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, method, c.BaseURL+url, body)
	if err != nil {
		return nil, err
	}
	if c.Token != "" {
		req.Header.Set(AuthHeader, c.Token)
	}
	if body != nil {
		req.Header.Set(ContentType, JSON)
	}
	injectTrace(ctx, req)
	return c.Client.Do(req)
}

// injectTrace 把 ctx 中的链路信息写入请求头（W3C traceparent），下游服务可以串起同一条链路
func injectTrace(ctx context.Context, req *http.Request) {
	otel.GetTextMapPropagator().Inject(ctx, propagation.HeaderCarrier(req.Header))
}

// MaxResponseBytes HandleResponse 读取的响应体上限，防止异常下游返回超大响应耗尽内存
const MaxResponseBytes = 32 << 20

// HandleResponse 读取并关闭响应体；非 2xx 时返回错误
func HandleResponse(resp *http.Response) ([]byte, error) {
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, MaxResponseBytes))
		return nil, errors.New("HTTP request failed with status code: " + resp.Status)
	}
	b, err := io.ReadAll(io.LimitReader(resp.Body, MaxResponseBytes+1))
	if err != nil {
		return nil, err
	}
	if len(b) > MaxResponseBytes {
		return nil, errors.New("HTTP response body exceeds limit")
	}
	return b, nil
}
