package main

import (
	"fmt"
	"net/http"

	opentracing "github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/ext"
	"github.com/theplant/appkit/log"
	"github.com/theplant/appkit/tracing"
)

func main() {
	log := log.Default()

	// Hacky way to configure the global tracer
	closer, _, err := tracing.Tracer(log)
	if err != nil {
		panic(err)
	}
	defer closer.Close()

	span := opentracing.StartSpan("client")
	defer span.Finish()

	httpClient := &http.Client{}
	httpReq, _ := http.NewRequest("GET", "http://localhost:9900/", nil)

	// Transmit the span's TraceContext as HTTP headers on our
	// outbound request.
	opentracing.GlobalTracer().Inject(
		span.Context(),
		opentracing.HTTPHeaders,
		opentracing.HTTPHeadersCarrier(httpReq.Header))

	ext.HTTPMethod.Set(span, httpReq.Method)
	ext.HTTPUrl.Set(span, httpReq.URL.String())
	ext.SpanKind.Set(span, ext.SpanKindRPCClientEnum)

	resp, err := httpClient.Do(httpReq)

	if err != nil {
		ext.Error.Set(span, true)
		span.LogKV("error", err)
	} else {
		if resp.StatusCode >= 400 {
			ext.Error.Set(span, true)
		}

		ext.HTTPStatusCode.Set(span, uint16(resp.StatusCode))
	}
	fmt.Println("response", resp, err)
}
