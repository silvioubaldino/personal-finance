package push

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

const (
	ExpoAPIURL     = "https://exp.host/--/api/v2/push/send"
	DefaultTimeout = 30 * time.Second
)

type PushMessage struct {
	To    string `json:"to"`
	Title string `json:"title"`
	Body  string `json:"body"`
	Data  any    `json:"data,omitempty"`
}

type ExpoPushTicket struct {
	Status  string `json:"status"`
	ID      string `json:"id,omitempty"`
	Message string `json:"message,omitempty"`
	Details struct {
		Error string `json:"error,omitempty"`
	} `json:"details,omitempty"`
}

type ExpoResponse struct {
	Data []ExpoPushTicket `json:"data"`
}

type SendResult struct {
	SuccessCount  int
	FailureCount  int
	InvalidTokens []string
}

type ExpoClient struct {
	httpClient *http.Client
	apiURL     string
}

func NewExpoClient() *ExpoClient {
	return &ExpoClient{
		httpClient: &http.Client{
			Timeout: DefaultTimeout,
		},
		apiURL: ExpoAPIURL,
	}
}

func (c *ExpoClient) Send(ctx context.Context, tokens []string, title, body string) (SendResult, error) {
	if len(tokens) == 0 {
		return SendResult{}, nil
	}

	messages := make([]PushMessage, len(tokens))
	for i, token := range tokens {
		messages[i] = PushMessage{
			To:    token,
			Title: title,
			Body:  body,
		}
	}

	jsonData, err := json.Marshal(messages)
	if err != nil {
		return SendResult{}, fmt.Errorf("error marshaling push messages: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.apiURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return SendResult{}, fmt.Errorf("error creating request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return SendResult{}, fmt.Errorf("error sending push notification: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return SendResult{}, fmt.Errorf("expo api returned status %d", resp.StatusCode)
	}

	var expoResp ExpoResponse
	if err := json.NewDecoder(resp.Body).Decode(&expoResp); err != nil {
		return SendResult{}, fmt.Errorf("error decoding expo response: %w", err)
	}

	result := SendResult{}
	for i, ticket := range expoResp.Data {
		if ticket.Status == "ok" {
			result.SuccessCount++
		} else {
			result.FailureCount++
			if ticket.Details.Error == "DeviceNotRegistered" {
				result.InvalidTokens = append(result.InvalidTokens, tokens[i])
			}
		}
	}

	return result, nil
}
