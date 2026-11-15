package branta_test

import (
	"context"
	"net/url"
	"strings"
	"testing"

	"github.com/BrantaOps/branta-go/v4"
)

// Extra live staging POST→GET combinations. Skipped unless
// BRANTA_RUN_INTEGRATION=1 and BRANTA_API_KEY is set.
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
			t.Fatalf("verify %.80s: got %+v err=%v want %s", qr, result, err, want)
		}
		return result
	}

	encryptedOf := func(payment branta.Payment, destType branta.DestinationType) string {
		for _, d := range payment.Destinations {
			if d.Type == destType {
				return d.Value
			}
		}
		return ""
	}

	onlyType := func(t *testing.T, result branta.VerifyResult, want branta.DestinationType) {
		t.Helper()
		if len(result.MatchedDestinations) != 1 || result.MatchedDestinations[0].Type != want {
			t.Fatalf("want only %s: %+v", want, result.MatchedDestinations)
		}
	}

	t.Run("bitcoin round-trip match and tamper", func(t *testing.T) {
		added := mustAdd(t, branta.NewPaymentBuilder().
			AddDestination(btc, branta.BitcoinAddress).
			SetTTL(120).
			Build())
		enc := added.Payment.Destinations[0].Value
		onlyType(t, mustStatus(t, bitcoinURI(btc, enc, added.Secret), branta.VerifyMatch), branta.BitcoinAddress)
		mustStatus(t, bitcoinURI(btcSw, enc, added.Secret), branta.VerifyTampered)
		mustStatus(t, "bitcoin:"+btc, branta.VerifyUnknown)
	})

	t.Run("usdt+usdc different addresses bind own rail", func(t *testing.T) {
		added := mustAdd(t, branta.NewPaymentBuilder().
			AddDestination(usdt, branta.UsdtPolygon).
			AddDestination(usdc, branta.UsdcPolygon).
			SetTTL(120).
			Build())
		encUsdt := encryptedOf(added.Payment, branta.UsdtPolygon)
		encUsdc := encryptedOf(added.Payment, branta.UsdcPolygon)
		onlyType(t, mustStatus(t, transferURI(t, branta.UsdtPolygon, usdt, encUsdt, added.Secret), branta.VerifyMatch), branta.UsdtPolygon)
		onlyType(t, mustStatus(t, transferURI(t, branta.UsdcPolygon, usdc, encUsdc, added.Secret), branta.VerifyMatch), branta.UsdcPolygon)
	})

	t.Run("same address under two tokens binds only the scanned token", func(t *testing.T) {
		added := mustAdd(t, branta.NewPaymentBuilder().
			AddDestination(usdt, branta.UsdtPolygon).
			AddDestination(usdt, branta.UsdcPolygon).
			SetTTL(120).
			Build())
		encUsdt := encryptedOf(added.Payment, branta.UsdtPolygon)
		encUsdc := encryptedOf(added.Payment, branta.UsdcPolygon)
		onlyType(t, mustStatus(t, transferURI(t, branta.UsdtPolygon, usdt, encUsdt, added.Secret), branta.VerifyMatch), branta.UsdtPolygon)
		onlyType(t, mustStatus(t, transferURI(t, branta.UsdcPolygon, usdt, encUsdc, added.Secret), branta.VerifyMatch), branta.UsdcPolygon)
		mustStatus(t, transferURI(t, branta.UsdtPolygon, swap, encUsdt, added.Secret), branta.VerifyTampered)
	})

	t.Run("bitcoin+usdt sibling rails isolate by type", func(t *testing.T) {
		added := mustAdd(t, branta.NewPaymentBuilder().
			AddDestination(btc, branta.BitcoinAddress).
			AddDestination(usdt, branta.UsdtPolygon).
			SetTTL(120).
			Build())
		encBtc := encryptedOf(added.Payment, branta.BitcoinAddress)
		encUsdt := encryptedOf(added.Payment, branta.UsdtPolygon)

		btcMatch := mustStatus(t, bitcoinURI(btc, encBtc, added.Secret), branta.VerifyMatch)
		onlyType(t, btcMatch, branta.BitcoinAddress)
		for _, d := range btcMatch.Payments[0].Destinations {
			if d.Type == branta.UsdtPolygon && !d.IsEncrypted {
				t.Fatalf("usdt should stay encrypted: %+v", d)
			}
		}

		usdtMatch := mustStatus(t, transferURI(t, branta.UsdtPolygon, usdt, encUsdt, added.Secret), branta.VerifyMatch)
		onlyType(t, usdtMatch, branta.UsdtPolygon)
		for _, d := range usdtMatch.Payments[0].Destinations {
			if d.Type == branta.BitcoinAddress && !d.IsEncrypted {
				t.Fatalf("bitcoin should stay encrypted: %+v", d)
			}
		}

		// A swapped deposit is still tampered with a valid sibling bitcoin rail on the payment.
		mustStatus(t, transferURI(t, branta.UsdtPolygon, swap, encUsdt, added.Secret), branta.VerifyTampered)
	})

	t.Run("usdt case-insensitive, wrong secret, and removed scheme", func(t *testing.T) {
		added := mustAdd(t, branta.NewPaymentBuilder().
			AddDestination(usdt, branta.UsdtPolygon).
			SetTTL(120).
			Build())
		enc := added.Payment.Destinations[0].Value
		mustStatus(t, transferURI(t, branta.UsdtPolygon, strings.ToLower(usdt), enc, added.Secret), branta.VerifyMatch)
		mustStatus(t, transferURI(t, branta.UsdtPolygon, usdt, enc, "00000000-0000-0000-0000-000000000000"), branta.VerifyUnknown)
		mustStatus(t, usdt, branta.VerifyUnknown)
		// The old pol: scheme no longer verifies, even with valid Branta params.
		q := url.Values{}
		q.Set("branta_id", enc)
		q.Set("branta_secret", added.Secret)
		mustStatus(t, "pol:"+usdt+"?value=50&"+q.Encode(), branta.VerifyUnknown)
	})
}
