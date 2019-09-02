package influxdb

import (
	"fmt"
	neturl "net/url"
	"sync"
	"time"

	"github.com/hashicorp/vault/api"
	influxdb "github.com/influxdata/influxdb1-client/v2"
	"github.com/pkg/errors"
	"github.com/theplant/appkit/credentials/vault"
	"github.com/theplant/appkit/log"
	"github.com/theplant/appkit/monitoring"
)

func NewInfluxDBMonitor(l log.Logger, vault *vault.Client, url *neturl.URL) (monitoring.Monitor, func(), error) {
	l = l.With(
		"component", "appkit/credentials/influxdb",
	)
	c := newClient(l, vault, url)

	return monitoring.NewInfluxdbMonitorWithClient(c.configUrl, l, c)
}

////////////////////////////////////////

type client struct {
	client influxdb.Client
	logger log.Logger
	vault  *vault.Client
	url    *neturl.URL

	sync.Mutex

	configUrl monitoring.InfluxMonitorConfig
}

func newClient(l log.Logger, vault *vault.Client, url *neturl.URL) *client {
	c := client{
		client:    errorClient{errors.New("no client")},
		configUrl: monitoring.InfluxMonitorConfig(url.String()),
		logger:    l,
		vault:     vault,
		url:       url,
	}

	vault.OnAuth(c.fetchAuth)

	return &c
}

func (c *client) resetClient(client influxdb.Client) {
	c.Lock()
	c.client = client
	c.Unlock()
}

func (c *client) fetchAuth() error {
	l := c.logger

	// Fetch credentials from Vault...
	role := "database/creds" + c.url.Path + "-influxdb"

	l.Debug().Log(
		"msg", "fetching influxdb credentials from vault",
		"role", role,
	)

	var secret *api.Secret
	secret, err := c.vault.Logical().Read(role)
	if err != nil {
		l.WithError(errors.Wrap(err, "error fetching credentials")).Log()
		return nil
	} else if secret == nil {
		l.WithError(errors.New("vault client returned nil secret")).Log()
		return nil
	}

	username := secret.Data["username"].(string)
	password := secret.Data["password"].(string)

	// ... then create new InfluxDB client with fetched credentials
	httpConfig := influxdb.HTTPConfig{
		Addr:     fmt.Sprintf("https://%s", c.url.Host),
		Username: username,
		Password: password,
	}

	client, err := influxdb.NewHTTPClient(httpConfig)
	if err != nil {
		l.WithError(errors.Wrap(err, "error creating influxdb http client")).Log()
	} else {
		l.Info().Log(
			"msg", fmt.Sprintf("updating influxdb client credentials"),
			"username", username,
		)
		c.resetClient(client)
	}

	return nil
}

////////////////////
// influxdb.Client interface methods
func (c *client) Ping(timeout time.Duration) (time.Duration, string, error) {
	c.Lock()
	client := c.client
	c.Unlock()

	return client.Ping(timeout)
}

func (c *client) Write(bp influxdb.BatchPoints) error {
	c.Lock()
	client := c.client
	c.Unlock()

	return client.Write(bp)
}

func (c *client) Query(q influxdb.Query) (*influxdb.Response, error) {
	c.Lock()
	client := c.client
	c.Unlock()

	return client.Query(q)
}
func (c *client) QueryAsChunk(q influxdb.Query) (*influxdb.ChunkedResponse, error) {
	c.Lock()
	client := c.client
	c.Unlock()

	return client.QueryAsChunk(q)
}

func (c *client) Close() error {
	c.Lock()
	client := c.client
	c.Unlock()

	return client.Close()
}

////////////////////////////////////////

type errorClient struct {
	error
}

func (e errorClient) Ping(timeout time.Duration) (time.Duration, string, error) {
	return 0, "", e
}

func (e errorClient) Write(bp influxdb.BatchPoints) error {
	return e
}

func (e errorClient) Query(q influxdb.Query) (*influxdb.Response, error) {
	return nil, e
}
func (e errorClient) QueryAsChunk(q influxdb.Query) (*influxdb.ChunkedResponse, error) {
	return nil, e
}

func (e errorClient) Close() error {
	return e
}
