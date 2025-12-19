package mid

import (
	"context"
	"net/http"

	"github.com/jcpaschoal/spi-exata/business/sdk/web"
	"github.com/jcpaschoal/spi-exata/foundatiton/otel"
	"go.opentelemetry.io/otel/trace"
)

func Otel(tracer trace.Tracer) web.MidFunc {
	m := func(next web.HandlerFunc) web.HandlerFunc {
		h := func(ctx context.Context, r *http.Request) web.Encoder {
			ctx = otel.InjectTracing(ctx, tracer)

			return next(ctx, r)
		}

		return h
	}

	return m
}
