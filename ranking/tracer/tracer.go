package tracer

import (
	"context"

	"github.com/cloudwego/kitex/pkg/klog"
	kitexlogrus "github.com/kitex-contrib/obs-opentelemetry/logging/logrus"
	"github.com/kitex-contrib/obs-opentelemetry/provider"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"
)

var provd provider.OtelProvider

func InitTracer(serviceName string, addr string) {
	klog.SetLogger(kitexlogrus.NewLogger())
	klog.SetLevel(klog.LevelDebug)

	provd = provider.NewOpenTelemetryProvider(
		provider.WithServiceName(serviceName),
		provider.WithExportEndpoint(addr),
		provider.WithInsecure(),
	)
}

func FinitTracer() {
	if provd != nil {
		provd.Shutdown(context.Background())
	}
}

func GetCtxSpan() (context.Context, trace.Span) {
	ctx, span := otel.Tracer("client").Start(context.Background(), "root")
	return ctx, span
}

func GetCtxSpan2(c context.Context) (context.Context, trace.Span) {
	ctx, span := otel.Tracer("client").Start(c, "root")
	return ctx, span
}