# Branta Go SDK

Go SDK at `github.com/BrantaOps/branta-go/v4`. Consumers (wallets, platforms, parent platforms) call `BrantaService` to look up or post payments to Branta. `README.md` is the install, constructor, and samples guide.

## Behavior contract

What `verify` returns and how ERC-681 is parsed: [Status rules](https://developer.branta.pro/setup/send/status-rules). Which rails encrypt with what key: [Encryption](https://developer.branta.pro/setup/receive/encryption). Send handling is on [Send](https://developer.branta.pro/setup/send), and the type list on [Destination types](https://developer.branta.pro/destination-types). Do not restate them here. Change the docs first, then the SDK.

Invariants that break silently. Check them before you change `verify` or `addPayment`:

- A raw on-chain address (including `tb1` / `bcrt1` / bare `0x`) is `unknown` and does not GET.
- A swapped address is `tampered`. `unknown`, `tampered`, and `incomplete` are returned, never thrown.
- The token is `matchedDestinations[].type`, never the URI scheme.
- A logo-domain mismatch throws. Every other lookup failure is a status.
- `addPayment` encrypts a copy and leaves the caller's `Payment` unchanged.
- Keep parity with every 4.x Branta SDK. A public method added or changed here is mirrored in each of them.

## Go specifics

- The contract's names are `Verify`, `AddPayment`, `MatchedDestinations[].Type`, and `VerifyURL`. "Throws" means the call returns an error. `LogoURLDomainMismatchError` is returned, and `Verify` does not map it to a status.
- `Verify` returns an error for an explicit context cancel. A caller context deadline, a client timeout, no network, a 5xx, a 429, or a malformed GET body is `incomplete`.
- `AddPayment` returns `UnsupportedDestinationTypeError` for a type it cannot encrypt.
- `zk: false` skips decrypt. A missing `zk` also skips. `zk: true` attempts decrypt. The wire destination always sends `zk: true` and `zk_id` from `Destination.ID`.
- `VerifyURL` fragment names and values use the `encodeURIComponent` set. An empty fragment is `""`. The path segment stays `pathEscape` (unreserved ALPHA / DIGIT / `-._~` only), so `+`, `/`, and `=` in ciphertext are encoded.
- `DestinationType` is only what `AddPayment` accepts. It also includes `usdt_tron`, `usdt_ton`, `usdt_solana`, and `usdc_solana`. They can be posted but no URI verifies them, and they are not on the public Destination types page. There is no `usdc_ton`.
- Default HTTP timeout is 30s. No retry logic.
- HMAC signs `POST|{base}/v2/payments|{compact json}|{unix ts}` when a secret is set. Prefer a shared `APIKey` plus `SetChildPlatform`. The `X-Branta-Integration-Token` header stays for live tests.
- Context stays the first argument. `BrantaApiOptions` is the constructor argument and the per-call override (`Environment`, `APIKey`; `HMACSecret` is deprecated).
- Do not reintroduce `PrivacyMode`, `GetPayments`, `GetPaymentsByQRCode`, `BrantaClientOptions`, `DefaultAPIKey`, `SetZk`, `Destination.ZkID`, `Payment.DefaultValue`, or `btc_pay_server_plugin_version`.

## Package layout

Single Go package `branta` at module root `github.com/BrantaOps/branta-go/v4`. Do not add a `/v2` subdirectory. That collides with Go module major-version import paths.

- `service.go` — `BrantaService`. `Verify` and `AddPayment` are the public entry points.
- `examples/branta_example` — runnable demo. README samples should stay aligned with it.
- `integration_test.go` — live production and staging example QR codes. The scheduled workflow runs `Test(Production|Staging)`.

## Build / test

- Test: `go test ./...`
- Coverage gate: `go test -coverprofile=coverage.out ./...` then total line coverage ≥80%
- Live integration tests: `BRANTA_RUN_INTEGRATION=1 go test ./...`
- Live multi-rail write: `BRANTA_RUN_INTEGRATION=1 BRANTA_API_KEY=… go test -run MultiRail`
- Format: `go fmt ./...`
- Vet: `go vet ./...`

Release: tag `vX.Y.Z` on the release commit and push the tag. `go get github.com/BrantaOps/branta-go` (without `/v4`) stays on `v1.0.3`.

## Conventions

- Consumers use `BrantaService`. `Client`, `QRParser`, and AES types live in the same package because Go has no export list. Do not document them as the integration entry.
- Do not export the rail table (`evmTokenRails`, `evmTokenType`) or `isSecretKeyAddressType`. `public_api_test.go` locks those names.
- `NewBrantaService(options)` for production. `NewBrantaServiceWithDeps` injects `Client`, `AESEncryption`, and `SecretGenerator` for tests.
- `NewQRParser` does not return an error. Unrecognized input is an untyped destination. It drops tab, line feed, and carriage return, plus leading and trailing C0 controls, before it types a destination. A repeated query key keeps the first value.
