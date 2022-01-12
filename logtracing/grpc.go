package logtracing

import (
	"context"
	"fmt"
	"path"

	"google.golang.org/grpc"
	"google.golang.org/grpc/status"
)

// UnaryClientInterceptor returns a new unary client interceptor that optionally logs the execution of external gRPC calls.
func UnaryClientInterceptor() grpc.UnaryClientInterceptor {
	return func(ctx context.Context, method string, req, reply interface{}, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) (err error) {
		requestService, requestMethod := parseGRPCFullMethod(method)
		ctx, _ = StartSpan(ctx, grpcRequestName(requestService, requestMethod))
		defer func() {
			if err != nil {
				AppendKVs(ctx, "grpc.code", status.Code(err).String())
			}
			EndSpan(ctx, err)
		}()
		AppendKVs(ctx, GRPCClientKVs(requestService, requestMethod)...)

		return invoker(ctx, method, req, reply, cc, opts...)
	}
}

// StreamClientInterceptor returns a new streaming client interceptor that optionally logs the execution of external gRPC calls.
func StreamClientInterceptor() grpc.StreamClientInterceptor {
	return func(ctx context.Context, desc *grpc.StreamDesc, cc *grpc.ClientConn, method string, streamer grpc.Streamer, opts ...grpc.CallOption) (cs grpc.ClientStream, err error) {
		requestService, requestMethod := parseGRPCFullMethod(method)
		ctx, _ = StartSpan(ctx, grpcRequestName(requestService, requestMethod))
		defer func() {
			if err != nil {
				AppendKVs(ctx, "grpc.code", status.Code(err).String())
			}
			EndSpan(ctx, err)
		}()
		AppendKVs(ctx, GRPCClientKVs(requestService, requestMethod)...)

		return streamer(ctx, desc, cc, method, opts...)
	}
}

func GRPCClientKVs(service, method string) []interface{} {
	return []interface{}{
		"span.type", "grpc",
		"span.role", "client",
		"grpc.service", service,
		"grpc.method", method,
	}
}

func grpcRequestName(service, method string) string {
	return fmt.Sprintf("%s.call(%s)", service, method)
}

func parseGRPCFullMethod(fullMethodString string) (service, method string) {
	return path.Dir(fullMethodString)[1:], path.Base(fullMethodString)
}
