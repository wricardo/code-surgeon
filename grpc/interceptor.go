package grpc

import (
	"context"
	"log"

	"connectrpc.com/connect"
)

// LoggerInterceptor is a Connect RPC middleware that logs all incoming requests.
func LoggerInterceptor() connect.UnaryInterceptorFunc {
	interceptor := func(next connect.UnaryFunc) connect.UnaryFunc {
		return connect.UnaryFunc(func(
			ctx context.Context,
			req connect.AnyRequest,
		) (connect.AnyResponse, error) {
			if req.Spec().IsClient {
			} else {
				log.Printf("Incoming request: %s", req.Spec().Procedure)
				// log.Printf("Request headers: %v", req.Header())
				// log.Printf("Request message: %v", req.Any())
			}
			res, err := next(ctx, req)
			if err != nil {
				log.Printf("Error: %v", err)
			} else {
				log.Printf("Response: %v", res)
			}
			return res, err
		})
	}
	return connect.UnaryInterceptorFunc(interceptor)

}
