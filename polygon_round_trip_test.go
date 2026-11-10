package branta_test

import (
	"context"
	"net/url"
	"os"
	"strings"
	"testing"

	"github.com/BrantaOps/branta-go/v4"
)

func skipLiveWrite(t *testing.T) {
	t.Helper()
	skipIntegration(t)
	if os.Getenv("BRANTA_API_KEY") == "" {
		t.Skip("Live POST skipped unless BRANTA_API_KEY is set")
	}
}

func polRoundTripURI(recipient, encrypted, secret string) string {
	q := url.Values{}
	q.Set("value", "50")
	q.Set("user", "send")
	q.Set("currency", "USD")
	q.Set("branta_id", encrypted)
	q.Set("branta_secret", secret)
	return "pol:" + recipient + "?" + q.Encode()
}

func TestStagingUsdtPolygonRoundTrip(t *testing.T) {
	skipLiveWrite(t)
	const address = "0x278297528c32eCa01295B0d2135984d3442ec963"
	const swapped = "0x1111111111111111111111111111111111111111"

	svc := service(branta.Staging)
	payment := branta.NewPaymentBuilder().
		AddDestination(address, branta.UsdtPolygon).
		SetTTL(120).
		Build()

	added, err := svc.AddPayment(context.Background(), payment, nil)
	if err != nil {
		t.Fatal(err)
	}
	encrypted := added.Payment.Destinations[0].Value
	if added.Payment.Destinations[0].Type != branta.UsdtPolygon {
		t.Fatalf("%+v", added.Payment.Destinations[0])
	}

	match, err := svc.Verify(context.Background(), polRoundTripURI(address, encrypted, added.Secret), nil)
	if err != nil || match.Status != branta.VerifyMatch {
		t.Fatalf("%+v %v", match, err)
	}
	if match.Payments[0].Destinations[0].Type != branta.UsdtPolygon ||
		!strings.EqualFold(match.Payments[0].Destinations[0].Value, address) {
		t.Fatalf("%+v", match.Payments[0].Destinations[0])
	}

	lower, err := svc.Verify(context.Background(), polRoundTripURI(strings.ToLower(address), encrypted, added.Secret), nil)
	if err != nil || lower.Status != branta.VerifyMatch {
		t.Fatalf("%+v %v", lower, err)
	}

	tampered, err := svc.Verify(context.Background(), polRoundTripURI(swapped, encrypted, added.Secret), nil)
	if err != nil || tampered.Status != branta.VerifyTampered {
		t.Fatalf("%+v %v", tampered, err)
	}

	bare, err := svc.Verify(context.Background(), address, nil)
	if err != nil || bare.Status != branta.VerifyUnknown {
		t.Fatalf("%+v %v", bare, err)
	}

	polBare, err := svc.Verify(context.Background(), "pol:"+address+"?value=50&currency=USD", nil)
	if err != nil || polBare.Status != branta.VerifyUnknown {
		t.Fatalf("%+v %v", polBare, err)
	}
}

func TestStagingUsdcPolygonRoundTrip(t *testing.T) {
	skipLiveWrite(t)
	const address = "0xAb5801a7D398351b8bE11C439e05C5B3259aeC9B"
	const swapped = "0x1111111111111111111111111111111111111111"

	svc := service(branta.Staging)
	payment := branta.NewPaymentBuilder().
		AddDestination(address, branta.UsdcPolygon).
		SetTTL(120).
		Build()

	added, err := svc.AddPayment(context.Background(), payment, nil)
	if err != nil {
		t.Fatal(err)
	}
	encrypted := added.Payment.Destinations[0].Value

	match, err := svc.Verify(context.Background(), polRoundTripURI(address, encrypted, added.Secret), nil)
	if err != nil || match.Status != branta.VerifyMatch {
		t.Fatalf("%+v %v", match, err)
	}
	if match.MatchedDestinations[0].Type != branta.UsdcPolygon {
		t.Fatalf("%+v", match.MatchedDestinations)
	}

	tampered, err := svc.Verify(context.Background(), polRoundTripURI(swapped, encrypted, added.Secret), nil)
	if err != nil || tampered.Status != branta.VerifyTampered {
		t.Fatalf("%+v %v", tampered, err)
	}
}
