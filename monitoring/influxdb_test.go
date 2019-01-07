package monitoring

import (
	"errors"
	"strings"
	"testing"
	"time"

	influxdb "github.com/influxdata/influxdb/client/v2"
	"github.com/theplant/appkit/log"
	"github.com/theplant/testingutils/errorassert"
	"github.com/theplant/testingutils/fatalassert"
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
			name:   "default batch-write-second-interval, cache-events, max-cache-events",
			config: "https://root:password@localhost:8086/local",
			expectedCfg: &influxMonitorCfg{
				Scheme:             "https",
				Host:               "localhost:8086",
				Addr:               "https://localhost:8086",
				Username:           "root",
				Password:           "password",
				Database:           "local",
				BatchWriteInterval: defaultBatchWriteInterval,
				CacheEvents:        defaultCacheEvents,
				MaxCacheEvents:     defaultMaxCacheEvents,
			},
		},

		{
			name:   "custom batch-write-second-interval, cache-events, max-cache-events",
			config: "http://localhost:8086/local?batch-write-second-interval=30&cache-events=1000&max-cache-events=5000",
			expectedCfg: &influxMonitorCfg{
				Scheme:             "http",
				Host:               "localhost:8086",
				Addr:               "http://localhost:8086",
				Username:           "",
				Password:           "",
				Database:           "local",
				BatchWriteInterval: time.Second * 30,
				CacheEvents:        1000,
				MaxCacheEvents:     5000,
			},
		},

		{
			name:                "batch-write-second-interval format error",
			config:              "http://localhost:8086/local?batch-write-second-interval=abc",
			expectedErrContains: "influxdb config parameter batch-write-second-interval format error",
		},

		{
			name:                "cache-events format error",
			config:              "http://localhost:8086/local?cache-events=abc",
			expectedErrContains: "influxdb config parameter cache-events format error",
		},

		{
			name:                "max-cache-events format error",
			config:              "http://localhost:8086/local?max-cache-events=-1",
			expectedErrContains: "influxdb config parameter max-cache-events format error",
		},

		{
			name:                "cache-events > max-cache-events error",
			config:              "http://localhost:8086/local?cache-events=1001&max-cache-events=1000",
			expectedErrContains: "cache-events can not be greater than max-cache-events",
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

func newMonitor(client influxdb.Client, cacheEvents int, maxCacheEvents int) *influxdbMonitor {
	monitor := &influxdbMonitor{
		database: "test_database",
		client:   client,
		logger:   log.NewNopLogger(),

		pointChan:          make(chan *influxdb.Point),
		batchWriteInterval: time.Second * 1,
		cacheEvent:         newCacheEvent(cacheEvents, maxCacheEvents),
	}
	go monitor.batchWriteTicker()

	return monitor
}

func insertRecords(monitor Monitor, callTimes int) {
	for i := 0; i < callTimes; i++ {
		monitor.InsertRecord("measurement", "value", nil, nil, time.Now())
	}
	time.Sleep(time.Millisecond * 100)
}

func TestInfluxdbBatchWrite(t *testing.T) {
	var callWriteCount int
	var pointsLengthList []int
	mockedClient := &ClientMock{
		WriteFunc: func(bp influxdb.BatchPoints) error {
			callWriteCount = callWriteCount + 1
			pointsLengthList = append(pointsLengthList, len(bp.Points()))
			return nil
		},
	}

	monitor := newMonitor(mockedClient, 5000, 10000)

	insertRecords(monitor, 4000)

	// not reach CacheEvents
	fatalassert.Equal(t, 0, callWriteCount)

	insertRecords(monitor, 1000)

	// reach CacheEvents
	fatalassert.Equal(t, 1, callWriteCount)
	fatalassert.Equal(t, []int{5000}, pointsLengthList)

	// reach CacheEvents twice and remain 1000
	insertRecords(monitor, 11000)

	fatalassert.Equal(t, 3, callWriteCount)
	fatalassert.Equal(t, []int{5000, 5000, 5000}, pointsLengthList)

	insertRecords(monitor, 1000)

	// not reach CacheEvents, len(points) = 2000
	fatalassert.Equal(t, 3, callWriteCount)

	time.Sleep(time.Second * 1)

	// ticker is triggered
	fatalassert.Equal(t, 4, callWriteCount)
	fatalassert.Equal(t, []int{5000, 5000, 5000, 2000}, pointsLengthList)
}

func TestInfluxdbBatchWrite__WriteFailed(t *testing.T) {
	var callWriteCount int
	var pointsLengthList []int
	writeError := errors.New("write error")
	mockedClient := &ClientMock{
		WriteFunc: func(bp influxdb.BatchPoints) error {
			callWriteCount = callWriteCount + 1
			pointsLengthList = append(pointsLengthList, len(bp.Points()))
			return writeError
		},
	}

	monitor := newMonitor(mockedClient, 5000, 16000)

	insertRecords(monitor, 5000)

	// CurrentCacheEvents = 10000
	// len(points) = 5000

	fatalassert.Equal(t, 1, callWriteCount)
	fatalassert.Equal(t, []int{5000}, pointsLengthList)

	insertRecords(monitor, 10000)

	// CurrentCacheEvents = 16000
	// len(points) = 15000

	fatalassert.Equal(t, 3, callWriteCount)
	fatalassert.Equal(t, []int{5000, 10000, 15000}, pointsLengthList)

	insertRecords(monitor, 100)

	// CurrentCacheEvents = 16000
	// len(points) = 15100

	fatalassert.Equal(t, 3, callWriteCount)

	insertRecords(monitor, 2000)

	// CurrentCacheEvents = 16000
	// len(points) = 16000
	// 16000 points is lost

	fatalassert.Equal(t, 4, callWriteCount)
	fatalassert.Equal(t, []int{5000, 10000, 15000, 16000}, pointsLengthList)

	time.Sleep(time.Second * 1)

	// ticker is triggered
	//
	// CurrentCacheEvents = 16000
	// len(points) = 1100

	fatalassert.Equal(t, 5, callWriteCount)
	fatalassert.Equal(t, []int{5000, 10000, 15000, 16000, 1100}, pointsLengthList)

	insertRecords(monitor, 10000)

	// not reach CurrentCacheEvents(16000)
	// len(points) = 11100
	// not trigger batch write

	fatalassert.Equal(t, 5, callWriteCount)

	// the influxdb is recover to normal

	*mockedClient = ClientMock{
		WriteFunc: func(bp influxdb.BatchPoints) error {
			callWriteCount = callWriteCount + 1
			pointsLengthList = append(pointsLengthList, len(bp.Points()))
			return nil
		},
	}

	insertRecords(monitor, 4900)

	// CurrentCacheEvents = 5000
	// len(points) = 16000

	fatalassert.Equal(t, 6, callWriteCount)
	fatalassert.Equal(t, []int{5000, 10000, 15000, 16000, 1100, 16000}, pointsLengthList)

	insertRecords(monitor, 5000)

	fatalassert.Equal(t, 7, callWriteCount)
	fatalassert.Equal(t, []int{5000, 10000, 15000, 16000, 1100, 16000, 5000}, pointsLengthList)
}

func TestInfluxdbBatchWrite__WriteFailed__CacheEventsAndMaxCacheEventsIsDefault(t *testing.T) {
	var callWriteCount int
	var pointsLengthList []int
	writeError := errors.New("write error")
	mockedClient := &ClientMock{
		WriteFunc: func(bp influxdb.BatchPoints) error {
			callWriteCount = callWriteCount + 1
			pointsLengthList = append(pointsLengthList, len(bp.Points()))
			return writeError
		},
	}

	monitor := newMonitor(mockedClient, 5000, 10000)

	insertRecords(monitor, 9000)

	fatalassert.Equal(t, 1, callWriteCount)
	fatalassert.Equal(t, []int{5000}, pointsLengthList)

	insertRecords(monitor, 2000)

	// 10000 points is lost

	fatalassert.Equal(t, 2, callWriteCount)
	fatalassert.Equal(t, []int{5000, 10000}, pointsLengthList)

	time.Sleep(time.Second)

	fatalassert.Equal(t, 3, callWriteCount)
	fatalassert.Equal(t, []int{5000, 10000, 1000}, pointsLengthList)
}
