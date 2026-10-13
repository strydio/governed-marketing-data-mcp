package telemetry

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.37.0"
	"go.opentelemetry.io/otel/trace"
)

type ShutdownFunc func(context.Context) error

func Tracer() trace.Tracer {
	return otel.Tracer(TracerName)
}

func InitOTLP(ctx context.Context, serviceName, endpoint string) (ShutdownFunc, error) {
	InitTextMapPropagator()
	if serviceName == "" {
		serviceName = "governed-marketing-data-mcp"
	}
	if endpoint == "" {
		return func(context.Context) error { return nil }, nil
	}
	exporter, err := otlptracegrpc.New(ctx, otlptracegrpc.WithEndpoint(strings.TrimPrefix(endpoint, "http://")), otlptracegrpc.WithInsecure())
	if err != nil {
		return nil, fmt.Errorf("create OTLP trace exporter: %w", err)
	}
	provider := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(resource.NewWithAttributes(semconv.SchemaURL, semconv.ServiceName(serviceName))),
	)
	otel.SetTracerProvider(provider)
	return provider.Shutdown, nil
}

func CampaignIDHash(campaign, market string) string {
	sum := sha256.Sum256([]byte(campaign + ":" + market))
	return hex.EncodeToString(sum[:])[:12]
}
