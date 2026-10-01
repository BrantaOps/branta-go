# Branta Go SDK

Go SDK for the [Branta](https://branta.pro) V2 API — payment destination lookup and registration with zero-knowledge encryption support.

# Requirements

- Go 1.22 or higher

# Installation

```bash
go get github.com/BrantaOps/branta-go
```

# Integration Guide

If you are an AI agent implementing this SDK, follow the steps below exactly. If you are a developer, use this as a reference for the recommended integration patterns.

**1. Ask which side(s) of a payment they are implementing:**
- **Send side** — the app is paying someone (e.g. a wallet). The user scans or pastes a destination and you verify it belongs to a known platform before funds are sent.
- **Receive side** — the app is receiving payment (e.g. a checkout, POS, invoicing platform). You post destinations to Branta so wallets can verify them.
- **Both** — some apps do both (e.g. an exchange or self-custodial wallet with invoice generation). Implement each side independently.

If they are on the receive side, ask one follow-up:
- **Platform** — single-tenant, one API key.
- **Parent Platform** — multi-tenant, manages multiple child platforms. Two variants:
  - **Shared key (Recommended)** — one API key for all children, no HMAC secret needed; tag the child per-payment with `SetChildPlatform()`. Default to this unless there's a specific reason for separate per-child keys.
  - **Per-client keys** — each child has its own API key, and the parent signs every request with an HMAC secret to prove it originated from the parent; scope requests per-call. Use only if each child needs an independent, separately-revocable API key.

**2. Follow the matching Quick Start section below.**

**3. Apply these rules:**

General (all types):
- Always use `PrivacyStrict`. Never switch to `PrivacyLoose` unless there is no QR scanner and ZK is impossible.
- Never call `BrantaClient` directly — always go through `BrantaService`.
- Never show an error or "not verified" message when a lookup returns empty or an error. An empty result means the destination is unknown to Branta, not that it is malicious. Show nothing.
- For `BaseURL`: use `Production` only in production environments. Use `Staging` everywhere else — including local development, CI, and staging/test environments.

Send side (wallets):
- Prefer `GetPaymentsByQRCode` over `GetPayments` — it handles multi-value ZK QR payloads correctly.
- Only fall back to `GetPayments` for copy/paste flows where there is no QR code.
- If `result.Payments` is empty or the call returns an error, render nothing.
- When `result.Payments` is non-empty, display: the platform logo, the platform name (`payment.Platform`), and the payment description (`payment.Description`). Only render description when non-empty. Make the verification card a clickable link to `result.VerifyURL` — do not display the raw URL.
- For the platform logo: on dark backgrounds use `payment.PlatformLogoURL`. On light backgrounds prefer `payment.PlatformLogoLightURL` when available, falling back to `payment.PlatformLogoURL`.
- Optionally display `payment.ParentPlatform.LogoURL` / `payment.ParentPlatform.LogoLightURL` as a small secondary badge (e.g. corner icon). This is not required.

Receive side (platforms):
- Always call `.SetZk()` on the `PaymentBuilder` before calling `AddPayment`. Plain-text destinations are rejected in `PrivacyStrict` mode.
- Store the `Secret` returned by `AddPayment` alongside the invoice — it is required to reconstruct the verify URL for the wallet.

Receive side (parent platforms — per-client keys), in addition to the platform rules:
- Set `HMACSecret` on `BrantaClientOptions` but leave `DefaultAPIKey` unset at service construction.
- Pass per-call `BrantaClientOptions` with each child's API key to scope requests.

Receive side (parent platforms — shared key), in addition to the platform rules:
- Set `DefaultAPIKey` on `BrantaClientOptions`. Do not set `HMACSecret`.
- Call `.SetChildPlatform(name, logoURL, logoLightURL)` on the builder to tag each payment with the child's branding.

# Quick Start

## For Wallets

Wallets should use `PrivacyStrict`. Two flows are supported:

- **Copy/paste**: call `GetPayments` with the pasted text. Plain-text on-chain addresses will not return results in strict mode — they must be ZK-encoded. Hash-ZK destinations (bolt11, ark_address, silent_payment) work as plain text.
- **QR scan**: call `GetPaymentsByQRCode` with the raw QR text. This handles both on-chain (when the QR includes `branta_id` / `branta_secret`) and hash-ZK destinations.

Always handle the error and show nothing on not-found — a missing record just means the address was not posted to Branta.

```go
package main

import (
	"context"

	"github.com/BrantaOps/branta-go"
)

func lookup(ctx context.Context, service *branta.BrantaService, input string, isQRCode bool) {
	var (
		result branta.PaymentsResult
		err    error
	)
	if isQRCode {
		result, err = service.GetPaymentsByQRCode(ctx, input, nil)
	} else {
		result, err = service.GetPayments(ctx, input, "", nil)
	}

	if err != nil || len(result.Payments) == 0 {
		// Not found — show nothing. The address may simply not exist in Branta.
		return
	}

	// Render result.Payments and result.VerifyURL
}

func main() {
	service := branta.NewBrantaService(branta.BrantaClientOptions{
		BaseURL: branta.Production,
		Privacy: branta.PrivacyStrict,
	})
	lookup(context.Background(), service, "bitcoin:bc1q...", false)
}
```

Prefer `GetPaymentsByQRCode` for QR-driven flows. It handles multi-destination payloads (`branta_id` / `branta_secret` fragments) automatically.

### Looking up a payment by destination value

```go
// Plain bitcoin address (requires PrivacyLoose or will return ErrPrivacyModeViolation):
result, err := service.GetPayments(ctx, "bc1q...", "", nil)

// ZK-encrypted bitcoin address with secret:
result, err = service.GetPayments(ctx, encryptedAddress, "my-secret", nil)

// BOLT-11 invoice (hash-ZK — works in strict mode):
result, err = service.GetPayments(ctx, "lnbc...", "", nil)
```

### No-QR-Code Flows

When QR scanning is not available, three options exist. Choose one based on how much control you want to give users over privacy:

**Option 1 — Keep Strict mode (no code changes)**

Only hash-ZK destinations (bolt11, ark_address, silent_payment) will return results. Plain-text on-chain address lookups return `ErrPrivacyModeViolation`. This is the safest default and requires no additional work.

**Option 2 — Opt-in Loose mode (Recommended)**

Add a user-facing setting (e.g. "Enable on-chain address verification"). Only switch to `PrivacyLoose` when the user explicitly opts in — this sends on-chain addresses in plain text, so the choice should be theirs.

```go
var override *branta.BrantaClientOptions
if userOptedIn {
	opts := branta.BrantaClientOptions{BaseURL: branta.Production, Privacy: branta.PrivacyLoose}
	override = &opts
}

result, err := service.GetPayments(ctx, input, "", override)
```

**Option 3 — Always Loose mode**

Configure with `PrivacyLoose` globally. All lookups including plain-text on-chain addresses are sent to Branta. Simplest, but gives users no privacy control.

```go
service := branta.NewBrantaService(branta.BrantaClientOptions{
	BaseURL: branta.Production,
	Privacy: branta.PrivacyLoose,
})
```

## For Platforms

Platforms post payments to Branta so wallets can verify them. Use `PrivacyStrict` and mark each destination ZK via `.SetZk()` on the `PaymentBuilder`.

```go
payment := branta.NewPaymentBuilder().
	AddDestination("bc1q...", branta.BitcoinAddress).SetZk().
	AddDestination("lnbc...", branta.Bolt11).SetZk().
	SetDescription("Donation").
	AddMetadata("email", "donor@example.com").
	Build()

opts := branta.BrantaClientOptions{
	BaseURL:       branta.Production,
	DefaultAPIKey: "your-api-key",
	Privacy:       branta.PrivacyStrict,
}
result, err := service.AddPayment(ctx, payment, &opts)
// result.Payment — the registered payment from the server
// result.Secret — the random encryption key for the bitcoin address
// result.VerifyURL — share this URL to verify the payment
```

## For Parent Platforms

Choose a variant based on how API keys are structured. Only the per-client keys variant signs requests with HMAC — shared key needs none.

<details>
<summary>Shared key — one API key covers all children (Recommended)</summary>

Construct with a single API key; identify the child platform per-payment.

```go
service := branta.NewBrantaService(branta.BrantaClientOptions{
	BaseURL:       branta.Production,
	DefaultAPIKey: "<shared-api-key>",
	Privacy:       branta.PrivacyStrict,
})

payment := branta.NewPaymentBuilder().
	AddDestination("bc1q...", branta.BitcoinAddress).SetZk().
	SetChildPlatform("ChildBrand", "https://example.com/logo.png", "").
	SetTTL(600).
	Build()

result, err := service.AddPayment(ctx, payment, nil)
```

</details>

<details>
<summary>Per-client keys — each child has its own API key</summary>

Construct the service with the shared HMAC secret only; pass each child's API key per-call.

```go
service := branta.NewBrantaService(branta.BrantaClientOptions{
	BaseURL:    branta.Production,
	HMACSecret: "<hmac-secret>",
	Privacy:    branta.PrivacyStrict,
})

payment := branta.NewPaymentBuilder().
	AddDestination("bc1q...", branta.BitcoinAddress).SetZk().
	SetTTL(600).
	Build()

perCall := branta.BrantaClientOptions{
	BaseURL:       branta.Production,
	DefaultAPIKey: "<child-api-key>",
	Privacy:       branta.PrivacyStrict,
}
result, err := service.AddPayment(ctx, payment, &perCall)
```

</details>

### Validating an API key

```go
isValid, err := service.IsAPIKeyValid(ctx, &opts)
```

### Per-call option overrides

Every public method accepts an optional `*BrantaClientOptions` parameter that overrides the service's default options for that call only, field-by-field:

```go
service := branta.NewBrantaService(defaultOptions)
result, err := service.GetPayments(ctx, "lnbc...", "", &overrideOptions)
```

# Privacy

`PrivacyMode` controls whether plain-text on-chain lookups are allowed.

| Value | Behavior |
|-------|----------|
| `PrivacyStrict` (default) | Only ZK lookups. `GetPayments` returns `ErrPrivacyModeViolation` for plain addresses. `GetPaymentsByQRCode` returns an empty `PaymentsResult` with a populated `VerifyURL`. `AddPayment` returns an error if any destination has `IsZk = false`. |
| `PrivacyLoose` | Both plain and ZK lookups are permitted. |

## ZK destination types

| Type | Encryption |
|------|-----------|
| `BitcoinAddress` | Random secret (UUID) per payment |
| `Bolt11` | Deterministic: SHA-256 of lowercase invoice |
| `ArkAddress` | Deterministic: SHA-256 of lowercase address |
| `SilentPayment` | Deterministic: SHA-256 of lowercase address |

# BrantaService

The primary service type. Always use `BrantaService` — never call `BrantaClient` directly.

**Prefer `GetPaymentsByQRCode` for integrations.** It parses the raw QR text and correctly resolves multiple ZK values in a single scan. `GetPayments` only handles a single destination value and does not support multi-value ZK lookups.

```go
func (s *BrantaService) GetPaymentsByQRCode(ctx context.Context, qrText string, options *BrantaClientOptions) (PaymentsResult, error)
func (s *BrantaService) GetPayments(ctx context.Context, destinationValue, destinationEncryptionKey string, options *BrantaClientOptions) (PaymentsResult, error)
func (s *BrantaService) AddPayment(ctx context.Context, payment Payment, options *BrantaClientOptions) (AddPaymentResult, error)
func (s *BrantaService) IsAPIKeyValid(ctx context.Context, options *BrantaClientOptions) (bool, error)
```

`PaymentsResult` contains the list of matching `Payments` and the `VerifyURL` to display to the user — `VerifyURL` is always returned, even when `Payments` is empty.

# Release

Go modules are versioned by git tags. After merging to `main`:

```bash
git tag v1.0.0
git push origin v1.0.0
```

[`pkg.go.dev/github.com/BrantaOps/branta-go`](https://pkg.go.dev/github.com/BrantaOps/branta-go) indexes the tag automatically. The Go module major version is 1 (import path `github.com/BrantaOps/branta-go`) per [Go module conventions](https://go.dev/ref/mod#major-version-suffixes); functionality matches sibling SDKs at 3.2.2.

# Development

```bash
go test ./...
BRANTA_SKIP_INTEGRATION=1 go test ./...    # skip the live integration suite
go test -run Integration                   # integration tests only (requires network)
go fmt ./...
go vet ./...
go run ./examples/branta_example
```

# Responsible Disclosure

Found critical bugs/vulnerabilities? Please email them to support@branta.pro. Thanks!
