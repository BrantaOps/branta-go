package branta_test

import (
	"context"
	"net/url"
	"strings"
	"testing"

	"github.com/BrantaOps/branta-go/v4"
)

// Live staging AddPayment → Verify for every secret-key rail.
// Gated like polygon_round_trip_test.go.

type secretKeyRail struct {
	name     string
	destType branta.DestinationType
	scheme   string
	address  string
	swapped  string
	caseFold bool
}

func secretKeyRails() []secretKeyRail {
	return []secretKeyRail{
		{
			name: "bitcoin", destType: branta.BitcoinAddress, scheme: "bitcoin",
			address: "bc1qxy2kgdygjrsqtzq2n0yrf2493p83kkfjhx0wlh",
			swapped: "bc1qw508d6qejxtdg4y5r3zarvary0c5xw7kv8f3t4",
		},
		{
			name: "usdt_polygon", destType: branta.UsdtPolygon, scheme: "pol",
			address: "0x278297528c32eCa01295B0d2135984d3442ec963",
			swapped: "0x1111111111111111111111111111111111111111", caseFold: true,
		},
		{
			name: "usdc_polygon", destType: branta.UsdcPolygon, scheme: "pol",
			address: "0xAb5801a7D398351b8bE11C439e05C5B3259aeC9B",
			swapped: "0x2222222222222222222222222222222222222222", caseFold: true,
		},
		{
			name: "usdt_ethereum", destType: branta.UsdtEthereum, scheme: "eth",
			address: "0xBF5550253939187432Ef4dccE8089765d3DCb47f",
			swapped: "0x3333333333333333333333333333333333333333", caseFold: true,
		},
		{
			name: "usdc_ethereum", destType: branta.UsdcEthereum, scheme: "eth",
			address: "0xA0b86991c6218b36c1d19D4a2e9Eb0cE3606eB48",
			swapped: "0x4444444444444444444444444444444444444444", caseFold: true,
		},
		{
			name: "usdt_bsc", destType: branta.UsdtBsc, scheme: "bsc",
			address: "0x55d398326f99059fF775485246999027B3197955",
			swapped: "0x5555555555555555555555555555555555555555", caseFold: true,
		},
		{
			name: "usdc_bsc", destType: branta.UsdcBsc, scheme: "bsc",
			address: "0x8AC76a51cc950d9822D68b83fE1Ad97B32Cd580d",
			swapped: "0x6666666666666666666666666666666666666666", caseFold: true,
		},
		{
			name: "usdt_tron", destType: branta.UsdtTron, scheme: "tron",
			address: "T9yD14Nj9j7xAB4dbGeiX9h8unkKHxuWwb",
			swapped: "TXyZ1111111111111111111111111111111",
		},
		{
			name: "usdt_ton", destType: branta.UsdtTon, scheme: "ton",
			address: "EQDtFpEwcFAEcRe5mLVh2N6C0y-_tur3BYOM93pEZPw390by",
			swapped: "EQAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAM9c",
		},
		{
			name: "usdt_solana", destType: branta.UsdtSolana, scheme: "sol",
			address: "Es9vMFrzaCERmJfrF4H2FYD4KCoNkY11McCe8BenwNYB",
			swapped: "11111111111111111111111111111111",
		},
		{
			name: "usdc_solana", destType: branta.UsdcSolana, scheme: "sol",
			address: "7EqQdEULxWcraVx3mXKFjc84LhCkMGZCkRuDofycyinf",
			swapped: "TokenkegQfeZyiNwAJbNbGKPFXCWuBvf9Ss623VQ5DA",
		},
	}
}

func secretKeyURI(scheme, recipient, encrypted, secret string) string {
	q := url.Values{}
	q.Set("branta_id", encrypted)
	q.Set("branta_secret", secret)
	return scheme + ":" + recipient + "?" + q.Encode()
}

func TestStagingSecretKeyRoundTrip(t *testing.T) {
	skipLiveWrite(t)
	svc := service(branta.Staging)

	for _, rail := range secretKeyRails() {
		rail := rail
		t.Run(rail.name, func(t *testing.T) {
			payment := branta.NewPaymentBuilder().
				AddDestination(rail.address, rail.destType).
				SetTTL(120).
				Build()

			added, err := svc.AddPayment(context.Background(), payment, nil)
			if err != nil {
				t.Fatal(err)
			}
			encrypted := added.Payment.Destinations[0].Value
			if added.Payment.Destinations[0].Type != rail.destType {
				t.Fatalf("%+v", added.Payment.Destinations[0])
			}

			match, err := svc.Verify(context.Background(), secretKeyURI(rail.scheme, rail.address, encrypted, added.Secret), nil)
			if err != nil || match.Status != branta.VerifyMatch {
				t.Fatalf("%+v %v", match, err)
			}
			if len(match.MatchedDestinations) != 1 || match.MatchedDestinations[0].Type != rail.destType {
				t.Fatalf("%+v", match.MatchedDestinations)
			}
			if rail.caseFold {
				if !strings.EqualFold(match.MatchedDestinations[0].Value, rail.address) {
					t.Fatalf("%+v", match.MatchedDestinations[0])
				}
				lower, err := svc.Verify(context.Background(), secretKeyURI(rail.scheme, strings.ToLower(rail.address), encrypted, added.Secret), nil)
				if err != nil || lower.Status != branta.VerifyMatch {
					t.Fatalf("%+v %v", lower, err)
				}
			} else if match.MatchedDestinations[0].Value != rail.address {
				t.Fatalf("%+v", match.MatchedDestinations[0])
			}

			tampered, err := svc.Verify(context.Background(), secretKeyURI(rail.scheme, rail.swapped, encrypted, added.Secret), nil)
			if err != nil || tampered.Status != branta.VerifyTampered {
				t.Fatalf("%+v %v", tampered, err)
			}

			bare, err := svc.Verify(context.Background(), rail.address, nil)
			if err != nil || bare.Status != branta.VerifyUnknown {
				t.Fatalf("%+v %v", bare, err)
			}
		})
	}
}
