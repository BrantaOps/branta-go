# Branta Go SDK

Go SDK for the [Branta](https://developer.branta.pro) V2 API.

Use `Verify` on send and `AddPayment` on receive. Status handling and parent platforms are in the [developer docs](https://developer.branta.pro).

## Requirements

Go 1.22 or higher.

## Install

```bash
go get github.com/BrantaOps/branta-go
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
case branta.VerifyUnknown:
    // Not in Branta. Show nothing.
case branta.VerifyTampered:
    // Destination was altered. Warn. Do not show a verified card.
case branta.VerifyIncomplete:
    // 5xx, 429, or unreachable. Tell the user. Do not block send.
}
```

On-chain Bitcoin needs a `bitcoin:` URI with `branta_id` and `branta_secret`. bolt11, Ark, and silent payments resolve from the value. A bare address, including testnet `tb1` and `bcrt1`, is `unknown`.

A 404 is `unknown`. A 5xx, a 429, or no network is `incomplete`.

## Receive

```go
service := branta.NewBrantaService(branta.BrantaApiOptions{
    Environment: branta.Production,
    APIKey:      "<api-key>",
})

payment := branta.NewPaymentBuilder().
    SetDescription("Invoice #12345").
    AddDestination("bc1q…", branta.BitcoinAddress).
    SetTTL(600).
    Build()

added, err := service.AddPayment(ctx, payment, nil)
```

`AddPayment` encrypts a copy. The caller's `Payment` stays plaintext, so a failed post can be retried with the same value.

Store `added.Secret` with the invoice. Put `branta_id` and `branta_secret` on the invoice QR. `added.VerifyURL` is the link for the wallet.

A parent platform with one shared key sets `APIKey` on the service and calls `SetChildPlatform` for each payment. That is the path for new parent platforms. The deprecated per-client HMAC path is in the developer docs.

## Destination types

`AddPayment` accepts `BitcoinAddress`, `Bolt11`, `ArkAddress`, and `SilentPayment`. Bitcoin addresses use a random UUID secret. bolt11, Ark, and silent payments use a key derived from the lowercased value, so the same input always produces the same lookup token.

## Development

```bash
go test ./...
BRANTA_RUN_INTEGRATION=1 go test ./...
go fmt ./...
go vet ./...
```
