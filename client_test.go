package branta

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func clientFor(server *httptest.Server) *BrantaClient {
	c := NewBrantaClient(NewBrantaClientOptions(Localhost))
	c.baseURLOverride = server.URL
	return c
}

func TestGetPaymentsPercentEncodesPathSegmentIncludingPlusSlashEquals(t *testing.T) {
	ciphertext := "abc+def/ghi="
	encoded := "abc%2Bdef%2Fghi%3D"
	var gotPath string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.EscapedPath()
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte("[]"))
	}))
	defer server.Close()

	result, err := clientFor(server).GetPayments(context.Background(), ciphertext, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(result) != 0 {
		t.Fatal(result)
	}
	if gotPath != "/v2/payments/"+encoded {
		t.Fatalf("path %s", gotPath)
	}
}

func TestGetPaymentsReturnsEmptyOnNon2xx(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(500)
	}))
	defer server.Close()
	result, err := clientFor(server).GetPayments(context.Background(), "value", nil)
	if err != nil || len(result) != 0 {
		t.Fatalf("%v %v", result, err)
	}
}

func TestGetPaymentsReturnsEmptyOnEmptyBody(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	}))
	defer server.Close()
	result, err := clientFor(server).GetPayments(context.Background(), "value", nil)
	if err != nil || len(result) != 0 {
		t.Fatalf("%v %v", result, err)
	}
}

func TestGetPaymentsReturnsEmptyOnMalformedJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("not json"))
	}))
	defer server.Close()
	result, err := clientFor(server).GetPayments(context.Background(), "value", nil)
	if err != nil || len(result) != 0 {
		t.Fatalf("%v %v", result, err)
	}
}

func TestGetPaymentsChecksEveryPaymentsLogoNotJustTheFirst(t *testing.T) {
	body, _ := json.Marshal([]Payment{
		{},
		{PlatformLogoURL: "https://evil.example.com/logo.png"},
	})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write(body)
	}))
	defer server.Close()
	_, err := clientFor(server).GetPayments(context.Background(), "value", nil)
	var mismatch *LogoURLDomainMismatchError
	if !errors.As(err, &mismatch) {
		t.Fatalf("got %v", err)
	}
}

func TestGetPaymentsCatchesMismatchedPlatformLogoLightURL(t *testing.T) {
	body, _ := json.Marshal([]Payment{{PlatformLogoLightURL: "https://evil.example.com/logo-light.png"}})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write(body)
	}))
	defer server.Close()
	_, err := clientFor(server).GetPayments(context.Background(), "value", nil)
	var mismatch *LogoURLDomainMismatchError
	if !errors.As(err, &mismatch) {
		t.Fatalf("got %v", err)
	}
}

func TestGetPaymentsCatchesMismatchedParentPlatformLogoURL(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`[{"parent_platform": {"logo_url": "https://evil.example.com/logo.png"}}]`))
	}))
	defer server.Close()
	_, err := clientFor(server).GetPayments(context.Background(), "value", nil)
	var mismatch *LogoURLDomainMismatchError
	if !errors.As(err, &mismatch) {
		t.Fatalf("got %v", err)
	}
}

func TestGetPaymentsCatchesMismatchedParentPlatformLogoLightURL(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`[{"parent_platform": {"logo_light_url": "https://evil.example.com/logo-light.png"}}]`))
	}))
	defer server.Close()
	_, err := clientFor(server).GetPayments(context.Background(), "value", nil)
	var mismatch *LogoURLDomainMismatchError
	if !errors.As(err, &mismatch) {
		t.Fatalf("got %v", err)
	}
}

func TestGetPaymentsCatchesMismatchedChildPlatformLogoURL(t *testing.T) {
	body, _ := json.Marshal([]Payment{{ChildPlatform: &Platform{LogoURL: "https://evil.example.com/logo.png"}}})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write(body)
	}))
	defer server.Close()
	_, err := clientFor(server).GetPayments(context.Background(), "value", nil)
	var mismatch *LogoURLDomainMismatchError
	if !errors.As(err, &mismatch) {
		t.Fatalf("got %v", err)
	}
}

func TestGetPaymentsCatchesMismatchedChildPlatformLogoLightURL(t *testing.T) {
	body, _ := json.Marshal([]Payment{{ChildPlatform: &Platform{LogoLightURL: "https://evil.example.com/logo-light.png"}}})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write(body)
	}))
	defer server.Close()
	_, err := clientFor(server).GetPayments(context.Background(), "value", nil)
	var mismatch *LogoURLDomainMismatchError
	if !errors.As(err, &mismatch) {
		t.Fatalf("got %v", err)
	}
}

func TestGetPaymentsPassesWhenLogosMatchOrAreAbsent(t *testing.T) {
	var body []byte
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write(body)
	}))
	defer server.Close()

	matching := server.URL + "/logo.png"
	matchingLight := server.URL + "/logo-light.png"
	payload := []map[string]any{
		{},
		{
			"platform_logo_url":       matching,
			"platform_logo_light_url": matchingLight,
			"parent_platform":         map[string]string{"logo_url": matching, "logo_light_url": matchingLight},
			"child_platform":          map[string]string{"logo_url": matching, "logo_light_url": matchingLight},
		},
	}
	body, _ = json.Marshal(payload)

	result, err := clientFor(server).GetPayments(context.Background(), "value", nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(result) != 2 {
		t.Fatalf("len %d", len(result))
	}
}

func TestPostPaymentSendsBearerHeader(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer test-key" {
			t.Errorf("auth %s", r.Header.Get("Authorization"))
		}
		_ = json.NewEncoder(w).Encode(Payment{})
	}))
	defer server.Close()
	opts := NewBrantaClientOptions(Localhost)
	opts.DefaultAPIKey = "test-key"
	c := NewBrantaClient(opts)
	c.baseURLOverride = server.URL
	if _, err := c.PostPayment(context.Background(), Payment{}, nil); err != nil {
		t.Fatal(err)
	}
}

func TestPostPaymentUnauthorizedWithoutMakingAnyRequest(t *testing.T) {
	called := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
	}))
	defer server.Close()
	_, err := clientFor(server).PostPayment(context.Background(), Payment{}, nil)
	if !errors.Is(err, ErrUnauthorized) {
		t.Fatalf("got %v", err)
	}
	if called {
		t.Fatal("should not hit network")
	}
}

func TestPostPaymentIncludesHMACHeadersWhenConfigured(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-HMAC-Signature") == "" || r.Header.Get("X-HMAC-Timestamp") == "" {
			t.Error("missing hmac headers")
		}
		_ = json.NewEncoder(w).Encode(Payment{})
	}))
	defer server.Close()
	opts := NewBrantaClientOptions(Localhost)
	opts.DefaultAPIKey = "test-key"
	opts.HMACSecret = "shared-secret"
	c := NewBrantaClient(opts)
	c.baseURLOverride = server.URL
	if _, err := c.PostPayment(context.Background(), Payment{}, nil); err != nil {
		t.Fatal(err)
	}
}

func TestPostPaymentOmitsHMACHeadersWhenNotConfigured(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-HMAC-Signature") != "" {
			t.Error("unexpected hmac")
		}
		_ = json.NewEncoder(w).Encode(Payment{})
	}))
	defer server.Close()
	opts := NewBrantaClientOptions(Localhost)
	opts.DefaultAPIKey = "test-key"
	c := NewBrantaClient(opts)
	c.baseURLOverride = server.URL
	if _, err := c.PostPayment(context.Background(), Payment{}, nil); err != nil {
		t.Fatal(err)
	}
}

func TestPostPaymentNon2xxReturnsRequestFailedWithStatus(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(401)
	}))
	defer server.Close()
	opts := NewBrantaClientOptions(Localhost)
	opts.DefaultAPIKey = "test-key"
	c := NewBrantaClient(opts)
	c.baseURLOverride = server.URL
	_, err := c.PostPayment(context.Background(), Payment{}, nil)
	var reqErr *RequestFailedError
	if !errors.As(err, &reqErr) || reqErr.Status != 401 {
		t.Fatalf("got %v", err)
	}
}

func TestIsAPIKeyValidTrueOnSuccess(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v2/api-keys/health-check" {
			t.Errorf("path %s", r.URL.Path)
		}
		w.WriteHeader(200)
	}))
	defer server.Close()
	opts := NewBrantaClientOptions(Localhost)
	opts.DefaultAPIKey = "test-key"
	c := NewBrantaClient(opts)
	c.baseURLOverride = server.URL
	ok, err := c.IsAPIKeyValid(context.Background(), nil)
	if err != nil || !ok {
		t.Fatalf("%v %v", ok, err)
	}
}

func TestIsAPIKeyValidFalseOnFailureStatus(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(401)
	}))
	defer server.Close()
	opts := NewBrantaClientOptions(Localhost)
	opts.DefaultAPIKey = "test-key"
	c := NewBrantaClient(opts)
	c.baseURLOverride = server.URL
	ok, err := c.IsAPIKeyValid(context.Background(), nil)
	if err != nil || ok {
		t.Fatalf("%v %v", ok, err)
	}
}

func TestIsAPIKeyValidUnauthorizedWithoutAPIKey(t *testing.T) {
	called := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
	}))
	defer server.Close()
	_, err := clientFor(server).IsAPIKeyValid(context.Background(), nil)
	if !errors.Is(err, ErrUnauthorized) {
		t.Fatalf("got %v", err)
	}
	if called {
		t.Fatal("should not hit network")
	}
}

func TestEscapedPathHelper(t *testing.T) {
	if !strings.Contains(pathEscape("a+b/c="), "%2B") {
		t.Fatal(pathEscape("a+b/c="))
	}
}
