package main

import (
	"fmt"
	"net/http"
	"time"
)

// MaxRetries is the maximum number of retries
const MaxRetries = 1 * 60 * 1000 / 100 // 1 minute

// retryRoundTripper wraps around an existing http.RoundTripper, adding retry logic
type retryRoundTripper struct {
	underlyingTransport http.RoundTripper
}

// RoundTrip executes a single HTTP transaction and retries up to MaxRetries times until a successful response is received
func (rt *retryRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	for i := 0; i < MaxRetries; i++ {
		resp, err := rt.underlyingTransport.RoundTrip(req)
		if err != nil { // Retry on error or server errors
			time.Sleep(100 * time.Millisecond)
			continue
		}
		return resp, err // Return on success or client errors
	}
	return nil, fmt.Errorf("max retries exceeded")
}
