package logtracing

import (
	"context"
	"fmt"
	"path"

	grpc_middleware "github.com/grpc-ecosystem/go-grpc-middleware"
	"google.golang.org/grpc"
	"google.golang.org/grpc/status"
)

// UnaryClientInterceptor returns a new unary client interceptor that optionally logs the execution of external gRPC calls.
func UnaryClientInterceptor() grpc.UnaryClientInterceptor {
	return func(ctx context.Context, fullMethod string, req, reply interface{}, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) (err error) {
		service, method := parseGRPCFullMethod(fullMethod)
		ctx, _ = StartSpan(ctx, grpcCallName(service, method))
		defer func() {
			AppendKVs(ctx, "grpc.code", status.Code(err).String())
			EndSpan(ctx, err)
		}()
		AppendKVs(ctx, GRPCClientKVs(service, method)...)

		return invoker(ctx, fullMethod, req, reply, cc, opts...)
	}
}

// StreamClientInterceptor returns a new streaming client interceptor that optionally logs the execution of external gRPC calls.
func StreamClientInterceptor() grpc.StreamClientInterceptor {
	return func(ctx context.Context, desc *grpc.StreamDesc, cc *grpc.ClientConn, fullMethod string, streamer grpc.Streamer, opts ...grpc.CallOption) (cs grpc.ClientStream, err error) {
		service, method := parseGRPCFullMethod(fullMethod)
		ctx, _ = StartSpan(ctx, grpcCallName(service, method))
		defer func() {
			AppendKVs(ctx, "grpc.code", status.Code(err).String())
			EndSpan(ctx, err)
		}()
		AppendKVs(ctx, GRPCClientKVs(service, method)...)

		return streamer(ctx, desc, cc, fullMethod, opts...)
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

func grpcCallName(service, method string) string {
	return fmt.Sprintf("%s.call(%s)", service, method)
}

func UnaryServerInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp interface{}, err error) {
		service, method := parseGRPCFullMethod(info.FullMethod)
		ctx, _ = StartSpan(ctx, grpcServeName(service, method))
		defer func() {
			AppendKVs(ctx, "grpc.code", status.Code(err).String())
			EndSpan(ctx, err)
		}()
		AppendKVs(ctx, GRPCServerKVs(service, method)...)

		return handler(ctx, req)
	}
}

func StreamServerInterceptor() grpc.StreamServerInterceptor {
	return func(srv interface{}, stream grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) (err error) {
		service, method := parseGRPCFullMethod(info.FullMethod)
		ctx, _ := StartSpan(stream.Context(), grpcServeName(service, method))
		defer func() {
			AppendKVs(ctx, "grpc.code", status.Code(err).String())
			EndSpan(ctx, err)
		}()
		AppendKVs(ctx, GRPCClientKVs(service, method)...)

		wrapped := grpc_middleware.WrapServerStream(stream)
		wrapped.WrappedContext = ctx

		return handler(srv, wrapped)
	}
}

func GRPCServerKVs(service, method string) []interface{} {
	return []interface{}{
		"span.type", "grpc",
		"span.role", "server",
		"grpc.service", service,
		"grpc.method", method,
	}
}

func grpcServeName(service, method string) string {
	return fmt.Sprintf("%s.serve(%s)", service, method)
}

func parseGRPCFullMethod(fullMethodString string) (service, method string) {
	return path.Dir(fullMethodString)[1:], path.Base(fullMethodString)
}
