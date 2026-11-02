# Branta Go SDK

If you are implementing this SDK in a consumer project: product rules (send vs receive, `match` / `unknown` / `tampered` / `incomplete`, parent platforms) live in the [developer docs](https://developer.branta.pro). `README.md` is the Go install, constructor, and samples.

---

## Developer notes (SDK contributors)

### Package layout

Single Go package `branta` at module root `github.com/BrantaOps/branta-go`. Do not add a `/v2` subdirectory. That collides with Go module major-version import paths.

- `enums.go`, `options.go`, `errors.go`, `extensions.go`, `models.go` — shared primitives.
- `service.go` — `BrantaService`. `Verify` and `AddPayment` are the public entry points.
- `client.go` — raw HTTP client (`BrantaClient`). Do not call it from consumer code.
- `builder.go` — `PaymentBuilder`.
- `parser.go` — `QRParser`.
- `encryption.go` — AES-256-GCM.
- `secret.go` — `GuidSecretGenerator`.
- `examples/branta_example` — runnable demo. README samples should stay aligned with it.
- `integration_test.go` — live production and staging example QR codes, skipped unless `BRANTA_RUN_INTEGRATION=1`. The scheduled workflow runs `Test(Production|Staging)`.

### Build / test

- Test: `go test ./...`
- Live integration tests: `BRANTA_RUN_INTEGRATION=1 go test ./...`
- Format: `go fmt ./...`
- Vet: `go vet ./...`

Publishing a breaking 4.0 as `v4.0.0` requires the module path `github.com/BrantaOps/branta-go/v4`. That cut is separate from this port. Keep developing on `github.com/BrantaOps/branta-go` until the release is cut.

### Key behaviors to preserve

- **`Verify` returns a result.** `match` includes payments and `VerifyURL`. `unknown`, `tampered`, and `incomplete` include only `VerifyURL`. Do not return an error for a swapped address. A 404 is `unknown`. A 5xx, a 429, or no network is `incomplete`. Malformed JSON and `LogoURLDomainMismatchError` still return an error.
- **Unencrypted destinations are not supported.** A raw on-chain address, including testnet `tb1` and `bcrt1`, is `unknown` and does not GET. `AddPayment` encrypts every destination. Types that cannot be encrypted return `UnsupportedDestinationTypeError`.
- **`DestinationType` is only what `AddPayment` accepts:** `bitcoin_address`, `bolt11`, `ark_address`, `silent_payment`. BIP21 extras this SDK cannot encrypt (`lno`, `pay`, non-bolt11 `lightning`) make `Verify` return `unknown`. They are not destination types.
- **`branta_id` / `branta_secret` with no payable destination is `unknown`.** A bolt11, Ark, or silent-payment GET hit is `unknown` unless a destination of that type decrypts to the scanned value, case-insensitively. A different plaintext is `tampered` for the whole call.
- **`VerifyURL` is always returned.** Format: `{apiOrigin}/v2/verify/{lookup}` plus a fragment of per-destination keys on `match`. Fragment names and values use the `encodeURIComponent` set. An empty fragment is `""`. The path segment stays `pathEscape` (unreserved ALPHA / DIGIT / `-._~` only), so `+`, `/`, and `=` in ciphertext are encoded.
- **Encryption.** Bitcoin addresses use a caller-supplied secret (UUID via `GuidSecretGenerator`) and a random nonce. bolt11, Ark, and silent payments use `ToNormalizedHash` and a deterministic nonce. `AddPayment` encrypts a copy and posts that copy. Description is folded into encrypted metadata and lifted after decrypt. The wire destination always sends `zk: true` and `zk_id` from `Destination.ID`.
- **Public names.** `BrantaApiOptions` is the constructor argument and the per-call override (`Environment`, `APIKey`, `HMACSecret`). `BrantaEnvironment`. Context stays the first argument. Do not reintroduce `PrivacyMode`, `GetPayments`, `GetPaymentsByQRCode`, `BrantaClientOptions`, `DefaultAPIKey`, `SetZk`, `Destination.ZkID`, `Payment.DefaultValue`, or `btc_pay_server_plugin_version`.
- **`zk: false` skips decrypt.** Missing `zk` also skips. `zk: true` attempts decrypt.
- **Crypto wire format must match every sibling SDK.** `Encrypt("hello world", "my-secret", true)` == `mPIKHc3ywVlsBHf3Lv2Rwpz2+fKE0kgUePq2m4fPIUidMuGEHVIB`.
- **No retry logic, no configurable timeout.**
- **Logo check** covers every payment and all six logo fields. It runs on GET. `Verify` does not map that error to a status.
- **HMAC.** `POST|{base}/v2/payments|{compact json}|{unix ts}` when a secret is set. Header `X-Branta-Integration-Token` stays for live tests.

### Conventions

- Consumers use `BrantaService`. `Client`, `QRParser`, and AES types live in the same package because Go has no export list. Do not document them as the integration entry.
- `NewBrantaService(options)` for production. `NewBrantaServiceWithDeps` injects `Client`, `AESEncryption`, and `SecretGenerator` for tests.
- `NewQRParser` does not return an error. Unrecognized input is an untyped destination. It drops tab, line feed, and carriage return, plus leading and trailing C0 controls, before it types a destination. A repeated query key keeps the first value.
- Keep parity with `branta-js`, `branta-python`, and `branta-dotnet`.
