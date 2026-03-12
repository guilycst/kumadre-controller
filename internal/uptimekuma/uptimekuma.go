package uptimekuma

import (
	"context"
	"fmt"
	"strings"

	kuma "github.com/breml/go-uptime-kuma-client"
	"github.com/breml/go-uptime-kuma-client/monitor"
	"github.com/breml/go-uptime-kuma-client/tag"
)

type Client struct {
	client  *kuma.Client
	baseURL string
}

type Monitor struct {
	ID   int64
	Name string
	URL  string
	Type string
}

func NewClient(ctx context.Context, u, username, password string, insecure bool) (*Client, error) {
	baseURL := strings.TrimRight(u, "/")

	client, err := kuma.New(ctx, baseURL, username, password)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to Uptime Kuma: %w", err)
	}

	return &Client{
		client:  client,
		baseURL: baseURL,
	}, nil
}

func (c *Client) Disconnect() error {
	return c.client.Disconnect()
}

func (c *Client) CreateHTTPMonitor(ctx context.Context, name, targetURL string, interval int) (*Monitor, error) {
	httpMonitor := &monitor.HTTP{
		Base: monitor.Base{
			Name:            name,
			Interval:        int64(interval),
			RetryInterval:   60,
			MaxRetries:      0,
			UpsideDown:      false,
			NotificationIDs: []int64{},
		},
		HTTPDetails: monitor.HTTPDetails{
			URL:                 targetURL,
			Method:              "GET",
			AcceptedStatusCodes: []string{"200-299"},
			Timeout:             30,
		},
	}

	monitorID, err := c.client.CreateMonitor(ctx, httpMonitor)
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP monitor: %w", err)
	}

	return &Monitor{
		ID:   monitorID,
		Name: name,
		URL:  targetURL,
		Type: "http",
	}, nil
}

func (c *Client) CreatePingMonitor(ctx context.Context, name, hostname string, interval int) (*Monitor, error) {
	pingMonitor := &monitor.Ping{
		Base: monitor.Base{
			Name:            name,
			Interval:        int64(interval),
			RetryInterval:   60,
			MaxRetries:      0,
			UpsideDown:      false,
			NotificationIDs: []int64{},
		},
		PingDetails: monitor.PingDetails{
			Hostname: hostname,
		},
	}

	monitorID, err := c.client.CreateMonitor(ctx, pingMonitor)
	if err != nil {
		return nil, fmt.Errorf("failed to create Ping monitor: %w", err)
	}

	return &Monitor{
		ID:   monitorID,
		Name: name,
		URL:  hostname,
		Type: "ping",
	}, nil
}

func (c *Client) DeleteMonitor(ctx context.Context, monitorID int64) error {
	return c.client.DeleteMonitor(ctx, monitorID)
}

func (c *Client) GetMonitors(ctx context.Context) ([]Monitor, error) {
	monitors, err := c.client.GetMonitors(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get monitors: %w", err)
	}

	result := make([]Monitor, 0, len(monitors))
	for _, m := range monitors {
		result = append(result, Monitor{
			ID:   m.ID,
			Name: m.Name,
			Type: m.Type(),
		})
	}

	return result, nil
}

func (c *Client) AddTag(ctx context.Context, monitorID int64, tagName, value string) error {
	newTag := tag.Tag{
		Name: tagName,
	}

	tagID, err := c.client.CreateTag(ctx, newTag)
	if err != nil {
		return fmt.Errorf("failed to create tag: %w", err)
	}

	_, err = c.client.AddMonitorTag(ctx, tagID, monitorID, value)
	return err
}
