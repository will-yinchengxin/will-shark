package core

import (
	"context"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/jaeger"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.10.0"
	"will/consts"
)

var Trace *sdktrace.TracerProvider

func initJaeger() func() {
	Trace = initConn(CoreConfig["jaegerhost"].(string)+consts.JaegerURL, consts.APP_NAME+"_"+consts.APP_ID)
	return func() {
		ctx, cancelFunc := context.WithCancel(context.Background())
		Trace.Shutdown(ctx)
		cancelFunc()
		Log.SuccessDefault("Close Jaeger Sunccess")
	}
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
