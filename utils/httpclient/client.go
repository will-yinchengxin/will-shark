package httpclient

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"time"
)

// DefaultTimeout 默认整体超时，防止下游无响应时 goroutine 永久阻塞
const DefaultTimeout = 10 * time.Second

var defaultClient = &http.Client{Timeout: DefaultTimeout}

func PostJSONRequest(requestBody, url string) (response []byte, err error) {
	return doJSONRequest(context.Background(), Post, url, requestBody)
}

func GetJSONRequest(requestBody, url string) (response []byte, err error) {
	return doJSONRequest(context.Background(), Get, url, requestBody)
}

func doJSONRequest(ctx context.Context, method, url, requestBody string) ([]byte, error) {
	request, err := http.NewRequestWithContext(ctx, method, url, bytes.NewBufferString(requestBody))
	if err != nil {
		return nil, err
	}
	request.Header.Add(ContentType, JSON)
	injectTrace(ctx, request)
	resp, err := defaultClient.Do(request)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	return io.ReadAll(io.LimitReader(resp.Body, MaxResponseBytes))
}
