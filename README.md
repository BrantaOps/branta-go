# Branta Go SDK

Go SDK for the [Branta](https://developer.branta.pro) V2 API.

Use `Verify` on send and `AddPayment` on receive. Status handling and parent platforms are in the [developer docs](https://developer.branta.pro).

## Requirements

Go 1.22 or higher.

## Install

```bash
go get github.com/BrantaOps/branta-go/v4
```

## Send

```go
service := branta.NewBrantaService(branta.BrantaApiOptions{
    Environment: branta.Production, // Staging everywhere else
})

result, err := service.Verify(ctx, input) // scanned QR or pasted text
if err != nil {
    // Logo on the wrong origin, or the caller canceled. Do not treat this as unknown.
    return
}

switch result.Status {
case branta.VerifyMatch:
    // Show the card for result.Payments[0]. Link it to result.VerifyURL.
    // For EVM stablecoins, token is on result.MatchedDestinations[i].Type
    // (e.g. UsdcBase). Pay from that list.
case branta.VerifyUnknown:
    // Not in Branta. Show nothing.
case branta.VerifyTampered:
    // Destination was altered. Warn. Do not show a verified card.
case branta.VerifyIncomplete:
    // Branta could not complete the check. Tell the user. Do not block send.
}
```

Pass the scanned or pasted text to `Verify`. A Bitcoin or EVM stablecoin lookup needs `branta_id` and `branta_secret` in the URI. bolt11, Ark, and silent payments resolve from the value. A bare address is `unknown`. The full rules are on [Status rules](https://developer.branta.pro/setup/send/status-rules), and supported chains and contracts are on [Destination types](https://developer.branta.pro/destination-types).

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

Store `added.Secret` with the invoice. Put `branta_id` and `branta_secret` on the invoice QR (`bitcoin:…` or the ERC-681 `ethereum:…` URI). `added.VerifyURL` is the link for the wallet.

## Destination types

`AddPayment` accepts the types on [Destination types](https://developer.branta.pro/destination-types), and [Encryption](https://developer.branta.pro/setup/receive/encryption) says which key each uses. `UsdtTron`, `UsdtTon`, `UsdtSolana`, and `UsdcSolana` also encrypt and post, but no URI verifies them.

## Development

```bash
go test ./...
BRANTA_RUN_INTEGRATION=1 BRANTA_API_KEY=… go test ./... -run MultiRail
go fmt ./...
go vet ./...
```
