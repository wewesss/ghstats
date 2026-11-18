package collect

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/google/go-github/v93/github"
)

type apiError struct {
	msg string
	err error
}

func (e *apiError) Error() string { return e.msg }

func (e *apiError) Unwrap() error { return e.err }

func sanitize(err error) error {
	if err == nil {
		return nil
	}
	var (
		rl    *github.RateLimitError
		abuse *github.AbuseRateLimitError
		resp  *github.ErrorResponse
		uerr  *url.Error
	)
	switch {
	case errors.As(err, &rl):
		msg := status(rl.Response) + rl.Message
		if !rl.Rate.Reset.IsZero() {
			msg += " [resets at " + rl.Rate.Reset.UTC().Format(time.RFC3339) + "]"
		}
		return &apiError{msg: msg, err: err}
	case errors.As(err, &abuse):
		msg := status(abuse.Response) + abuse.Message
		if abuse.RetryAfter != nil && *abuse.RetryAfter > 0 {
			msg += fmt.Sprintf(" [retry after %s]", abuse.RetryAfter.Round(time.Second))
		}
		return &apiError{msg: msg, err: err}
	case errors.As(err, &resp):
		return &apiError{msg: status(resp.Response) + resp.Message, err: err}
	case errors.As(err, &uerr):
		return &apiError{msg: "request failed: " + uerr.Err.Error(), err: err}
	}
	return err
}

func status(r *http.Response) string {
	if r == nil {
		return ""
	}
	return fmt.Sprintf("%d ", r.StatusCode)
}
