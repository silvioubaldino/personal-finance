//go:build acceptance

package harness

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"personal-finance/internal/plataform/authentication"
)

// HeaderNow carries the scenario's "today" (RFC 3339). Only the harness wrapper reads it.
const HeaderNow = "X-Test-Now"

// Identity says who calls and what day it is for the caller.
type Identity struct {
	UserID string
	Plan   string // "plus" (default) | "free"
	Now    time.Time
}

// Response is the raw answer of one call.
type Response struct {
	Status int
	Body   []byte
}

// OK reports a 2xx status.
func (r Response) OK() bool { return r.Status >= 200 && r.Status < 300 }

func (r Response) String() string { return fmt.Sprintf("HTTP %d: %s", r.Status, r.Body) }

// Decode unmarshals the body into out.
func (r Response) Decode(out any) error {
	if err := json.Unmarshal(r.Body, out); err != nil {
		return fmt.Errorf("decoding %s: %w", r, err)
	}
	return nil
}

// Client calls the API over HTTP. The suite never imports the API's output structs: the
// JSON contract is what is under test.
type Client struct {
	baseURL string
	http    *http.Client
}

func NewClient(baseURL string) *Client {
	return &Client{
		baseURL: baseURL,
		http:    &http.Client{Timeout: 30 * time.Second},
	}
}

// Do performs one request as the given identity. body may be nil.
func (c *Client) Do(id Identity, method, path string, body any) (Response, error) {
	var reader io.Reader
	if body != nil {
		raw, err := json.Marshal(body)
		if err != nil {
			return Response{}, fmt.Errorf("marshalling request body: %w", err)
		}
		reader = bytes.NewReader(raw)
	}

	req, err := http.NewRequest(method, c.baseURL+path, reader)
	if err != nil {
		return Response{}, err
	}
	req.Header.Set(authentication.UserToken, buildToken(id.UserID, id.Plan))
	req.Header.Set(HeaderNow, id.Now.UTC().Format(time.RFC3339))
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return Response{}, fmt.Errorf("%s %s: %w", method, path, err)
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return Response{}, fmt.Errorf("reading %s %s: %w", method, path, err)
	}
	return Response{Status: resp.StatusCode, Body: raw}, nil
}
