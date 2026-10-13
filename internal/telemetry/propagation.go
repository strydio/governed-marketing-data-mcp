package telemetry

import (
	"context"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
)

type MetaCarrier map[string]any

func (c MetaCarrier) Get(key string) string {
	value, ok := c[key].(string)
	if !ok {
		return ""
	}
	return value
}

func (c MetaCarrier) Set(key, value string) {
	c[key] = value
}

func (c MetaCarrier) Keys() []string {
	keys := make([]string, 0, len(c))
	for key := range c {
		keys = append(keys, key)
	}
	return keys
}

func InjectTraceContext(ctx context.Context, meta map[string]any) map[string]any {
	if meta == nil {
		meta = map[string]any{}
	}
	otel.GetTextMapPropagator().Inject(ctx, MetaCarrier(meta))
	return meta
}

func ExtractTraceContext(ctx context.Context, meta map[string]any) context.Context {
	if meta == nil {
		return ctx
	}
	return otel.GetTextMapPropagator().Extract(ctx, MetaCarrier(meta))
}

func InitTextMapPropagator() {
	otel.SetTextMapPropagator(propagation.TraceContext{})
}
