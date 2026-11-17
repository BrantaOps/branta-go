package branta_test

import (
	"context"
	"net/url"
	"os"
	"strconv"
	"strings"
	"testing"

	"github.com/BrantaOps/branta-go/v4"
)

// Live staging AddPayment → Verify for every rail. Skipped unless
// BRANTA_RUN_INTEGRATION=1 and BRANTA_API_KEY is set.

func skipLiveWrite(t *testing.T) {
	t.Helper()
	skipIntegration(t)
	if os.Getenv("BRANTA_API_KEY") == "" {
		t.Skip("Live POST skipped unless BRANTA_API_KEY is set")
	}
}

// liveRail is one EVM stablecoin rail, listed independently of the SDK's
// internal table so a wrong contract or chain id in the SDK fails here.
type liveRail struct {
	destType branta.DestinationType
	chainID  int
	contract string
}

var liveRails = []liveRail{
	{branta.UsdcEthereum, 1, "0xA0b86991c6218b36c1d19D4a2e9Eb0cE3606eB48"},
	{branta.UsdtEthereum, 1, "0xdAC17F958D2ee523a2206206994597C13D831ec7"},
	{branta.UsdcPolygon, 137, "0x3c499c542cEF5E3811e1192ce70d8cC03d5c3359"},
	{branta.UsdtPolygon, 137, "0xc2132d05d31c914a87c6611c10748aeb04b58e8f"},
	{branta.UsdcBase, 8453, "0x833589fCD6eDb6E08f4c7C32D4f71b54bdA02913"},
	{branta.UsdtBase, 8453, "0xfde4C96c8593536E31F229EA8f37b2ADa2699bb2"},
	{branta.UsdcArbitrum, 42161, "0xaf88d065e77c8cC2239327C5EDb3A432268e5831"},
	{branta.UsdtArbitrum, 42161, "0xFd086bC7CD5C481DCC9C85ebE478A1C0b69FCbb9"},
	{branta.UsdcBsc, 56, "0x8AC76a51cc950d9822D68b83fE1Ad97B32Cd580d"},
	{branta.UsdtBsc, 56, "0x55d398326f99059fF775485246999027B3197955"},
}

func liveRailFor(t *testing.T, destType branta.DestinationType) liveRail {
	t.Helper()
	for _, rail := range liveRails {
		if rail.destType == destType {
			return rail
		}
	}
	t.Fatalf("no live rail for %s", destType)
	return liveRail{}
}

// transferURI is the ERC-681 transfer URI a wallet would show for one rail.
func transferURI(t *testing.T, destType branta.DestinationType, recipient, encrypted, secret string) string {
	t.Helper()
	rail := liveRailFor(t, destType)
	q := url.Values{}
	q.Set("branta_id", encrypted)
	q.Set("branta_secret", secret)
	return "ethereum:" + rail.contract + "@" + strconv.Itoa(rail.chainID) +
		"/transfer?address=" + recipient + "&uint256=1000000&" + q.Encode()
}

func bitcoinURI(address, encrypted, secret string) string {
	q := url.Values{}
	q.Set("branta_id", encrypted)
	q.Set("branta_secret", secret)
	return "bitcoin:" + address + "?" + q.Encode()
}

// depositFor is a mixed-case deposit per rail, so the lowercase-scan check differs from the stored text.
func depositFor(destType branta.DestinationType) string {
	for i, rail := range liveRails {
		if rail.destType == destType {
			hex := strconv.FormatInt(int64(i+1), 16)
			return "0xAb" + strings.Repeat("1", 38-len(hex)) + hex
		}
	}
	return ""
}

const (
	liveBitcoin        = "bc1qxy2kgdygjrsqtzq2n0yrf2493p83kkfjhx0wlh"
	liveBitcoinSwapped = "bc1qw508d6qejxtdg4y5r3zarvary0c5xw7kv8f3t4"
	liveSwappedEvm     = "0x9999999999999999999999999999999999999999"
)

func TestStagingBitcoinRoundTrip(t *testing.T) {
	skipLiveWrite(t)
	svc := service(branta.Staging)
	ctx := context.Background()

	added, err := svc.AddPayment(ctx, branta.NewPaymentBuilder().
		AddDestination(liveBitcoin, branta.BitcoinAddress).SetTTL(180).Build(), nil)
	if err != nil {
		t.Fatal(err)
	}
	encrypted := added.Payment.Destinations[0].Value
	if encrypted == liveBitcoin {
		t.Fatal("not encrypted")
	}
	match, err := svc.Verify(ctx, bitcoinURI(liveBitcoin, encrypted, added.Secret), nil)
	if err != nil || match.Status != branta.VerifyMatch ||
		len(match.MatchedDestinations) != 1 || match.MatchedDestinations[0].Type != branta.BitcoinAddress ||
		match.MatchedDestinations[0].Value != liveBitcoin {
		t.Fatalf("%+v %v", match, err)
	}
	tampered, err := svc.Verify(ctx, bitcoinURI(liveBitcoinSwapped, encrypted, added.Secret), nil)
	if err != nil || tampered.Status != branta.VerifyTampered {
		t.Fatalf("%+v %v", tampered, err)
	}
	bare, err := svc.Verify(ctx, liveBitcoin, nil)
	if err != nil || bare.Status != branta.VerifyUnknown {
		t.Fatalf("%+v %v", bare, err)
	}
}

func TestStagingEVMRailsRoundTripThroughERC681(t *testing.T) {
	skipLiveWrite(t)
	svc := service(branta.Staging)
	ctx := context.Background()

	for _, rail := range liveRails {
		rail := rail
		t.Run(string(rail.destType), func(t *testing.T) {
			address := depositFor(rail.destType)
			added, err := svc.AddPayment(ctx, branta.NewPaymentBuilder().
				AddDestination(address, rail.destType).SetTTL(180).Build(), nil)
			if err != nil {
				t.Fatal(err)
			}
			encrypted := added.Payment.Destinations[0].Value
			if added.Payment.Destinations[0].Type != rail.destType || encrypted == address {
				t.Fatalf("%+v", added.Payment.Destinations[0])
			}

			match, err := svc.Verify(ctx, transferURI(t, rail.destType, address, encrypted, added.Secret), nil)
			if err != nil || match.Status != branta.VerifyMatch ||
				len(match.MatchedDestinations) != 1 || match.MatchedDestinations[0].Type != rail.destType ||
				match.MatchedDestinations[0].Value != address {
				t.Fatalf("%+v %v", match, err)
			}

			folded, err := svc.Verify(ctx, transferURI(t, rail.destType, strings.ToLower(address), encrypted, added.Secret), nil)
			if err != nil || folded.Status != branta.VerifyMatch {
				t.Fatalf("lowercase: %+v %v", folded, err)
			}

			tampered, err := svc.Verify(ctx, transferURI(t, rail.destType, liveSwappedEvm, encrypted, added.Secret), nil)
			if err != nil || tampered.Status != branta.VerifyTampered {
				t.Fatalf("swapped: %+v %v", tampered, err)
			}

			bare, err := svc.Verify(ctx, address, nil)
			if err != nil || bare.Status != branta.VerifyUnknown {
				t.Fatalf("bare: %+v %v", bare, err)
			}

			noBranta, err := svc.Verify(ctx, "ethereum:"+rail.contract+"@"+strconv.Itoa(rail.chainID)+
				"/transfer?address="+address+"&uint256=1000000", nil)
			if err != nil || noBranta.Status != branta.VerifyUnknown {
				t.Fatalf("no params: %+v %v", noBranta, err)
			}
		})
	}
}

func TestStagingTronTonAndSolanaStillEncryptOnAddPayment(t *testing.T) {
	skipLiveWrite(t)
	svc := service(branta.Staging)
	ctx := context.Background()

	cases := []struct {
		destType branta.DestinationType
		address  string
	}{
		{branta.UsdtTron, "T9yD14Nj9j7xAB4dbGeiX9h8unkKHxuWwb"},
		{branta.UsdtTon, "EQDtFpEwcFAEcRe5mLVh2N6C0y-_tur3BYOM93pEZPw390by"},
		{branta.UsdtSolana, "Es9vMFrzaCERmJfrF4H2FYD4KCoNkY11McCe8BenwNYB"},
		{branta.UsdcSolana, "7EqQdEULxWcraVx3mXKFjc84LhCkMGZCkRuDofycyinf"},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(string(tc.destType), func(t *testing.T) {
			added, err := svc.AddPayment(ctx, branta.NewPaymentBuilder().
				AddDestination(tc.address, tc.destType).SetTTL(180).Build(), nil)
			if err != nil {
				t.Fatal(err)
			}
			if added.Payment.Destinations[0].Type != tc.destType || added.Payment.Destinations[0].Value == tc.address {
				t.Fatalf("%+v", added.Payment.Destinations[0])
			}
			// No verify URI exists for these rails, so a bare address stays unknown.
			bare, err := svc.Verify(ctx, tc.address, nil)
			if err != nil || bare.Status != branta.VerifyUnknown {
				t.Fatalf("%+v %v", bare, err)
			}
		})
	}
}

func TestStagingAnEthereumScanDoesNotBindAPolygonOnlyPayment(t *testing.T) {
	skipLiveWrite(t)
	svc := service(branta.Staging)
	ctx := context.Background()
	address := depositFor(branta.UsdtPolygon)

	added, err := svc.AddPayment(ctx, branta.NewPaymentBuilder().
		AddDestination(address, branta.UsdtPolygon).SetTTL(180).Build(), nil)
	if err != nil {
		t.Fatal(err)
	}
	encrypted := added.Payment.Destinations[0].Value

	cross, err := svc.Verify(ctx, transferURI(t, branta.UsdtEthereum, address, encrypted, added.Secret), nil)
	if err != nil || cross.Status != branta.VerifyUnknown {
		t.Fatalf("cross-chain: %+v %v", cross, err)
	}
	correct, err := svc.Verify(ctx, transferURI(t, branta.UsdtPolygon, address, encrypted, added.Secret), nil)
	if err != nil || correct.Status != branta.VerifyMatch {
		t.Fatalf("correct: %+v %v", correct, err)
	}
}

func TestStagingADualTokenPaymentBindsOnlyTheScannedToken(t *testing.T) {
	skipLiveWrite(t)
	svc := service(branta.Staging)
	ctx := context.Background()
	usdt := depositFor(branta.UsdtEthereum)
	usdc := depositFor(branta.UsdcEthereum)

	added, err := svc.AddPayment(ctx, branta.NewPaymentBuilder().
		AddDestination(usdt, branta.UsdtEthereum).
		AddDestination(usdc, branta.UsdcEthereum).
		SetTTL(180).Build(), nil)
	if err != nil {
		t.Fatal(err)
	}
	var encUsdt string
	for _, d := range added.Payment.Destinations {
		if d.Type == branta.UsdtEthereum {
			encUsdt = d.Value
		}
	}

	match, err := svc.Verify(ctx, transferURI(t, branta.UsdtEthereum, usdt, encUsdt, added.Secret), nil)
	if err != nil || match.Status != branta.VerifyMatch ||
		len(match.MatchedDestinations) != 1 || match.MatchedDestinations[0].Type != branta.UsdtEthereum {
		t.Fatalf("%+v %v", match, err)
	}
	for _, d := range match.Payments[0].Destinations {
		if d.Type == branta.UsdcEthereum && !d.IsEncrypted {
			t.Fatalf("usdc should stay encrypted: %+v", d)
		}
	}

	swapped, err := svc.Verify(ctx, transferURI(t, branta.UsdtEthereum, liveSwappedEvm, encUsdt, added.Secret), nil)
	if err != nil || swapped.Status != branta.VerifyTampered {
		t.Fatalf("swapped: %+v %v", swapped, err)
	}
}
