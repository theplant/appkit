package monitoring

import (
	"strings"
	"testing"

	"github.com/theplant/appkit/log"
)

func TestInvalidInfluxdbConfig(t *testing.T) {
	logger := log.NewNopLogger()
	cases := map[string]string{
		"not absolute url":            "",
		"Unsupported protocol scheme": "localhost:8086/local",
		"not database":                "http://root:password@localhost:8086",
	}

	for reason, config := range cases {
		_, err := NewInfluxdbMonitor(InfluxMonitorConfig(config), logger)

		if err == nil || !strings.Contains(err.Error(), reason) {
			t.Fatalf("no error creating influxdb monitor with config url %s", config)
		}
	}
}

func TestValidInfluxdbConfig(t *testing.T) {
	logger := log.NewNopLogger()
	cases := []string{
		"http://localhost:8086/local",
		"https://localhost:8086/local",
		"https://root@localhost:8086/local",
		"https://:password@localhost:8086/local",
		"https://root:password@localhost:8086/local",
	}

	for _, config := range cases {
		_, err := NewInfluxdbMonitor(InfluxMonitorConfig(config), logger)

		if err != nil {
			t.Fatalf("error creating influxdb monitor with config url %s", config)
		}
	}
}
