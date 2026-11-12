# Branta Go SDK

Go SDK for the [Branta](https://developer.branta.pro) V2 API.

Use `Verify` on send and `AddPayment` on receive. Status handling and parent platforms are in the [developer docs](https://developer.branta.pro).

## Requirements

Go 1.22 or higher.

## Install

```bash
go get github.com/BrantaOps/branta-go/v4@v4.3.0
```

## Send

```go
service := branta.NewBrantaService(branta.BrantaApiOptions{
    Environment: branta.Production, // Staging everywhere else
})

result, err := service.Verify(ctx, input) // scanned QR or pasted text
if err != nil {
    // Logo on the wrong origin. Do not treat this as unknown.
    return
}

switch result.Status {
case branta.VerifyMatch:
    // Show the card for result.Payments[0]. Link it to result.VerifyURL.
    // For chain URIs (pol:/eth:/bsc:/tron:/ton:/sol:), token is on
    // result.MatchedDestinations[i].Type — do not infer it from the scheme.
case branta.VerifyUnknown:
    // Not in Branta. Show nothing.
case branta.VerifyTampered:
    // Destination was altered. Warn. Do not show a verified card.
case branta.VerifyIncomplete:
    // 5xx, 429, unreachable, transport/client timeout, caller deadline,
    // or malformed lookup. Tell the user.
}
```

On-chain Bitcoin needs a `bitcoin:` URI with `branta_id` and `branta_secret`. Stablecoin rails need a chain URI (`pol:`, `eth:`, `bsc:`, `tron:`, `ton:`, `sol:`) with the same params — token is on `MatchedDestinations[].Type`, not the scheme. bolt11, Ark, and silent payments resolve from the value. A bare address (including bare `0x`) is `unknown`.

A 404 is `unknown`. 5xx, 429, no network, a transport/client timeout, a caller `context` deadline, or a malformed GET body is `incomplete`. Explicit context cancel still returns an error (not a status).

## Receive

```go
service := branta.NewBrantaService(branta.BrantaApiOptions{
    Environment: branta.Production,
    APIKey:      "<api-key>",
})

payment := branta.NewPaymentBuilder().
    SetDescription("Invoice #12345").
    AddDestination("bc1q…", branta.BitcoinAddress).
    // Stablecoins: pass the rail type explicitly — bare 0x / Tron / TON / Solana
    // addresses are not auto-detected.
    // AddDestination("0x…", branta.UsdtPolygon).
    // AddDestination("0x…", branta.UsdtEthereum).
    SetTTL(600).
    Build()

added, err := service.AddPayment(ctx, payment, nil)
```

`AddPayment` encrypts a copy. The caller's `Payment` stays plaintext, so a failed post can be retried with the same value.

Store `added.Secret` with the invoice. Put `branta_id` and `branta_secret` on the invoice QR (`bitcoin:…` or `pol:` / `eth:` / …). `added.VerifyURL` is the link for the wallet.

A parent platform with one shared key sets `APIKey` on the service and calls `SetChildPlatform` for each payment. That is the path for new parent platforms. The deprecated per-client HMAC path is in the developer docs.

## Destination types

`AddPayment` accepts `BitcoinAddress`, `Bolt11`, `ArkAddress`, `SilentPayment`, `UsdtPolygon`, `UsdcPolygon`, `UsdtEthereum`, `UsdcEthereum`, `UsdtBsc`, `UsdcBsc`, `UsdtTron`, `UsdtTon`, `UsdtSolana`, and `UsdcSolana`. Bitcoin and stablecoin addresses use a random UUID secret. bolt11, Ark, and silent payments use a key derived from the lowercased value, so the same input always produces the same lookup token. Pass stablecoin types explicitly — a bare address is not auto-detected. `usdc_ton` is not available yet.

## Development

```bash
go test ./...
BRANTA_RUN_INTEGRATION=1 BRANTA_API_KEY=… go test ./... -run MultiRail
go fmt ./...
go vet ./...
```
