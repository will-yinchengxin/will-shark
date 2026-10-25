package http_client

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

func PostJsonRequest(requestBody, url string) (response []byte, err error) {
	return doJsonRequest(context.Background(), Post, url, requestBody)
}

func GetJsonRequest(requestBody, url string) (response []byte, err error) {
	return doJsonRequest(context.Background(), Get, url, requestBody)
}

func doJsonRequest(ctx context.Context, method, url, requestBody string) ([]byte, error) {
	request, err := http.NewRequestWithContext(ctx, method, url, bytes.NewBufferString(requestBody))
	if err != nil {
		return nil, err
	}
	request.Header.Add(ContentType, Json)
	resp, err := defaultClient.Do(request)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	return io.ReadAll(resp.Body)
}
