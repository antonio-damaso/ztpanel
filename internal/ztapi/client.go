// Package ztapi is a thin client for the ZeroTier controller's local
// JSON API (the same API the zerotier-one daemon exposes on
// 127.0.0.1:9993, authenticated with authtoken.secret).
//
// It intentionally stays close to the wire format: most calls accept
// and return arbitrary JSON objects (map[string]any) rather than a
// fully-typed model, because the controller's network/member schema
// has many optional, version-dependent fields and ztpanel's job is to
// expose them, not to guess which ones matter.
package ztapi

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

type Client struct {
	baseURL    string
	authToken  string
	httpClient *http.Client
}

func New(baseURL, authTokenPath string) (*Client, error) {
	token, err := readAuthToken(authTokenPath)
	if err != nil {
		return nil, fmt.Errorf("reading zerotier authtoken: %w", err)
	}
	return &Client{
		baseURL:   strings.TrimRight(baseURL, "/"),
		authToken: token,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}, nil
}

func readAuthToken(path string) (string, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(b)), nil
}

func (c *Client) do(method, path string, body any, out any) error {
	var reqBody io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return err
		}
		reqBody = bytes.NewReader(b)
	}

	req, err := http.NewRequest(method, c.baseURL+path, reqBody)
	if err != nil {
		return err
	}
	req.Header.Set("X-ZT1-Auth", c.authToken)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	if resp.StatusCode >= 400 {
		return &APIError{Status: resp.StatusCode, Body: string(respBody)}
	}

	if out != nil && len(respBody) > 0 {
		if err := json.Unmarshal(respBody, out); err != nil {
			return fmt.Errorf("decoding controller response: %w", err)
		}
	}
	return nil
}

type APIError struct {
	Status int
	Body   string
}

func (e *APIError) Error() string {
	return fmt.Sprintf("zerotier controller returned %d: %s", e.Status, e.Body)
}

// Status returns the controller's own node status (address, version, ...).
func (c *Client) Status() (map[string]any, error) {
	var out map[string]any
	err := c.do(http.MethodGet, "/status", nil, &out)
	return out, err
}

// ListNetworkIDs returns the IDs of every network this controller manages.
func (c *Client) ListNetworkIDs() ([]string, error) {
	var out []string
	err := c.do(http.MethodGet, "/controller/network", nil, &out)
	return out, err
}

func (c *Client) GetNetwork(id string) (map[string]any, error) {
	var out map[string]any
	err := c.do(http.MethodGet, "/controller/network/"+id, nil, &out)
	return out, err
}

// UpsertNetwork creates a network (id == "______") or updates one,
// merging the given fields. The controller computes the rest.
func (c *Client) UpsertNetwork(id string, fields map[string]any) (map[string]any, error) {
	var out map[string]any
	err := c.do(http.MethodPost, "/controller/network/"+id, fields, &out)
	return out, err
}

func (c *Client) DeleteNetwork(id string) error {
	return c.do(http.MethodDelete, "/controller/network/"+id, nil, nil)
}

func (c *Client) ListMemberIDs(networkID string) (map[string]any, error) {
	var out map[string]any
	err := c.do(http.MethodGet, "/controller/network/"+networkID+"/member", nil, &out)
	return out, err
}

func (c *Client) GetMember(networkID, memberID string) (map[string]any, error) {
	var out map[string]any
	err := c.do(http.MethodGet, "/controller/network/"+networkID+"/member/"+memberID, nil, &out)
	return out, err
}

func (c *Client) UpsertMember(networkID, memberID string, fields map[string]any) (map[string]any, error) {
	var out map[string]any
	err := c.do(http.MethodPost, "/controller/network/"+networkID+"/member/"+memberID, fields, &out)
	return out, err
}

func (c *Client) DeleteMember(networkID, memberID string) error {
	return c.do(http.MethodDelete, "/controller/network/"+networkID+"/member/"+memberID, nil, nil)
}

// Peers returns live peer info (online status, latency, paths) keyed
// by ZeroTier address, as reported by `GET /peer`.
func (c *Client) Peers() ([]map[string]any, error) {
	var out []map[string]any
	err := c.do(http.MethodGet, "/peer", nil, &out)
	return out, err
}
