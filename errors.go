package branta

import (
	"fmt"
)

// Sentinel errors returned by fallible SDK operations.
//
// Note what is deliberately absent: a QR-parse error. No sibling SDK ever
// raises one — QRParser degrades gracefully rather than erroring.
//
// Decryption failures during a lookup are always swallowed internally and
// never surface as an error — only AddPayment's HTTP call and the logo-URL
// domain check are allowed to bubble an error out of BrantaService. The one
// decrypt-path exception is ErrTampered.
var (
	ErrPrivacyModeViolation = fmt.Errorf("PrivacyMode Strict does not permit plain-text lookups for this destination type")

	ErrNonZkDestinationInStrictMode = fmt.Errorf("PrivacyMode Strict requires all destinations to be ZK; one or more destinations have is_zk = false")

	ErrUnauthorized = fmt.Errorf("Unauthorized")

	ErrNoPaymentReturned = fmt.Errorf("No payment returned from server")

	ErrEncryptedDataTooShort = fmt.Errorf("invalid encrypted data: too short")

	ErrNoDestinations = fmt.Errorf("Payment has no destinations")

	ErrTampered = fmt.Errorf("The Bitcoin address in the QR code does not match the address verified by Branta. The QR code may have been tampered with.")
)

// RequestFailedError is returned when a POST (or other mutating call) receives
// a non-2xx status.
type RequestFailedError struct {
	Status int
}

func (e *RequestFailedError) Error() string {
	return fmt.Sprintf("request failed with status %d", e.Status)
}

// UnsupportedZkDestinationTypeError is returned when AddPayment is asked to
// encrypt a destination whose type is not a hash-ZK type and is not a Bitcoin
// address.
type UnsupportedZkDestinationTypeError struct {
	Type DestinationType
}

func (e *UnsupportedZkDestinationTypeError) Error() string {
	if e.Type == "" {
		return "destination type <nil> does not support ZK"
	}
	return fmt.Sprintf("destination type %s does not support ZK", e.Type)
}

// LogoURLDomainMismatchError is returned when a GET response contains a logo
// URL whose origin does not match the configured base URL.
type LogoURLDomainMismatchError struct {
	Field string
}

func (e *LogoURLDomainMismatchError) Error() string {
	return fmt.Sprintf("%s domain does not match the configured base_url domain", e.Field)
}

// EncryptionError wraps a failed encrypt operation.
type EncryptionError struct {
	Err error
}

func (e *EncryptionError) Error() string {
	if e.Err == nil {
		return "encryption failed"
	}
	return "encryption failed: " + e.Err.Error()
}

func (e *EncryptionError) Unwrap() error { return e.Err }

// DecryptionError wraps a failed decrypt operation (wrong key, malformed
// ciphertext, etc). Distinct from ErrEncryptedDataTooShort.
type DecryptionError struct {
	Err error
}

func (e *DecryptionError) Error() string {
	if e.Err == nil {
		return "decryption failed"
	}
	return "decryption failed: " + e.Err.Error()
}

func (e *DecryptionError) Unwrap() error { return e.Err }
