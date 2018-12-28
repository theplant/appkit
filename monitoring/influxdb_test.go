package monitoring

import (
	"strings"
	"testing"

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

		expectedCfg      *influxMonitorCfg
		expectedErrorStr string
	}{
		{
			name:   "http scheme",
			config: "http://localhost:8086/local",
			expectedCfg: &influxMonitorCfg{
				Addr:     "http://localhost:8086",
				Username: "",
				Password: "",
				Database: "local",
			},
		},

		{
			name:   "https scheme",
			config: "https://localhost:8086/local",
			expectedCfg: &influxMonitorCfg{
				Addr:     "https://localhost:8086",
				Username: "",
				Password: "",
				Database: "local",
			},
		},

		{
			name:   "has username and no password",
			config: "https://root@localhost:8086/local",
			expectedCfg: &influxMonitorCfg{
				Addr:     "https://localhost:8086",
				Username: "root",
				Password: "",
				Database: "local",
			},
		},

		{
			name:   "no username and has password",
			config: "https://:password@localhost:8086/local",
			expectedCfg: &influxMonitorCfg{
				Addr:     "https://localhost:8086",
				Username: "",
				Password: "password",
				Database: "local",
			},
		},

		{
			name:   "has username and password",
			config: "https://root:password@localhost:8086/local",
			expectedCfg: &influxMonitorCfg{
				Addr:     "https://localhost:8086",
				Username: "root",
				Password: "password",
				Database: "local",
			},
		},

		{
			name:             "no database",
			config:           "https://localhost:8086/",
			expectedErrorStr: "config format error",
		},

		{
			name:             "no scheme",
			config:           "localhost:8086/local",
			expectedErrorStr: "config format error",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			cfg, err := parseConfig(test.config)
			if err != nil {
				errorassert.Equal(t, test.expectedErrorStr, err.Error())
			} else {
				errorassert.Equal(t, test.expectedErrorStr, "")
			}

			errorassert.Equal(t, test.expectedCfg, cfg)
		})
	}
}
