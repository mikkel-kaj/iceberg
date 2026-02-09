package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

type ControlAPI interface {
	CreateServer(ctx context.Context, cfgPath string) (*ServerCreateResult, error)
	DestroyServer(ctx context.Context, cfgPath, name string) error
	Deploy(ctx context.Context, cfgPath string, in DeployInput) (*DeployResult, error)
}

type controlClient struct {
	BaseURL string
	HTTP    *http.Client
}

func newDefaultControlClient(baseURL string) ControlAPI {
	return &controlClient{
		BaseURL: strings.TrimRight(baseURL, "/"),
		HTTP:    &http.Client{},
	}
}

func (c *controlClient) client() *http.Client {
	if c.HTTP == nil {
		return &http.Client{}
	}
	return c.HTTP
}

func (c *controlClient) CreateServer(ctx context.Context, cfgPath string) (*ServerCreateResult, error) {
	reqBody := map[string]string{"config_path": cfgPath}
	body, _ := json.Marshal(reqBody)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.BaseURL+"/v1/servers", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.client().Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return nil, readControlError(resp)
	}
	var out ServerCreateResult
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *controlClient) DestroyServer(ctx context.Context, cfgPath, name string) error {
	v := url.Values{}
	v.Set("config_path", cfgPath)
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, c.BaseURL+"/v1/servers/"+url.PathEscape(name)+"?"+v.Encode(), nil)
	if err != nil {
		return err
	}
	resp, err := c.client().Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return readControlError(resp)
	}
	return nil
}

func (c *controlClient) Deploy(ctx context.Context, cfgPath string, in DeployInput) (*DeployResult, error) {
	payload := struct {
		ConfigPath string      `json:"config_path"`
		Input      DeployInput `json:"input"`
	}{
		ConfigPath: cfgPath,
		Input:      in,
	}
	body, _ := json.Marshal(payload)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.BaseURL+"/v1/deploy", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.client().Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return nil, readControlError(resp)
	}
	var out DeployResult
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, err
	}
	return &out, nil
}

func readControlError(resp *http.Response) error {
	raw, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
	var payload struct {
		Error string `json:"error"`
	}
	if json.Unmarshal(raw, &payload) == nil && payload.Error != "" {
		return errors.New(payload.Error)
	}
	if len(raw) == 0 {
		return fmt.Errorf("control-agent request failed: status=%d", resp.StatusCode)
	}
	return fmt.Errorf("control-agent request failed: status=%d body=%s", resp.StatusCode, strings.TrimSpace(string(raw)))
}
