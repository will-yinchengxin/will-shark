package core

import (
	"context"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/jaeger"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.10.0"
	"willshark/consts"
	"willshark/utils/logs/logger"
)

var Trace *sdktrace.TracerProvider

type jeagerConfig struct {
	Host    string `yaml:"host,omitempty"`
	Enables bool   `yaml:"enable,omitempty"`
}

func initJaeger() func() {
	cfg, err := GetSingleConfig(CoreConfig, "jaeger", jeagerConfig{})
	if err != nil {
		panic("init jeager failed, jeager config not found")
	}
	jConfig, ok := cfg.(*jeagerConfig)
	if !ok {
		panic("init jeager failed, jeager config incorrect")
	}
	if jConfig.Enables {
		Trace = initConn(jConfig.Host+consts.JaegerURL, consts.APP_NAME+"_"+consts.APP_ID)
		return func() {
			ctx, cancelFunc := context.WithCancel(context.Background())
			Trace.Shutdown(ctx)
			cancelFunc()
			logger.Info("Close Jaeger Sunccess")
		}
	}
	return func() {}
}

func initConn(host, serviceName string) *sdktrace.TracerProvider {
	if host == "" {
		panic("host should not empty")
	}
	if serviceName == "" {
		panic("ServiceName should not empty")
	}

	exp, err := jaeger.New(jaeger.WithCollectorEndpoint(jaeger.WithEndpoint(host)))
	if err != nil {
		panic(err)
	}
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exp),
		sdktrace.WithResource(resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceNameKey.String(serviceName),
			attribute.String(consts.JaegerEnvironment, Environment),
		)),
	)
	otel.SetTracerProvider(tp)
	return tp
}
