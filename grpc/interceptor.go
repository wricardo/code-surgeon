package grpc

import (
	"context"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"connectrpc.com/connect"
)

// LoggerInterceptor is a Connect RPC middleware that logs all incoming requests.
func LoggerInterceptor() connect.UnaryInterceptorFunc {
	interceptor := func(next connect.UnaryFunc) connect.UnaryFunc {
		return connect.UnaryFunc(func(
			ctx context.Context,
			req connect.AnyRequest,
		) (connect.AnyResponse, error) {
			x := log.Debug()
			defer x.Msg("GRPC Request")
			if req.Spec().IsClient {
			} else {
				if log.Logger.GetLevel() == zerolog.TraceLevel {
					x.Any("request", req.Any())
				}
				// log.Printf("Request headers: %v", req.Header())
				// log.Printf("Request message: %v", req.Any())
			}
			res, err := next(ctx, req)
			if err != nil {
				x.Err(err)
			} else {
				if log.Logger.GetLevel() == zerolog.TraceLevel {
					x.Any("response", res)
				}
			}
			return res, err
		})
	}
	return connect.UnaryInterceptorFunc(interceptor)

}
