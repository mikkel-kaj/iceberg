package cloudflare

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

const defaultBaseURL = "https://api.cloudflare.com/client/v4"

type Client struct {
	Token      string
	HTTPClient *http.Client
	BaseURL    string
}

func NewClient(token string) *Client {
	return &Client{Token: token, HTTPClient: &http.Client{}, BaseURL: defaultBaseURL}
}

type DNSRecord struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Type    string `json:"type"`
	Content string `json:"content"`
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
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.httpClient().Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		raw, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
		return fmt.Errorf("cloudflare %s %s failed: status=%d body=%s", method, path, resp.StatusCode, string(raw))
	}
	if out == nil {
		return nil
	}
	return json.NewDecoder(resp.Body).Decode(out)
}

func extractZone(domain string) string {
	parts := strings.Split(domain, ".")
	if len(parts) <= 2 {
		return domain
	}
	return strings.Join(parts[len(parts)-2:], ".")
}

func (c *Client) ValidateToken(ctx context.Context) error {
	var resp struct {
		Success bool `json:"success"`
	}
	if err := c.do(ctx, http.MethodGet, "/zones", nil, &resp); err != nil {
		return err
	}
	if !resp.Success {
		return fmt.Errorf("cloudflare token invalid")
	}
	return nil
}

func (c *Client) GetZoneID(ctx context.Context, domain string) (string, error) {
	zone := extractZone(domain)
	var resp struct {
		Result []struct {
			ID string `json:"id"`
		} `json:"result"`
	}
	path := "/zones?name=" + url.QueryEscape(zone)
	if err := c.do(ctx, http.MethodGet, path, nil, &resp); err != nil {
		return "", err
	}
	if len(resp.Result) == 0 {
		return "", fmt.Errorf("zone not found for %s", zone)
	}
	return resp.Result[0].ID, nil
}

func (c *Client) CreateARecord(ctx context.Context, zoneID, subdomain, ip string) (*DNSRecord, error) {
	var resp struct {
		Result DNSRecord `json:"result"`
	}
	payload := map[string]any{"type": "A", "name": subdomain, "content": ip, "proxied": false}
	if err := c.do(ctx, http.MethodPost, "/zones/"+zoneID+"/dns_records", payload, &resp); err != nil {
		return nil, err
	}
	return &resp.Result, nil
}

func (c *Client) DeleteRecord(ctx context.Context, zoneID, recordID string) error {
	return c.do(ctx, http.MethodDelete, "/zones/"+zoneID+"/dns_records/"+recordID, nil, nil)
}

func (c *Client) FindRecord(ctx context.Context, zoneID, subdomain string) (*DNSRecord, error) {
	path := "/zones/" + zoneID + "/dns_records?type=A&name=" + url.QueryEscape(subdomain)
	var resp struct {
		Result []DNSRecord `json:"result"`
	}
	if err := c.do(ctx, http.MethodGet, path, nil, &resp); err != nil {
		return nil, err
	}
	if len(resp.Result) == 0 {
		return nil, nil
	}
	return &resp.Result[0], nil
}
