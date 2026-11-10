package branta_test

import (
	"context"
	"net/url"
	"strings"
	"testing"

	"github.com/BrantaOps/branta-go/v4"
)

// Extra live staging POST→GET combinations beyond the dedicated polygon round-trips.
func TestStagingLiveCombos(t *testing.T) {
	skipLiveWrite(t)
	svc := service(branta.Staging)
	ctx := context.Background()

	const (
		btc   = "bc1qgw3dzmhnyvcswc9r0v0z0ajtp8ulm4nuyeahwr"
		usdt  = "0x278297528c32eCa01295B0d2135984d3442ec963"
		usdc  = "0xAb5801a7D398351b8bE11C439e05C5B3259aeC9B"
		swap  = "0x1111111111111111111111111111111111111111"
		btcSw = "bc1qw508d6qejxtdg4y5r3zarvary0c5xw7kv8f3t4"
	)

	mustAdd := func(t *testing.T, payment branta.Payment) branta.AddPaymentResult {
		t.Helper()
		added, err := svc.AddPayment(ctx, payment, nil)
		if err != nil {
			t.Fatalf("addPayment: %v", err)
		}
		return added
	}

	mustStatus := func(t *testing.T, qr string, want branta.VerifyStatus) branta.VerifyResult {
		t.Helper()
		result, err := svc.Verify(ctx, qr, nil)
		if err != nil || result.Status != want {
			t.Fatalf("verify %s: got %+v err=%v want %s", qr, result, err, want)
		}
		return result
	}

	t.Run("bitcoin round-trip match and tamper", func(t *testing.T) {
		added := mustAdd(t, branta.NewPaymentBuilder().
			AddDestination(btc, branta.BitcoinAddress).
			SetTTL(120).
			Build())
		enc := added.Payment.Destinations[0].Value
		qr := "bitcoin:" + btc + "?branta_id=" + urlQueryEscape(enc) + "&branta_secret=" + urlQueryEscape(added.Secret)
		match := mustStatus(t, qr, branta.VerifyMatch)
		if len(match.MatchedDestinations) != 1 || match.MatchedDestinations[0].Type != branta.BitcoinAddress {
			t.Fatalf("%+v", match.MatchedDestinations)
		}
		tampered := "bitcoin:" + btcSw + "?branta_id=" + urlQueryEscape(enc) + "&branta_secret=" + urlQueryEscape(added.Secret)
		mustStatus(t, tampered, branta.VerifyTampered)
		mustStatus(t, "bitcoin:"+btc, branta.VerifyUnknown)
	})

	t.Run("usdt+usdc different addresses bind own rail", func(t *testing.T) {
		added := mustAdd(t, branta.NewPaymentBuilder().
			AddDestination(usdt, branta.UsdtPolygon).
			AddDestination(usdc, branta.UsdcPolygon).
			SetTTL(120).
			Build())
		var encUsdt, encUsdc string
		for _, d := range added.Payment.Destinations {
			switch d.Type {
			case branta.UsdtPolygon:
				encUsdt = d.Value
			case branta.UsdcPolygon:
				encUsdc = d.Value
			}
		}
		usdtMatch := mustStatus(t, polRoundTripURI(usdt, encUsdt, added.Secret), branta.VerifyMatch)
		if len(usdtMatch.MatchedDestinations) != 1 || usdtMatch.MatchedDestinations[0].Type != branta.UsdtPolygon {
			t.Fatalf("usdt matched: %+v", usdtMatch.MatchedDestinations)
		}
		usdcMatch := mustStatus(t, polRoundTripURI(usdc, encUsdc, added.Secret), branta.VerifyMatch)
		if len(usdcMatch.MatchedDestinations) != 1 || usdcMatch.MatchedDestinations[0].Type != branta.UsdcPolygon {
			t.Fatalf("usdc matched: %+v", usdcMatch.MatchedDestinations)
		}
	})

	t.Run("usdt+usdc same address both match", func(t *testing.T) {
		added := mustAdd(t, branta.NewPaymentBuilder().
			AddDestination(usdt, branta.UsdtPolygon).
			AddDestination(usdt, branta.UsdcPolygon).
			SetTTL(120).
			Build())
		enc := added.Payment.Destinations[0].Value
		match := mustStatus(t, polRoundTripURI(usdt, enc, added.Secret), branta.VerifyMatch)
		if len(match.MatchedDestinations) != 2 {
			t.Fatalf("want both rails: %+v", match.MatchedDestinations)
		}
	})

	t.Run("bitcoin+usdt sibling rails isolate by scheme", func(t *testing.T) {
		added := mustAdd(t, branta.NewPaymentBuilder().
			AddDestination(btc, branta.BitcoinAddress).
			AddDestination(usdt, branta.UsdtPolygon).
			SetTTL(120).
			Build())
		var encBtc, encUsdt string
		for _, d := range added.Payment.Destinations {
			switch d.Type {
			case branta.BitcoinAddress:
				encBtc = d.Value
			case branta.UsdtPolygon:
				encUsdt = d.Value
			}
		}
		btcQR := "bitcoin:" + btc + "?branta_id=" + urlQueryEscape(encBtc) + "&branta_secret=" + urlQueryEscape(added.Secret)
		btcMatch := mustStatus(t, btcQR, branta.VerifyMatch)
		if len(btcMatch.MatchedDestinations) != 1 || btcMatch.MatchedDestinations[0].Type != branta.BitcoinAddress {
			t.Fatalf("bitcoin scan: %+v", btcMatch.MatchedDestinations)
		}
		polMatch := mustStatus(t, polRoundTripURI(usdt, encUsdt, added.Secret), branta.VerifyMatch)
		if len(polMatch.MatchedDestinations) != 1 || polMatch.MatchedDestinations[0].Type != branta.UsdtPolygon {
			t.Fatalf("pol scan: %+v", polMatch.MatchedDestinations)
		}
		// Swapped pol still tampered even with a valid sibling bitcoin rail on the payment.
		mustStatus(t, polRoundTripURI(swap, encUsdt, added.Secret), branta.VerifyTampered)
	})

	t.Run("usdt case-insensitive and wrong secret unknown", func(t *testing.T) {
		added := mustAdd(t, branta.NewPaymentBuilder().
			AddDestination(usdt, branta.UsdtPolygon).
			SetTTL(120).
			Build())
		enc := added.Payment.Destinations[0].Value
		mustStatus(t, polRoundTripURI(strings.ToLower(usdt), enc, added.Secret), branta.VerifyMatch)
		mustStatus(t, polRoundTripURI(usdt, enc, "00000000-0000-0000-0000-000000000000"), branta.VerifyUnknown)
		mustStatus(t, usdt, branta.VerifyUnknown)
		mustStatus(t, "pol:"+usdt+"?value=50&currency=USD", branta.VerifyUnknown)
	})
}

func urlQueryEscape(v string) string {
	return url.QueryEscape(v)
}
