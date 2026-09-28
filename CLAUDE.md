# Branta Go SDK

If you are implementing this SDK in a consumer project, see the **Integration Guide** section at the top of `README.md` — it covers integration types, recommended flows, and rules.

---

## Developer notes (SDK contributors)

### Package layout

Single Go package `branta` at module root `github.com/BrantaOps/branta-go` (do not add a `/v2` subdirectory — that would collide with Go module major-version import paths).

- `enums.go`, `options.go`, `errors.go`, `extensions.go`, `models.go` — shared primitives.
- `service.go` — `BrantaService`, the public entry point.
- `client.go` — raw HTTP client (`BrantaClient`); do not call directly from consumer code.
- `builder.go` — `PaymentBuilder`.
- `parser.go` — `QRParser` / `QRDestination`.
- `encryption.go` — AES-256-GCM encrypt/decrypt.
- `secret.go` — `SecretGenerator` / `GuidSecretGenerator`.
- `examples/branta_example` — runnable end-to-end demo; README code samples should stay aligned with it.
- `integration_test.go` — live staging/production lookups, skipped when `BRANTA_SKIP_INTEGRATION` is set.

### Build / test / release

- Test: `go test ./...`
- Skip integration tests: `BRANTA_SKIP_INTEGRATION=1 go test ./...`
- Format: `go fmt ./...`
- Vet: `go vet ./...`
- Release: tag `vX.Y.Z` and push; pkg.go.dev indexes automatically. Keep the module path at `github.com/BrantaOps/branta-go` (major version 0 or 1 only). Functionality tracks sibling SDKs at 3.2.x.

### Key behaviors to preserve

- **`PrivacyStrict` is the default.** It forbids plain-text on-chain lookups (`GetPayments` returns `ErrPrivacyModeViolation`; `GetPaymentsByQRCode` returns an empty `PaymentsResult` with a populated `VerifyURL`) and forbids non-ZK destinations on `AddPayment`. `PrivacyLoose` removes those restrictions. Note the nuance in `GetPayments`: supplying *any* `destinationEncryptionKey` bypasses the Strict plain-lookup ban, even for a non-hash-ZK value — only `hashZkType == "" && destinationEncryptionKey == "" && Strict` errors.
- **`VerifyURL` is always returned**, including on a miss.
- **ZK destinations.** Bitcoin addresses are encrypted with one shared random secret per `AddPayment` call (UUID v4 via `GuidSecretGenerator`); hash-ZK types use a deterministic key derived from `SHA-256(lowercase(value))`. **Only `Bolt11`, `ArkAddress`, and `SilentPayment` are hash-ZK types** — `Bolt12`/`LnUrl`/`LnAddress`/`TetherAddress` are valid `DestinationType`s but `HashZkType` must keep returning empty for them. Don't "fix" this.
- **Decryption failures are silently swallowed** inside `BrantaService` — a destination that fails to decrypt is simply left with `IsEncrypted = true`, never surfaced as an error. Metadata/DEK decryption follows the same rule, and is only attempted once per payment (guarded by `payment.IsMetadataDecrypted`).
- **GET lookups never error.** Non-2xx, empty body, malformed JSON, or a transport failure all degrade to an empty slice inside `BrantaClient.GetPayments` — never propagate. Only `AddPayment`'s POST and the logo-domain check are allowed to return a real error.
- **Crypto wire format must byte-for-byte match every sibling SDK.** Checked-in cross-SDK fixed vector: `Encrypt("hello world", "my-secret", true)` == `mPIKHc3ywVlsBHf3Lv2Rwpz2+fKE0kgUePq2m4fPIUidMuGEHVIB`.
- **No retry logic, no configurable timeout.** This was explicitly removed from `branta-dotnet`'s history — don't reintroduce either.
- **Path encoding** must match .NET `Uri.EscapeDataString` (unreserved = ALPHA / DIGIT / `-` `.` `_` `~`). `url.PathEscape` / `url.QueryEscape` are the wrong tools — they would leave `+`/`=` unencoded or turn `+` into space.

### Known cross-SDK inconsistencies (do not "fix" unilaterally — coordinate across all sibling repos first)

- `branta-kotlin` serializes the BTCPay field as `btcpay_server_plugin_version`; every other SDK (including this one) uses `btc_pay_server_plugin_version`.

### Logo-URL domain validation

`BrantaClient.verifyLogoURLs` checks every payment in a GET response (never just the first), and every logo-bearing field: `platform_logo_url`, `platform_logo_light_url`, `parent_platform.logo_url`/`logo_light_url`, and `child_platform.logo_url`/`logo_light_url`. A mismatch returns `LogoURLDomainMismatchError` naming the offending field.

### Conventions

- Public API is flat at package `branta` (`branta.NewBrantaService`, etc.).
- `BrantaClientOptions` can be passed per-call as `*BrantaClientOptions` to override constructor defaults, field-by-field (`GetBaseURL`/`GetPrivacy`/`GetAPIKey`/`GetHMACSecret`).
- `NewBrantaService(options)` for production use; `NewBrantaServiceWithDeps(...)` injecting `Client`/`AESEncryption`/`SecretGenerator` for tests.
- `NewQRParser` is infallible by design — no sibling SDK ever raises a QR-parse error; unrecognized input just yields an untyped destination.
- HTTP methods take `context.Context` as the first argument (Go equivalent of cancellation tokens / coroutine cancellation).
- Keep parity with `branta-dotnet`, `branta-js`, `branta-dart`, `branta-python`, `branta-kotlin`, `branta-rust`: any new method, option, or enum value here must be reflected in all six, and vice versa.
