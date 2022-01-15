package logtracing

import (
	"net/rpc"
	"regexp"
)

func XMLRPCClientKVs(method, serverURL string, err error) []interface{} {
	kvs := []interface{}{
		"span.type", "xmlrpc",
		"span.role", "client",
		"xmlrpc.method", method,
		"http.method", "post",
		"http.url", serverURL,
	}
	if fault, ok := AssertFaultError(err); ok {
		kvs = append(kvs, "xmlrpc.fault_code", fault.Code, "xmlrpc.fault_string", fault.String)
	}
	return kvs
}

// Refer https://github.com/theplant/xmlrpc/blob/0141d8e3170ddf9104c2158f45fc5632dc297513/response.go#L20
var faultErrorRegexp = regexp.MustCompile(`^Fault\((\d+)\): (.+)$`)

type FaultError struct {
	Code   string
	String string
}

func AssertFaultError(err error) (*FaultError, bool) {
	serverErr, ok := err.(rpc.ServerError)
	if !ok {
		return nil, false
	}

	sub := faultErrorRegexp.FindAllStringSubmatch(serverErr.Error(), -1)
	if len(sub) != 1 {
		return nil, false
	}
	if len(sub[0]) != 3 {
		return nil, false
	}

	return &FaultError{
		Code:   sub[0][1],
		String: sub[0][2],
	}, true
}
