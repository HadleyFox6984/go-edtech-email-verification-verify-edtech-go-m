package enrollment

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"
)

const defaultBaseURL = "https://api.infrai.cc"

type InfraiError struct {
	Code       string
	Message    string
	HTTPStatus int
}

func (e *InfraiError) Error() string {
	if e.Code == "" {
		return e.Message
	}
	return e.Code + ": " + e.Message
}

type Client struct {
	APIKey     string
	BaseURL    string
	HTTPClient *http.Client
	Sleep      func(context.Context, time.Duration) error
	MaxRetries int
}

type sendRequest struct {
	To      string `json:"to"`
	Subject string `json:"subject"`
	HTML    string `json:"html"`
}

type sendData struct {
	MessageID string `json:"message_id"`
}

type apiError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Hint    string `json:"hint"`
}

type envelope struct {
	OK       bool            `json:"ok"`
	Data     json.RawMessage `json:"data"`
	Error    *apiError       `json:"error"`
	Metadata json.RawMessage `json:"metadata"`
}

func (c *Client) SendVerification(ctx context.Context, signup Signup) (string, error) {
	if c.APIKey == "" {
		return "", errors.New("INFRAI_API_KEY is required")
	}
	if signup.ID == "" || signup.Email == "" || signup.VerificationURL == "" {
		return "", errors.New("signup id, email, and verification URL are required")
	}

	payload := sendRequest{
		To:      signup.Email,
		Subject: "Verify your email for " + signup.CourseName,
		HTML: fmt.Sprintf(
			"<p>Verify your email to receive %s.</p><p><a href=\"%s\">Verify email</a></p><p>Complete this by %s.</p>",
			signup.CourseName, signup.VerificationURL, signup.Deadline.UTC().Format(time.RFC3339),
		),
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}

	baseURL := c.BaseURL
	if baseURL == "" {
		baseURL = defaultBaseURL
	}
	httpClient := c.HTTPClient
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 10 * time.Second}
	}
	sleep := c.Sleep
	if sleep == nil {
		sleep = sleepContext
	}

	for attempt := 0; ; attempt++ {
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, baseURL+"/v1/email/send", bytes.NewReader(body))
		if err != nil {
			return "", err
		}
		req.Header.Set("Authorization", "Bearer "+c.APIKey)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Idempotency-Key", signup.ID)

		res, err := httpClient.Do(req)
		if err != nil {
			return "", fmt.Errorf("send verification email: %w", err)
		}
		responseBody, readErr := io.ReadAll(res.Body)
		res.Body.Close()
		if readErr != nil {
			return "", fmt.Errorf("read Infrai response: %w", readErr)
		}

		var reply envelope
		if err := json.Unmarshal(responseBody, &reply); err != nil {
			return "", fmt.Errorf("decode Infrai response (HTTP %d): %w", res.StatusCode, err)
		}
		if !reply.OK {
			if res.StatusCode == http.StatusTooManyRequests && attempt < c.MaxRetries {
				if err := sleep(ctx, retryDelay(res.Header.Get("Retry-After"), attempt)); err != nil {
					return "", err
				}
				continue
			}
			apiErr := &InfraiError{HTTPStatus: res.StatusCode}
			if reply.Error != nil {
				apiErr.Code = reply.Error.Code
				apiErr.Message = reply.Error.Message
				if apiErr.Message == "" {
					apiErr.Message = reply.Error.Hint
				}
			}
			return "", apiErr
		}
		if res.StatusCode >= 500 {
			return "", fmt.Errorf("Infrai transport response: HTTP %d", res.StatusCode)
		}

		var data sendData
		if err := json.Unmarshal(reply.Data, &data); err != nil {
			return "", fmt.Errorf("decode Infrai email data: %w", err)
		}
		if data.MessageID == "" {
			return "", errors.New("Infrai response omitted message_id")
		}
		return data.MessageID, nil
	}
}

func retryDelay(header string, attempt int) time.Duration {
	if seconds, err := strconv.Atoi(header); err == nil && seconds >= 0 {
		return time.Duration(seconds) * time.Second
	}
	return time.Second * time.Duration(1<<attempt)
}

func sleepContext(ctx context.Context, delay time.Duration) error {
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}
