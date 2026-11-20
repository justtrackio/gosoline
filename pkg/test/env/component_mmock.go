package env

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"
)

// MmockRequest is an HTTP request captured by Mmock, without any client-side matching.
type MmockRequest struct {
	Method                string              `json:"method"`
	Path                  string              `json:"path"`
	Headers               map[string][]string `json:"headers"`
	QueryStringParameters map[string][]string `json:"queryStringParameters"`
	Body                  string              `json:"body"`
}

// MmockComponent provides mock traffic, the administration UI and captured requests.
type MmockComponent struct {
	baseComponent
	binding      ContainerBinding
	adminBinding ContainerBinding
}

// Address returns the base URL for application requests to the mock server.
func (c *MmockComponent) Address() string {
	return "http://" + c.binding.getAddress()
}

// AdminAddress returns the base URL for Mmock's UI and native administration API.
func (c *MmockComponent) AdminAddress() string {
	return "http://" + c.adminBinding.getAddress()
}

// Requests returns captured requests in journal order for ordinary test assertions.
func (c *MmockComponent) Requests(ctx context.Context) ([]MmockRequest, error) {
	data, err := c.request(ctx, http.MethodGet, "/api/request/all", nil)
	if err != nil {
		return nil, fmt.Errorf("could not retrieve mmock requests: %w", err)
	}
	var transactions []struct {
		Request MmockRequest `json:"request"`
	}
	if err = json.Unmarshal(data, &transactions); err != nil {
		return nil, fmt.Errorf("could not decode mmock requests: %w", err)
	}
	requests := make([]MmockRequest, len(transactions))
	for i, transaction := range transactions {
		requests[i] = transaction.Request
	}

	return requests, nil
}

// ResetRequests clears captured requests while preserving the loaded mock definitions.
func (c *MmockComponent) ResetRequests(ctx context.Context) error {
	if _, err := c.request(ctx, http.MethodGet, "/api/request/reset", nil); err != nil {
		return fmt.Errorf("could not reset mmock requests: %w", err)
	}

	return nil
}

func (c *MmockComponent) request(ctx context.Context, method, path string, body io.Reader) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, method, c.AdminAddress()+path, body)
	if err != nil {
		return nil, fmt.Errorf("could not create mmock request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	client := http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("could not call mmock API: %w", err)
	}
	data, err := io.ReadAll(resp.Body)
	if err = errors.Join(err, resp.Body.Close()); err != nil {
		return nil, fmt.Errorf("could not read mmock response: %w", err)
	}
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return nil, fmt.Errorf("mmock API %s %s returned %s: %s", method, path, resp.Status, data)
	}

	return data, nil
}
