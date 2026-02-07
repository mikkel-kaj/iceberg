package agentclient

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/mikkel-kaj/iceberg/internal/agent"
	"github.com/mikkel-kaj/iceberg/internal/compose"
)

type Client struct {
	BaseURL string
	Token   string
	HTTP    *http.Client
}

type StatusResponse = agent.StatusResponse

func NewClient(baseURL, token string) *Client {
	return &Client{BaseURL: strings.TrimRight(baseURL, "/"), Token: token, HTTP: &http.Client{}}
}

func (c *Client) httpClient() *http.Client {
	if c.HTTP == nil {
		return &http.Client{}
	}
	return c.HTTP
}

func (c *Client) do(req *http.Request) (*http.Response, error) {
	req.Header.Set("Authorization", "Bearer "+c.Token)
	return c.httpClient().Do(req)
}

func (c *Client) Deploy(ctx context.Context, name string, spec compose.DeploySpec, domain string) error {
	for i := range spec.Services {
		if domain != "" && spec.Services[i].Domain == "" {
			spec.Services[i].Domain = domain
		}
	}
	body, err := json.Marshal(spec)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.BaseURL+"/services/"+name+"/deploy", bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		raw, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
		return fmt.Errorf("deploy failed: status=%d body=%s", resp.StatusCode, string(raw))
	}
	return nil
}

func (c *Client) Destroy(ctx context.Context, name string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, c.BaseURL+"/services/"+name+"/destroy", nil)
	if err != nil {
		return err
	}
	resp, err := c.do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return fmt.Errorf("destroy failed: status=%d", resp.StatusCode)
	}
	return nil
}

func (c *Client) Status(ctx context.Context) (*StatusResponse, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.BaseURL+"/status", nil)
	if err != nil {
		return nil, err
	}
	resp, err := c.do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("status failed: %d", resp.StatusCode)
	}
	var out StatusResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) Logs(ctx context.Context, name string) (io.ReadCloser, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.BaseURL+"/services/"+name+"/logs?tail=100", nil)
	if err != nil {
		return nil, err
	}
	resp, err := c.do(req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode >= 300 {
		defer resp.Body.Close()
		return nil, fmt.Errorf("logs failed: %d", resp.StatusCode)
	}
	return resp.Body, nil
}
