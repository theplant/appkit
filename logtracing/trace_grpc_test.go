package logtracing

import (
	"context"
	"errors"
	"log"
	"net"
	"testing"

	"github.com/theplant/appkit/logtracing/greeter"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/test/bufconn"
)

type greeterServer struct {
	greeter.UnimplementedGreeterServer
}

func (s *greeterServer) SayHello(ctx context.Context, in *greeter.HelloRequest) (*greeter.HelloReply, error) {
	if in.Name == "It" {
		return nil, errors.New("Run away")
	}
	return &greeter.HelloReply{Message: "Hello " + in.Name}, nil
}

var _grpcServer *grpc.Server

const bufSize = 1024 * 1024

var lis *bufconn.Listener

func bufDialer(context.Context, string) (net.Conn, error) {
	return lis.Dial()
}

func startGreeterServer() {
	lis = bufconn.Listen(bufSize)
	_grpcServer = grpc.NewServer(
		grpc.ChainUnaryInterceptor(
			UnaryServerInterceptor(),
		),
		grpc.ChainStreamInterceptor(
			StreamServerInterceptor(),
		),
	)
	greeter.RegisterGreeterServer(_grpcServer, &greeterServer{})
	go func() {
		if err := _grpcServer.Serve(lis); err != nil {
			log.Fatalf("Server exited with error: %v", err)
		}
	}()
}

func stopGreeterServer() {
	_grpcServer.Stop()
}

func newHelloClient(ctx context.Context) (greeter.GreeterClient, error) {
	conn, err := grpc.DialContext(ctx, "bufnet",
		grpc.WithContextDialer(bufDialer),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithUnaryInterceptor(UnaryClientInterceptor()),
		grpc.WithStreamInterceptor(StreamClientInterceptor()),
	)
	if err != nil {
		return nil, err
	}
	if err != nil {
		return nil, err
	}

	greeterClient := greeter.NewGreeterClient(conn)
	return greeterClient, nil
}

func TestSayHello(t *testing.T) {
	startGreeterServer()
	defer stopGreeterServer()

	ctx := context.Background()
	greeterClient, err := newHelloClient(ctx)
	if err != nil {
		t.Fatal(err)
	}

	_, err = greeterClient.SayHello(ctx, &greeter.HelloRequest{Name: "It"})
	if err == nil {
		t.Fatal("Should fail")
	}
	_, err = greeterClient.SayHello(ctx, &greeter.HelloRequest{Name: "World"})
	if err != nil {
		t.Fatal(err)
	}
}
