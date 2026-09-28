# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [1.0.0] - 2026-09-28

### Added
- Initial release of the Branta Go SDK.
- Feature-parity port of `branta-rust` 3.2.1 (and `branta-dotnet`, `branta-js`, `branta-dart`, `branta-python`, `branta-kotlin`).
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
