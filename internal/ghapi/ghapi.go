// Package ghapi is a minimal GitHub GraphQL client on net/http.
package ghapi

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

const (
	maxRetries    = 2
	maxRetryAfter = 60 * time.Second
	maxBodyBytes  = 512
)

type Client struct {
	HTTP     *http.Client
	Endpoint string
	Token    string
}

type GraphQLError struct {
	Messages []string
}

func (e *GraphQLError) Error() string {
	return "ghapi: graphql: " + strings.Join(e.Messages, "; ")
}

type RateLimitError struct {
	ResetAt    time.Time
	RetryAfter time.Duration
}

func (e *RateLimitError) Error() string {
	return fmt.Sprintf("ghapi: rate limited, retry after %s (reset at %s)", e.RetryAfter, e.ResetAt.UTC().Format(time.RFC3339))
}

func (c *Client) Do(ctx context.Context, query string, vars map[string]any, out any) error {
	body, err := json.Marshal(struct {
		Query     string         `json:"query"`
		Variables map[string]any `json:"variables,omitempty"`
	}{query, vars})
	if err != nil {
		return fmt.Errorf("ghapi: encode request: %w", err)
	}
	for attempt := 0; ; attempt++ {
		err := c.once(ctx, body, out)
		var rl *RateLimitError
		if !errors.As(err, &rl) || attempt >= maxRetries || rl.RetryAfter > maxRetryAfter {
			return err
		}
		if err := sleep(ctx, rl.RetryAfter); err != nil {
			return err
		}
	}
}

func sleep(ctx context.Context, d time.Duration) error {
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return fmt.Errorf("ghapi: waiting for rate limit: %w", ctx.Err())
	case <-t.C:
		return nil
	}
}

func (c *Client) once(ctx context.Context, body []byte, out any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.Endpoint, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("ghapi: build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	if c.Token != "" {
		req.Header.Set("Authorization", "Bearer "+c.Token)
	}
	hc := c.HTTP
	if hc == nil {
		hc = http.DefaultClient
	}
	resp, err := hc.Do(req)
	if err != nil {
		var ue *url.Error
		if errors.As(err, &ue) {
			err = ue.Err
		}
		return fmt.Errorf("ghapi: request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		if rl := rateLimit(resp); rl != nil {
			return rl
		}
		b, _ := io.ReadAll(io.LimitReader(resp.Body, maxBodyBytes))
		return fmt.Errorf("ghapi: unexpected status %d: %s", resp.StatusCode, strings.TrimSpace(string(b)))
	}

	var env struct {
		Data   json.RawMessage `json:"data"`
		Errors []struct {
			Message string `json:"message"`
		} `json:"errors"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&env); err != nil {
		return fmt.Errorf("ghapi: decode response: %w", err)
	}
	if len(env.Errors) > 0 {
		msgs := make([]string, len(env.Errors))
		for i, e := range env.Errors {
			msgs[i] = e.Message
		}
		return &GraphQLError{Messages: msgs}
	}
	if out == nil || len(env.Data) == 0 {
		return nil
	}
	if err := json.Unmarshal(env.Data, out); err != nil {
		return fmt.Errorf("ghapi: decode data: %w", err)
	}
	return nil
}

func rateLimit(resp *http.Response) *RateLimitError {
	if resp.StatusCode != http.StatusForbidden && resp.StatusCode != http.StatusTooManyRequests {
		return nil
	}
	now := time.Now()
	if v := resp.Header.Get("Retry-After"); v != "" {
		if secs, err := strconv.Atoi(strings.TrimSpace(v)); err == nil && secs >= 0 {
			d := time.Duration(secs) * time.Second
			return &RateLimitError{ResetAt: now.Add(d), RetryAfter: d}
		}
	}
	if resp.Header.Get("X-RateLimit-Remaining") == "0" {
		secs, err := strconv.ParseInt(resp.Header.Get("X-RateLimit-Reset"), 10, 64)
		if err != nil {
			return &RateLimitError{ResetAt: now, RetryAfter: maxRetryAfter + time.Second}
		}
		reset := time.Unix(secs, 0)
		return &RateLimitError{ResetAt: reset, RetryAfter: max(reset.Sub(now), 0)}
	}
	return nil
}
