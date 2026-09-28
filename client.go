package branta

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// Client is the raw HTTP layer. Consumers should never call this directly —
// go through BrantaService.
type Client interface {
	GetPayments(ctx context.Context, destinationValue string, options *BrantaClientOptions) ([]Payment, error)
	PostPayment(ctx context.Context, payment Payment, options *BrantaClientOptions) (*Payment, error)
	IsAPIKeyValid(ctx context.Context, options *BrantaClientOptions) (bool, error)
}

// BrantaClient is the production HTTP client.
type BrantaClient struct {
	http            *http.Client
	defaultOptions  BrantaClientOptions
	baseURLOverride string // test-only: point at httptest.Server
}

// NewBrantaClient constructs a client with the given default options.
func NewBrantaClient(defaultOptions BrantaClientOptions) *BrantaClient {
	return &BrantaClient{
		http:           &http.Client{},
		defaultOptions: defaultOptions,
	}
}

func (c *BrantaClient) baseURL(options *BrantaClientOptions) string {
	if c.baseURLOverride != "" {
		return strings.TrimRight(c.baseURLOverride, "/")
	}
	return strings.TrimRight(c.defaultOptions.GetBaseURL(options), "/")
}

func (c *BrantaClient) resolveAPIKey(options *BrantaClientOptions) (string, error) {
	key := c.defaultOptions.GetAPIKey(options)
	if key == "" {
		return "", ErrUnauthorized
	}
	return key, nil
}

func (c *BrantaClient) hmacHeaders(baseURL, jsonBody string, options *BrantaClientOptions) (signature, timestamp string, ok bool) {
	hmacSecret := c.defaultOptions.GetHMACSecret(options)
	if hmacSecret == "" {
		return "", "", false
	}
	ts := time.Now().Unix()
	message := fmt.Sprintf("POST|%s/v2/payments|%s|%d", strings.TrimRight(baseURL, "/"), jsonBody, ts)
	mac := hmac.New(sha256.New, []byte(hmacSecret))
	mac.Write([]byte(message))
	return hex.EncodeToString(mac.Sum(nil)), fmt.Sprintf("%d", ts), true
}

func (c *BrantaClient) verifyLogoURLs(baseURL string, payments []Payment) error {
	base, err := url.Parse(baseURL)
	if err != nil || base.Scheme == "" || base.Host == "" {
		return nil
	}
	baseOrigin := originOf(base)

	check := func(logoURL, field string) error {
		if logoURL == "" {
			return nil
		}
		u, err := url.Parse(logoURL)
		if err != nil || originOf(u) != baseOrigin {
			return &LogoURLDomainMismatchError{Field: field}
		}
		return nil
	}

	for i := range payments {
		p := &payments[i]
		if err := check(p.PlatformLogoURL, "platform_logo_url"); err != nil {
			return err
		}
		if err := check(p.PlatformLogoLightURL, "platform_logo_light_url"); err != nil {
			return err
		}
		if p.ParentPlatform != nil {
			if err := check(p.ParentPlatform.LogoURL, "parent_platform.logo_url"); err != nil {
				return err
			}
			if err := check(p.ParentPlatform.LogoLightURL, "parent_platform.logo_light_url"); err != nil {
				return err
			}
		}
		if p.ChildPlatform != nil {
			if err := check(p.ChildPlatform.LogoURL, "child_platform.logo_url"); err != nil {
				return err
			}
			if err := check(p.ChildPlatform.LogoLightURL, "child_platform.logo_light_url"); err != nil {
				return err
			}
		}
	}
	return nil
}

func originOf(u *url.URL) string {
	return strings.ToLower(u.Scheme) + "://" + strings.ToLower(u.Host)
}

// GetPayments looks up payments by destination value. Lookup failures
// (transport, non-2xx, empty body, malformed JSON) degrade to an empty slice
// rather than an error. Logo-URL domain mismatches still error.
func (c *BrantaClient) GetPayments(ctx context.Context, destinationValue string, options *BrantaClientOptions) ([]Payment, error) {
	baseURL := c.baseURL(options)
	encoded := pathEscape(destinationValue)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, baseURL+"/v2/payments/"+encoded, nil)
	if err != nil {
		return []Payment{}, nil
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return []Payment{}, nil
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return []Payment{}, nil
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil || len(body) == 0 {
		return []Payment{}, nil
	}
	var payments []Payment
	if err := json.Unmarshal(body, &payments); err != nil {
		return []Payment{}, nil
	}
	if err := c.verifyLogoURLs(baseURL, payments); err != nil {
		return nil, err
	}
	return payments, nil
}

// PostPayment registers a payment. Unlike GetPayments, failures propagate.
func (c *BrantaClient) PostPayment(ctx context.Context, payment Payment, options *BrantaClientOptions) (*Payment, error) {
	baseURL := c.baseURL(options)
	apiKey, err := c.resolveAPIKey(options)
	if err != nil {
		return nil, err
	}
	jsonBody, err := json.Marshal(payment)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, baseURL+"/v2/payments", bytes.NewReader(jsonBody))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", "application/json; charset=utf-8")
	if signature, timestamp, ok := c.hmacHeaders(baseURL, string(jsonBody), options); ok {
		req.Header.Set("X-HMAC-Signature", signature)
		req.Header.Set("X-HMAC-Timestamp", timestamp)
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, &RequestFailedError{Status: resp.StatusCode}
	}
	text, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if len(text) == 0 {
		return nil, nil
	}
	var result Payment
	if err := json.Unmarshal(text, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// IsAPIKeyValid hits the API-key health-check endpoint.
func (c *BrantaClient) IsAPIKeyValid(ctx context.Context, options *BrantaClientOptions) (bool, error) {
	baseURL := c.baseURL(options)
	apiKey, err := c.resolveAPIKey(options)
	if err != nil {
		return false, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, baseURL+"/v2/api-keys/health-check", nil)
	if err != nil {
		return false, err
	}
	req.Header.Set("Authorization", "Bearer "+apiKey)
	resp, err := c.http.Do(req)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()
	return resp.StatusCode >= 200 && resp.StatusCode < 300, nil
}

// pathEscape percent-encodes everything except unreserved characters
// (ALPHA / DIGIT / "-" / "." / "_" / "~"), matching .NET Uri.EscapeDataString.
// Crucially this encodes `+`, `/`, and `=`, which routinely appear in base64
// ciphertext lookup values — QueryEscape is the wrong tool here.
func pathEscape(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	for i := 0; i < len(s); i++ {
		c := s[i]
		if isUnreserved(c) {
			b.WriteByte(c)
			continue
		}
		fmt.Fprintf(&b, "%%%02X", c)
	}
	return b.String()
}

func isUnreserved(c byte) bool {
	if (c >= 'A' && c <= 'Z') || (c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') {
		return true
	}
	switch c {
	case '-', '_', '.', '~':
		return true
	}
	return false
}
