module github.com/theplant/appkit

go 1.13

require (
	github.com/airbrake/gobrake v3.7.4+incompatible
	github.com/aws/aws-sdk-go v1.24.5
	github.com/caio/go-tdigest v2.3.0+incompatible // indirect
	github.com/cenkalti/backoff v2.2.1+incompatible
	github.com/codahale/hdrhistogram v0.0.0-20161010025455-3a0bb77429bd // indirect
	github.com/go-kit/kit v0.9.0
	github.com/goji/httpauth v0.0.0-20160601135302-2da839ab0f4d
	github.com/gorilla/sessions v1.2.0
	github.com/hashicorp/go-multierror v1.0.0
	github.com/hashicorp/vault/api v1.0.4
	github.com/influxdata/influxdb1-client v0.0.0-20190809212627-fc22c7df067e
	github.com/jinzhu/configor v1.1.1
	github.com/jinzhu/gorm v1.9.10
	github.com/jjeffery/errors v1.0.3
	github.com/jjeffery/kv v0.8.1 // indirect
	github.com/kr/logfmt v0.0.0-20140226030751-b84e30acd515
	github.com/kr/pretty v0.1.0 // indirect
	github.com/leesper/go_rng v0.0.0-20190531154944-a612b043e353 // indirect
	github.com/newrelic/go-agent v2.13.0+incompatible
	github.com/opentracing/opentracing-go v1.1.0
	github.com/pborman/uuid v1.2.0
	github.com/pkg/errors v0.8.1
	github.com/rs/cors v1.7.0
	github.com/theplant/testingutils v0.0.0-20190603093022-26d8b4d95c61
	github.com/uber-go/atomic v1.5.0 // indirect
	github.com/uber/jaeger-client-go v2.19.0+incompatible
	github.com/uber/jaeger-lib v2.2.0+incompatible // indirect
	go.uber.org/atomic v1.5.0 // indirect
	golang.org/x/crypto v0.0.0-20190923035154-9ee001bba392
	gonum.org/v1/gonum v0.6.0 // indirect
)

replace github.com/uber-go/atomic v1.5.0 => go.uber.org/atomic v1.5.0
