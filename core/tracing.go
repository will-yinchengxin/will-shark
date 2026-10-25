package core

import (
	"context"
	"time"
	"willshark/consts"
	"willshark/utils/logs/logger"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

// tracingConfig 链路追踪配置，使用 OTLP/HTTP 协议上报
// Jaeger(>=1.35)、SkyWalking、阿里云 ARMS、腾讯云 APM 均支持 OTLP
type tracingConfig struct {
	Enable   bool   `mapstructure:"enable"`
	Endpoint string `mapstructure:"endpoint"` // host:port，如 127.0.0.1:4318
	Insecure bool   `mapstructure:"insecure"` // 不使用 TLS
}

// initTracing 未配置或关闭时保持 otel 默认的 noop provider，业务代码无需判空
func initTracing() func() {
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(propagation.TraceContext{}, propagation.Baggage{}))

	cfg := &tracingConfig{}
	if raw, err := GetSingleConfig(CoreConfig, "tracing", tracingConfig{}); err == nil {
		if c, ok := raw.(*tracingConfig); ok {
			cfg = c
		}
	}
	if !cfg.Enable {
		return func() {}
	}
	if cfg.Endpoint == "" {
		panic("init tracing failed, tracing.endpoint should not be empty")
	}

	opts := []otlptracehttp.Option{otlptracehttp.WithEndpoint(cfg.Endpoint)}
	if cfg.Insecure {
		opts = append(opts, otlptracehttp.WithInsecure())
	}
	exp, err := otlptracehttp.New(context.Background(), opts...)
	if err != nil {
		panic("init tracing failed: " + err.Error())
	}
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exp),
		sdktrace.WithResource(resource.NewSchemaless(
			attribute.String("service.name", consts.APP_NAME+"_"+consts.APP_ID),
			attribute.String(consts.JaegerEnvironment, Environment),
		)),
	)
	otel.SetTracerProvider(tp)

	return func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := tp.Shutdown(ctx); err != nil {
			logger.Error("Close Tracer Err: " + err.Error())
			return
		}
		logger.Info("Close Tracer Success")
	}
}
