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
	configs := []string{
		"",
		"localhost:8086/local",
		"http://root:password@localhost:8086",
	}

	for _, conf := range configs {
		_, err := NewInfluxdbMonitor(InfluxMonitorConfig(conf), logger)
		if err == nil || !strings.Contains(err.Error(), "config format error") {
			t.Fatalf("no error creating influxdb monitor with config url %s", conf)
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

func TestParseConfig(t *testing.T) {
	tests := []struct {
		name string

		config string

		expectedCfg           *influxMonitorCfg
		expectedErrorContains string
	}{
		{
			name:   "http scheme",
			config: "http://localhost:8086/local",
			expectedCfg: &influxMonitorCfg{
				Addr:               "http://localhost:8086",
				Username:           "",
				Password:           "",
				Database:           "local",
				BatchWriteInterval: time.Minute,
			},
		},

		{
			name:   "https scheme",
			config: "https://localhost:8086/local",
			expectedCfg: &influxMonitorCfg{
				Addr:               "https://localhost:8086",
				Username:           "",
				Password:           "",
				Database:           "local",
				BatchWriteInterval: time.Minute,
			},
		},

		{
			name:   "has username and no password",
			config: "https://root@localhost:8086/local",
			expectedCfg: &influxMonitorCfg{
				Addr:               "https://localhost:8086",
				Username:           "root",
				Password:           "",
				Database:           "local",
				BatchWriteInterval: time.Minute,
			},
		},

		{
			name:   "no username and has password",
			config: "https://:password@localhost:8086/local",
			expectedCfg: &influxMonitorCfg{
				Addr:               "https://localhost:8086",
				Username:           "",
				Password:           "password",
				Database:           "local",
				BatchWriteInterval: time.Minute,
			},
		},

		{
			name:   "has username and password",
			config: "https://root:password@localhost:8086/local",
			expectedCfg: &influxMonitorCfg{
				Addr:               "https://localhost:8086",
				Username:           "root",
				Password:           "password",
				Database:           "local",
				BatchWriteInterval: time.Minute,
			},
		},

		{
			name:   "custom batch-write-second-interval",
			config: "https://root:password@localhost:8086/local?batch-write-second-interval=300",
			expectedCfg: &influxMonitorCfg{
				Addr:               "https://localhost:8086",
				Username:           "root",
				Password:           "password",
				Database:           "local",
				BatchWriteInterval: time.Second * 300,
			},
		},

		{
			name:                  "no database",
			config:                "https://localhost:8086/",
			expectedErrorContains: "influxdb config format error",
		},

		{
			name:                  "no scheme",
			config:                "localhost:8086/local",
			expectedErrorContains: "influxdb config format error",
		},

		{
			name:                  "query format error",
			config:                "https://root:password@localhost:8086/local?batch-write-second-interval=%",
			expectedErrorContains: "influxdb config query format error",
		},

		{
			name:                  "query format error",
			config:                "https://root:password@localhost:8086/local?batch-write-second-interval=abc",
			expectedErrorContains: "influxdb config parameter batch-write-second-interval format error",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			cfg, err := parseConfig(test.config)
			if err != nil {
				if !strings.Contains(err.Error(), test.expectedErrorContains) {
					t.Errorf(`expected error "%v", but got "%v"\n`, test.expectedErrorContains, err.Error())
				}
			} else {
				errorassert.Equal(t, test.expectedErrorContains, "")
			}

			errorassert.Equal(t, test.expectedCfg, cfg)
		})
	}
}
