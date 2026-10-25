package core

import (
	"context"
	"willshark/consts"
	"willshark/envconfig/config"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

// NewTracing 初始化链路追踪（OTLP/HTTP），返回用于 flush 并关闭的函数。
// 未开启时保持 otel 默认的 noop provider，业务代码无需判空。
// Jaeger(>=1.35)、SkyWalking、阿里云 ARMS、腾讯云 APM 均支持 OTLP。
func NewTracing(ctx context.Context, conf config.Tracing, app config.App) (func(context.Context) error, error) {
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(propagation.TraceContext{}, propagation.Baggage{}))
	if !conf.Enable {
		return func(context.Context) error { return nil }, nil
	}

	opts := []otlptracehttp.Option{otlptracehttp.WithEndpoint(conf.Endpoint)}
	if conf.Insecure {
		opts = append(opts, otlptracehttp.WithInsecure())
	}
	exp, err := otlptracehttp.New(ctx, opts...)
	if err != nil {
		return nil, err
	}
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exp),
		sdktrace.WithSampler(sdktrace.ParentBased(sdktrace.TraceIDRatioBased(conf.SampleRatio))),
		sdktrace.WithResource(resource.NewSchemaless(
			attribute.String("service.name", app.Name+"_"+app.ID),
			attribute.String(consts.JaegerEnvironment, app.Env),
		)),
	)
	otel.SetTracerProvider(tp)
	return tp.Shutdown, nil
}
