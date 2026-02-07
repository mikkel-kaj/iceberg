package hetzner

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

const defaultBaseURL = "https://api.hetzner.cloud/v1"

type Client struct {
	Token      string
	HTTPClient *http.Client
	BaseURL    string
}

func NewClient(token string) *Client {
	return &Client{
		Token:      token,
		HTTPClient: &http.Client{},
		BaseURL:    defaultBaseURL,
	}
}

func (c *Client) baseURL() string {
	if c.BaseURL == "" {
		return defaultBaseURL
	}
	return c.BaseURL
}

func (c *Client) httpClient() *http.Client {
	if c.HTTPClient == nil {
		return &http.Client{}
	}
	return c.HTTPClient
}

func (c *Client) do(ctx context.Context, method, path string, reqBody any, out any) error {
	var body io.Reader
	if reqBody != nil {
		enc, err := json.Marshal(reqBody)
		if err != nil {
			return err
		}
		body = bytes.NewReader(enc)
	}
	req, err := http.NewRequestWithContext(ctx, method, strings.TrimRight(c.baseURL(), "/")+path, body)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+c.Token)
	if reqBody != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := c.httpClient().Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		raw, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
		return fmt.Errorf("hetzner %s %s failed: status=%d body=%s", method, path, resp.StatusCode, string(raw))
	}
	if out == nil {
		return nil
	}
	return json.NewDecoder(resp.Body).Decode(out)
}

func (c *Client) ValidateToken(ctx context.Context) error {
	return c.do(ctx, http.MethodGet, "/ssh_keys", nil, &struct{}{})
}
