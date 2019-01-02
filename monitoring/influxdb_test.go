package monitoring

import (
	"strings"
	"testing"
	"time"

	"github.com/theplant/appkit/log"
	"github.com/theplant/testingutils/errorassert"
)

func TestInvalidInfluxdbConfig(t *testing.T) {
	logger := log.NewNopLogger()
	cases := map[string]string{
		"not absolute url":                  "",
		"localhost:8086/local not database": "localhost:8086/local",
		"not database":                      "http://root:password@localhost:8086",
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

func TestParseInfluxMonitorConfig(t *testing.T) {
	tests := []struct {
		name                string
		config              string
		expectedCfg         *influxMonitorCfg
		expectedErrContains string
	}{
		{
			name:   "default batch-write-second-interval and max-cache-events",
			config: "https://root:password@localhost:8086/local",
			expectedCfg: &influxMonitorCfg{
				Scheme:             "https",
				Host:               "localhost:8086",
				Addr:               "https://localhost:8086",
				Username:           "root",
				Password:           "password",
				Database:           "local",
				BatchWriteInterval: defaultBatchWriteInterval,
				MaxCacheEvents:     defaultMaxCacheEvents,
			},
		},

		{
			name:   "custom batch-write-second-interval and max-cache-events",
			config: "http://localhost:8086/local?batch-write-second-interval=30&max-cache-events=5000",
			expectedCfg: &influxMonitorCfg{
				Scheme:             "http",
				Host:               "localhost:8086",
				Addr:               "http://localhost:8086",
				Username:           "",
				Password:           "",
				Database:           "local",
				BatchWriteInterval: time.Second * 30,
				MaxCacheEvents:     5000,
			},
		},

		{
			name:                "batch-write-second-interval format error",
			config:              "http://localhost:8086/local?batch-write-second-interval=abc",
			expectedErrContains: "influxdb config parameter batch-write-second-interval format error",
		},

		{
			name:                "max-cache-events format error",
			config:              "http://localhost:8086/local?max-cache-events=-1",
			expectedErrContains: "influxdb config parameter max-cache-events format error",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			cfg, err := parseInfluxMonitorConfig(InfluxMonitorConfig(test.config))
			if err != nil {
				if !strings.Contains(err.Error(), test.expectedErrContains) {
					t.Errorf(`expected error contains "%v", but got error "%v"\n`, test.expectedErrContains, err.Error())
				}
			} else {
				errorassert.Equal(t, test.expectedErrContains, "")
			}

			errorassert.Equal(t, test.expectedCfg, cfg)
		})
	}
}
