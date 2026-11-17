package ghapi_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strconv"
	"sync/atomic"
	"testing"
	"time"

	"github.com/wewesss/ghstats/internal/ghapi"
)

func TestDoDecodesData(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "Bearer tok" {
			t.Errorf("Authorization = %q", got)
		}
		w.Write([]byte(`{"data":{"viewer":{"login":"me"}}}`))
	}))
	defer srv.Close()

	c := &ghapi.Client{HTTP: srv.Client(), Endpoint: srv.URL, Token: "tok"}
	var out struct {
		Viewer struct{ Login string }
	}
	if err := c.Do(t.Context(), "query{viewer{login}}", nil, &out); err != nil {
		t.Fatal(err)
	}
	if out.Viewer.Login != "me" {
		t.Fatalf("login = %q", out.Viewer.Login)
	}
}

func TestDoGraphQLErrors(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"data":null,"errors":[{"message":"boom"},{"message":"bang"}]}`))
	}))
	defer srv.Close()

	c := &ghapi.Client{HTTP: srv.Client(), Endpoint: srv.URL}
	err := c.Do(t.Context(), "q", nil, nil)
	var gqlErr *ghapi.GraphQLError
	if !errors.As(err, &gqlErr) {
		t.Fatalf("err = %v", err)
	}
	if len(gqlErr.Messages) != 2 || gqlErr.Messages[0] != "boom" {
		t.Fatalf("messages = %v", gqlErr.Messages)
	}
}

func TestDoRateLimitRemainingZero(t *testing.T) {
	reset := time.Now().Add(time.Hour).Unix()
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		w.Header().Set("X-RateLimit-Remaining", "0")
		w.Header().Set("X-RateLimit-Reset", strconv.FormatInt(reset, 10))
		w.WriteHeader(http.StatusForbidden)
	}))
	defer srv.Close()

	c := &ghapi.Client{HTTP: srv.Client(), Endpoint: srv.URL}
	err := c.Do(t.Context(), "q", nil, nil)
	var rl *ghapi.RateLimitError
	if !errors.As(err, &rl) {
		t.Fatalf("err = %v", err)
	}
	if rl.ResetAt.Unix() != reset {
		t.Fatalf("ResetAt = %v", rl.ResetAt)
	}
	if rl.RetryAfter < 59*time.Minute {
		t.Fatalf("RetryAfter = %v", rl.RetryAfter)
	}
	if calls.Load() != 1 {
		t.Fatalf("calls = %d, want 1 (no retry for long waits)", calls.Load())
	}
}

func TestDoRetriesAfterRetryAfter(t *testing.T) {
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if calls.Add(1) == 1 {
			w.Header().Set("Retry-After", "1")
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}
		w.Write([]byte(`{"data":{"ok":true}}`))
	}))
	defer srv.Close()

	c := &ghapi.Client{HTTP: srv.Client(), Endpoint: srv.URL}
	var out struct{ OK bool }
	if err := c.Do(t.Context(), "q", nil, &out); err != nil {
		t.Fatal(err)
	}
	if !out.OK || calls.Load() != 2 {
		t.Fatalf("ok=%v calls=%d", out.OK, calls.Load())
	}
}

func TestDoGivesUpAfterTwoRetries(t *testing.T) {
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		w.Header().Set("Retry-After", "0")
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	defer srv.Close()

	c := &ghapi.Client{HTTP: srv.Client(), Endpoint: srv.URL}
	err := c.Do(t.Context(), "q", nil, nil)
	var rl *ghapi.RateLimitError
	if !errors.As(err, &rl) {
		t.Fatalf("err = %v", err)
	}
	if calls.Load() != 3 {
		t.Fatalf("calls = %d, want 3", calls.Load())
	}
}

func TestDoContextCancelledWhileWaiting(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Retry-After", "30")
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	defer srv.Close()

	ctx, cancel := context.WithTimeout(t.Context(), 100*time.Millisecond)
	defer cancel()
	c := &ghapi.Client{HTTP: srv.Client(), Endpoint: srv.URL}
	start := time.Now()
	err := c.Do(ctx, "q", nil, nil)
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("err = %v", err)
	}
	if time.Since(start) > 5*time.Second {
		t.Fatal("did not stop waiting on cancellation")
	}
}

func TestDoContextCanceled(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Retry-After", "30")
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	defer srv.Close()

	ctx, cancel := context.WithCancel(t.Context())
	time.AfterFunc(100*time.Millisecond, cancel)
	c := &ghapi.Client{HTTP: srv.Client(), Endpoint: srv.URL}
	err := c.Do(ctx, "q", nil, nil)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("err = %v", err)
	}
}

func TestDoOtherStatusTruncatesBody(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		for range 100 {
			w.Write([]byte("0123456789"))
		}
	}))
	defer srv.Close()

	c := &ghapi.Client{HTTP: srv.Client(), Endpoint: srv.URL}
	err := c.Do(t.Context(), "q", nil, nil)
	if err == nil || len(err.Error()) > 600 {
		t.Fatalf("err = %v", err)
	}
}
