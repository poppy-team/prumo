package model

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// A provider failure used to reach the gateway as a string, so the gateway had to
// guess what it meant: `isRateLimit` searched the message for "429", "quota",
// "overload" and friends. That is fragile in a way that hides rather than
// surfaces — a vendor whose quota error says "RESOURCE_EXHAUSTED" is invisible
// to the list, so a rate-limited provider is not treated as one, and the
// Retry-After header the provider sent is never read, so the retry comes back
// faster than the limit resets and is rejected again (GAP-108).
//
// The response is parsed once, here, where the status code and the headers still
// exist. What travels onward is a typed error the gateway can classify without
// reading prose.

// ProviderError is a model-provider failure with the facts a retry needs.
type ProviderError struct {
	// Provider names the vendor, so a caller can attribute the failure.
	Provider string
	// StatusCode is the HTTP status, or 0 when the failure was not an HTTP
	// response (a dial failure, a malformed body).
	StatusCode int
	// Message is the provider's own error text.
	Message string
	// RetryAfter is the provider's requested wait, parsed from the Retry-After
	// header. Zero when the provider did not ask for one.
	RetryAfter time.Duration
	// Quota marks a failure that is a quota or rate limit rather than a fault.
	// It is the authoritative signal, replacing substring matching.
	Quota bool
}

// Error reports the failure with enough detail to act on.
func (e *ProviderError) Error() string {
	var b strings.Builder
	if e.Provider != "" {
		b.WriteString(e.Provider)
		b.WriteString(": ")
	}
	if e.StatusCode != 0 {
		fmt.Fprintf(&b, "http %d", e.StatusCode)
	} else {
		b.WriteString("provider error")
	}
	if e.Message != "" {
		b.WriteString(": ")
		b.WriteString(e.Message)
	}
	if e.RetryAfter > 0 {
		fmt.Fprintf(&b, " (retry after %s)", e.RetryAfter)
	}
	return b.String()
}

// NewProviderError builds a provider failure, classifying the quota signals the
// dialects use and reading the retry delay the provider asked for.
func NewProviderError(provider string, resp *http.Response, body string) *ProviderError {
	err := &ProviderError{Provider: provider, Message: body}
	if resp != nil {
		err.StatusCode = resp.StatusCode
		err.RetryAfter = ParseRetryAfter(resp.Header.Get("Retry-After"))
		err.Quota = isQuotaStatus(resp.StatusCode)
	}
	if body != "" {
		// A 200 that carries a resource-exhausted payload is still a quota
		// signal; some compatible gateways return it that way.
		err.Quota = err.Quota || mentionsResourceExhausted(body)
	}
	if err.Quota && err.RetryAfter == 0 {
		// Some providers put the retry hint in the body rather than a header.
		// Google does: a 429 whose details carry retryDelay, with no Retry-After
		// at all. Ignoring that makes the router fall back to its computed backoff
		// for a provider that said exactly when it would be ready, which is the
		// same defect as discarding the header (GAP-108).
		err.RetryAfter = retryDelayFromBody(body)
	}
	return err
}

// retryDelayFromBody reads a retry hint out of a structured error body.
//
// It looks for the shape Google uses — details[] entries with a retryDelay — and
// a bare retryDelay field, rather than parsing the message, because a delay in
// prose is not a number anybody should have to extract with a regular
// expression.
func retryDelayFromBody(body string) time.Duration {
	// The details live under an "error" envelope, not at the top level, and a
	// parser that looked only at the top would find nothing on every real
	// response while passing against a hand-written fixture.
	type retryDetail struct {
		RetryDelay string `json:"retryDelay"`
	}
	var payload struct {
		Details    []retryDetail `json:"details"`
		RetryDelay string        `json:"retryDelay"`
		Error      struct {
			Details    []retryDetail `json:"details"`
			RetryDelay string        `json:"retryDelay"`
		} `json:"error"`
	}
	if err := json.Unmarshal([]byte(body), &payload); err != nil {
		return 0
	}
	hints := append(append(append([]retryDetail{}, payload.Details...), payload.Error.Details...), retryDetail{RetryDelay: payload.RetryDelay})
	for _, hint := range append(hints, retryDetail{RetryDelay: payload.Error.RetryDelay}) {
		if hint.RetryDelay == "" {
			continue
		}
		// Google sends a protobuf duration string: "19s", "1.5s", "0.250s".
		if seconds, err := strconv.ParseFloat(strings.TrimSuffix(hint.RetryDelay, "s"), 64); err == nil && seconds > 0 {
			return time.Duration(seconds * float64(time.Second))
		}
	}
	return 0
}

// ProviderErrorFrom extracts the typed failure from err, if there is one.
func ProviderErrorFrom(err error) (*ProviderError, bool) {
	var pe *ProviderError
	if errors.As(err, &pe) {
		return pe, true
	}
	return nil, false
}

// isQuotaStatus reports whether a status means the provider is asking us to come
// back later rather than that it is broken.
func isQuotaStatus(status int) bool {
	return status == http.StatusTooManyRequests ||
		status == http.StatusServiceUnavailable ||
		status == 425 // Too Early
}

// mentionsResourceExhausted recognises the quota signal by name, for the
// gateways that report one inside a successful response.
func mentionsResourceExhausted(body string) bool {
	lowered := strings.ToLower(body)
	for _, signal := range []string{"resource_exhausted", "resource exhausted", "quota_exceeded", "quota exceeded"} {
		if strings.Contains(lowered, signal) {
			return true
		}
	}
	return false
}

// ParseRetryAfter reads a Retry-After header in either form the RFC allows: a
// number of seconds, or an HTTP date.
//
// It returns zero for anything it cannot read, so a malformed header falls back
// to the configured backoff instead of a made-up delay. It never returns a
// negative duration: a Retry-After in the past means "you may retry now".
func ParseRetryAfter(value string) time.Duration {
	value = strings.TrimSpace(value)
	if value == "" {
		return 0
	}
	if seconds, err := strconv.ParseInt(value, 10, 64); err == nil {
		if seconds <= 0 {
			return 0
		}
		return time.Duration(seconds) * time.Second
	}
	if at, err := http.ParseTime(value); err == nil {
		wait := time.Until(at)
		if wait <= 0 {
			return 0
		}
		return wait
	}
	return 0
}
