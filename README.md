# Branta Go SDK

Go SDK for the [Branta](https://developer.branta.pro) V2 API.

Status handling and parent platforms are in the [developer docs](https://developer.branta.pro).

## Install

```bash
go get github.com/BrantaOps/branta-go/v4
```

## Send

```go
service := branta.NewBrantaService(branta.BrantaApiOptions{
    Environment: branta.Production,
})

result, err := service.Verify(ctx, input) // scanned QR or pasted text
if err != nil {
    // Logo on the wrong origin, or the caller canceled. Do not treat this as unknown.
    return
}

switch result.Status {
case branta.VerifyMatch:
    // Show the card for result.Payments[0]. Link it to result.VerifyURL.
    // For EVM stablecoins, token is on result.MatchedDestinations[i].Type. Pay from that list.
case branta.VerifyUnknown:
    // Not in Branta. Show nothing.
case branta.VerifyTampered:
    // Destination was altered. Warn. Do not show a verified card.
case branta.VerifyIncomplete:
    // Branta could not complete the check. Tell the user. Do not block send.
}
```

Rules for each status: [Status rules](https://developer.branta.pro/setup/send/status-rules). Supported types: [Destination types](https://developer.branta.pro/destination-types).

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
// Store added.Secret with the invoice. Put branta_id and branta_secret on the QR.
// Link the sender to added.VerifyURL.
```

`AddPayment` accepts the types on [Destination types](https://developer.branta.pro/destination-types). Pass the type for stablecoins. A bare address is not auto-detected. `UsdtTron`, `UsdtTon`, `UsdtSolana`, and `UsdcSolana` also encrypt and post, but no URI verifies them.