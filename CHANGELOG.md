# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [4.2.0] - 2026-11-10

### Added
- `DestinationType.UsdtPolygon` for Polygon USDT. `AddPayment` encrypts the recipient with a random secret (same pattern as bitcoin). `Verify` parses `pol:` URIs with `branta_id` / `branta_secret`; other query params (`value`, `currency`, `user`) are ignored for binding. A bare `0x` address is `unknown`. EVM address compare is case-insensitive.
- `DestinationType.UsdcPolygon` for Polygon USDC. Same secret-key encrypt / `pol:` verify path as `usdt_polygon`. Token is the destination type on the payment; `pol:` remains the Polygon EVM scheme for both.
- `PolygonEvmDestinationTypes`, `IsPolygonEvmDestinationType`, and `IsSecretKeyAddressType`.
- `VerifyResult.MatchedDestinations` on `match` — decrypted rails that bound the scan (token for `pol:` is on `Type`).

### Changed
- `Verify` of a `pol:` URI binds any Polygon EVM family member whose decrypted address matches (match-if-any).
- `pol:` parsing no longer stamps a token `DestinationType` on the QR address. The scheme maps to the Polygon EVM family for verify; the token still comes from the matched payment destination.
- `Verify` swap-checks only the secret-key rail family that matches the QR scheme, so a payment with both bitcoin and `usdt_polygon` no longer returns `tampered` for a valid single-rail QR.
- This release matches `@branta-ops/branta` / Branta.NET 4.2.0.

## [4.0.0] - 2026-11-02

### Removed
- **BREAKING:** `PrivacyMode`, `PrivacyStrict`, and `PrivacyLoose`. Every destination is encrypted. On-chain lookups need `branta_id` and `branta_secret`. bolt11, Ark, and silent payments still resolve from the value.
- **BREAKING:** `PaymentBuilder.SetZk` and `Destination.IsZk`. `AddPayment` assigns `Destination.ID`. Guardrail still receives `zk: true` and `zk_id`.
- **BREAKING:** `GetPayments` and `GetPaymentsByQRCode` on `BrantaService`. Call `Verify` for a QR scan or a paste.
- **BREAKING:** `Verify` returns `VerifyResult` with `Status` of `match`, `unknown`, `tampered`, or `incomplete`. A swapped address is `tampered`. `Payments` is present only on `match`. `PaymentsResult` is gone.
- **BREAKING:** `DestinationType` values for bolt12, LNURL, a Lightning address, and tether. `DestinationType` is what `AddPayment` accepts: `bitcoin_address`, `bolt11`, `ark_address`, `silent_payment`.
- **BREAKING:** `BrantaClientOptions` is `BrantaApiOptions`. `BaseURL` is `Environment`. `BrantaServerBaseURL` is `BrantaEnvironment`. `DefaultAPIKey` is `APIKey`.
- **BREAKING:** `Payment.BtcPayServerPluginVersion` and `Payment.DefaultValue`.
- **BREAKING:** `Destination.ZkID` is `Destination.ID`. Guardrail still receives `zk_id`.

### Added
- `Verify` status `incomplete` when the lookup does not finish (5xx, 429, or no network). A 404 is `unknown`.

### Fixed
- An unset `TTL` is left off the POST. Guardrail then applies its 7-day default. A posted `0` is stored for 30 seconds.
- `Verify` of a bolt11, Ark, or silent payment is `unknown` unless a destination of that type decrypts. A different plaintext is `tampered`. Case differences still match.
- `Verify` of `branta_id` / `branta_secret` with no payable destination is `unknown` and does not look the record up.
- A raw on-chain address, including `tb1` and `bcrt1`, is `unknown` and is not fetched.
- `AddPayment` encrypts a copy of the caller `Payment`. A failed post leaves that object plaintext, so a retry encrypts the original destination again.
- `VerifyURL` fragment names and values are percent-encoded, so a crafted id or secret cannot add parameters.
- A tab, line feed, or carriage return in a QR is stripped before the destination is typed, and a repeated query key keeps the first value.

### Changed
- `SetDescription` writes `metadata.description`. The SDK does not POST or GET Guardrail plaintext `description`. After metadata decrypt, `Payment.Description` is lifted from `metadata.description`.
- This release matches `@branta-ops/branta` 4.0.5.

## [1.0.3] - 2026-10-29

### Fixed
- `GetPaymentsByQRCode` returns a payment only when every payable instruction in the QR matches a decrypted destination on that record. A swapped address with a rewritten `branta_secret`, an added `lightning=`, `ark=`, or `sp=` leg, or a conflicting `bc=` / `tb=` no longer comes back as verified. `bolt12`, `lno=`, `pay=`, and LNURL return no payments ([branta-js#78](https://github.com/BrantaOps/branta-js/pull/78)).

## [1.0.2] - 2026-10-28

### Added
- Recognize signet and regtest Ark addresses (`tark1`) as `ark_address`, alongside mainnet `ark1`. Strict-mode hash-ZK lookup and registration treat them the same way as `ark1`.

## [1.0.0] - 2026-09-28

### Added
- Initial release of the Branta Go SDK.
- Feature-parity port of the 3.2.2 language SDKs (`branta-rust`, `branta-dotnet`, `branta-js`, `branta-dart`, `branta-python`, `branta-kotlin`).
- `BrantaService` with `GetPayments`, `GetPaymentsByQRCode`, `AddPayment`, and `IsAPIKeyValid`.
- `PaymentBuilder` fluent builder with ZK support, metadata encryption, and child platform tagging.
- `QRParser` handles `bitcoin:`/`lightning:` URIs and plain-text values, with full query-string decoding.
- AES-256-GCM encryption with deterministic and random nonce modes (cross-SDK fixed vector covered by tests).
- Zero-knowledge (ZK) destination support for Bitcoin addresses, BOLT-11, Ark, and silent payments.
- Metadata DEK-envelope encryption.
- `PrivacyStrict` (default) and `PrivacyLoose` enforcement.
- HMAC-SHA256 request signing support for parent platform flows.
- `GetPaymentsByQRCode` verifies that the plaintext Bitcoin address parsed from a scanned QR code matches the address decrypted via `branta_id`/`branta_secret`, returning `ErrTampered` on mismatch.
- Logo-URL origin checks for `platform_logo_url`, `platform_logo_light_url`, and parent/child platform logos.
- Unit test coverage via mocked client/AES/secret-generator dependencies (`httptest`).
- Integration tests against staging and production, reusing the same example QR-code fixtures as `branta-python` and `branta-rust`.
