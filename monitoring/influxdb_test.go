package monitoring

import (
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	influxdb "github.com/influxdata/influxdb1-client/v2"
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
		_, _, err := NewInfluxdbMonitor(InfluxMonitorConfig(config), logger)

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
		_, _, err := NewInfluxdbMonitor(InfluxMonitorConfig(config), logger)

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
			name:   "default batch-write-interval, buffer-size, max-buffer-size",
			config: "https://root:password@localhost:8086/local?service-name=api",
			expectedCfg: &influxMonitorCfg{
				Scheme:             "https",
				Host:               "localhost:8086",
				Addr:               "https://localhost:8086",
				Username:           "root",
				Password:           "password",
				Database:           "local",
				BatchWriteInterval: defaultBatchWriteInterval,
				BufferSize:         defaultBufferSize,
				MaxBufferSize:      defaultMaxBufferSize,
				ServiceName:        "api",
			},
		},

		{
			name:   "custom batch-write-interval, buffer-size, max-buffer-size",
			config: "http://localhost:8086/local?batch-write-interval=30s&buffer-size=1000&max-buffer-size=5000",
			expectedCfg: &influxMonitorCfg{
				Scheme:             "http",
				Host:               "localhost:8086",
				Addr:               "http://localhost:8086",
				Username:           "",
				Password:           "",
				Database:           "local",
				BatchWriteInterval: time.Second * 30,
				BufferSize:         1000,
				MaxBufferSize:      5000,
				ServiceName:        "",
			},
		},

		{
			name:                "batch-write-interval format error, missing unit in duration",
			config:              "http://localhost:8086/local?batch-write-interval=30",
			expectedErrContains: "influxdb config parameter batch-write-interval format error",
		},

		{
			name:                "buffer-size format error",
			config:              "http://localhost:8086/local?buffer-size=abc",
			expectedErrContains: "influxdb config parameter buffer-size format error",
		},

		{
			name:                "max-buffer-size format error",
			config:              "http://localhost:8086/local?max-buffer-size=-1",
			expectedErrContains: "influxdb config parameter max-buffer-size format error",
		},

		{
			name:                "buffer-size > max-buffer-size error",
			config:              "http://localhost:8086/local?buffer-size=1001&max-buffer-size=1000",
			expectedErrContains: "buffer-size can not be greater than max-buffer-size",
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

func newMonitor(client influxdb.Client, bufferSize int, maxBufferSize int, serviceName string) (monitor *influxdbMonitor, closeFunc func()) {
	monitor = &influxdbMonitor{
		database: "test_database",
		client:   client,
		logger:   log.NewNopLogger(),

		pointChan:          make(chan *influxdb.Point),
		batchWriteInterval: time.Second * 1,
		bufferSize:         bufferSize,
		maxBufferSize:      maxBufferSize,

		done: &sync.WaitGroup{},

		serviceName: serviceName,
	}

	running := make(chan struct{})
	go monitor.batchWriteDaemon(running)

	return monitor, func() {
		close(running)
		monitor.done.Wait()
	}
}

func insertRecords(monitor Monitor, callTimes int) {
	for i := 0; i < callTimes; i++ {
		monitor.InsertRecord("measurement", "value", nil, nil, time.Now())
	}
	time.Sleep(time.Millisecond * 100)
}

func assertWriteCalls(t *testing.T, clientMock *ClientMock, expectedCallCount int, expectedCallPointsLengths []int) {
	t.Helper()

	fatalassert.Equal(t, expectedCallCount, len(clientMock.WriteCalls()))

	bps := clientMock.WriteCalls()
	var pointLengths []int
	for _, bp := range bps {
		pointLengths = append(pointLengths, len(bp.Bp.Points()))
	}
	fatalassert.Equal(t, expectedCallPointsLengths, pointLengths)
}

func TestInfluxdbBatchWrite(t *testing.T) {
	mockedClient := &ClientMock{
		WriteFunc: func(bp influxdb.BatchPoints) error {
			return nil
		},
	}

	monitor, _ := newMonitor(mockedClient, 5000, 10000, "")

	insertRecords(monitor, 4000)

	// not reach BufferSize
	assertWriteCalls(t, mockedClient, 0, nil)

	insertRecords(monitor, 1000)

	// reach BufferSize
	assertWriteCalls(t, mockedClient, 1, []int{5001})

	insertRecords(monitor, 11000)

	// reach BufferSize twice and remain 1000
	assertWriteCalls(t, mockedClient, 3, []int{5001, 5001, 5001})

	insertRecords(monitor, 1000)

	// not reach BufferSize, len(points) = 2000
	assertWriteCalls(t, mockedClient, 3, []int{5001, 5001, 5001})

	time.Sleep(time.Second * 1)

	// ticker is triggered
	assertWriteCalls(t, mockedClient, 4, []int{5001, 5001, 5001, 2001})
}

func TestInfluxdbBatchWrite__WriteFailed(t *testing.T) {
	writeError := errors.New("write error")
	mockedClient := &ClientMock{
		WriteFunc: func(bp influxdb.BatchPoints) error {
			return writeError
		},
	}

	monitor, _ := newMonitor(mockedClient, 5000, 16000, "")

	insertRecords(monitor, 5000)

	// nextWriteBufferSize = 10000
	// len(points) = 5000

	assertWriteCalls(t, mockedClient, 1, []int{5001})

	insertRecords(monitor, 10000)

	// nextWriteBufferSize = 16000
	// len(points) = 15000

	assertWriteCalls(t, mockedClient, 3, []int{5001, 10001, 15001})

	insertRecords(monitor, 100)

	// nextWriteBufferSize = 16000
	// len(points) = 15100

	assertWriteCalls(t, mockedClient, 3, []int{5001, 10001, 15001})

	insertRecords(monitor, 2000)

	// nextWriteBufferSize = 16000
	// len(points) = 16000
	// 16000 points is lost

	assertWriteCalls(t, mockedClient, 4, []int{5001, 10001, 15001, 16001})

	time.Sleep(time.Second * 1)

	// ticker is triggered
	//
	// nextWriteBufferSize = 16000
	// len(points) = 1100

	assertWriteCalls(t, mockedClient, 5, []int{5001, 10001, 15001, 16001, 1101})

	insertRecords(monitor, 10000)

	// not reach nextWriteBufferSize (16000)
	// len(points) = 11100
	// not trigger batch write

	assertWriteCalls(t, mockedClient, 5, []int{5001, 10001, 15001, 16001, 1101})

	// the influxdb is recover to normal

	*mockedClient = ClientMock{
		WriteFunc: func(bp influxdb.BatchPoints) error {
			return nil
		},
	}

	insertRecords(monitor, 4900)

	// nextWriteBufferSize = 5000
	// len(points) = 16000

	assertWriteCalls(t, mockedClient, 1, []int{16001})

	insertRecords(monitor, 5000)

	assertWriteCalls(t, mockedClient, 2, []int{16001, 5001})
}

func TestInfluxdbBatchWrite__WriteFailed__BufferSizeAndMaxBufferSizeIsDefault(t *testing.T) {
	writeError := errors.New("write error")
	mockedClient := &ClientMock{
		WriteFunc: func(bp influxdb.BatchPoints) error {
			return writeError
		},
	}

	monitor, _ := newMonitor(mockedClient, 5000, 10000, "")

	insertRecords(monitor, 9000)

	assertWriteCalls(t, mockedClient, 1, []int{5001})

	insertRecords(monitor, 2000)

	// 10000 points is lost

	assertWriteCalls(t, mockedClient, 2, []int{5001, 10001})

	time.Sleep(time.Second)

	assertWriteCalls(t, mockedClient, 3, []int{5001, 10001, 1001})
}

func TestServiceName(t *testing.T) {
	var bp influxdb.BatchPoints

	mockedClient := &ClientMock{
		WriteFunc: func(p influxdb.BatchPoints) error {
			bp = p
			return nil
		},
	}

	// tag is nil

	monitor, cf := newMonitor(mockedClient, 1, 1, "api")

	monitor.InsertRecord("request", 100, nil, nil, time.Time{})
	cf()

	fatalassert.Equal(t, 1, len(mockedClient.WriteCalls()))
	fatalassert.Equal(t, bp.Points()[0].Tags(), map[string]string{
		"service": "api",
	})
	fatalassert.Equal(t, bp.Points()[1].Name(), "influxdb-queue-length")
	fatalassert.Equal(t, bp.Points()[1].Tags(), map[string]string{
		"service": "api",
	})

	// tag is not nil

	monitor, cf = newMonitor(mockedClient, 1, 1, "api")

	monitor.InsertRecord("request", 100, map[string]string{"tag1": "value1"}, nil, time.Time{})
	cf()

	fatalassert.Equal(t, 2, len(mockedClient.WriteCalls()))
	fatalassert.Equal(t, bp.Points()[0].Tags(), map[string]string{
		"tag1":    "value1",
		"service": "api",
	})
	fatalassert.Equal(t, bp.Points()[1].Tags(), map[string]string{
		"service": "api",
	})

	// service name is empty

	monitor, cf = newMonitor(mockedClient, 1, 1, "")

	monitor.InsertRecord("request", 100, map[string]string{"tag1": "value1"}, nil, time.Time{})
	cf()

	fatalassert.Equal(t, 3, len(mockedClient.WriteCalls()))
	fatalassert.Equal(t, bp.Points()[0].Tags(), map[string]string{
		"tag1": "value1",
	})
	fatalassert.Equal(t, bp.Points()[1].Tags(), map[string]string{})
}
