package branta

import "fmt"

// Sentinel errors returned by fallible SDK operations.
//
// Verify does not return ErrTampered or ErrIncomplete. Those are internal:
// Verify turns them into VerifyTampered and VerifyIncomplete. A logo URL on
// the wrong origin still returns LogoURLDomainMismatchError.
var (
	ErrUnauthorized = fmt.Errorf("Unauthorized")

	ErrNoPaymentReturned = fmt.Errorf("No payment returned from server")

	ErrEncryptedDataTooShort = fmt.Errorf("invalid encrypted data: too short")

	errTampered = fmt.Errorf("tampered")

	errIncomplete = fmt.Errorf("incomplete")
)

// RequestFailedError is returned when a POST receives a non-2xx status.
type RequestFailedError struct {
	Status int
}

func (e *RequestFailedError) Error() string {
	return fmt.Sprintf("request failed with status %d", e.Status)
}

// UnsupportedDestinationTypeError is returned when AddPayment is asked to
// encrypt a value that is not a bitcoin address, bolt11, Ark address, or
// silent payment.
type UnsupportedDestinationTypeError struct {
	Type DestinationType
}

func (e *UnsupportedDestinationTypeError) Error() string {
	if e.Type == "" {
		return "destination type '' cannot be encrypted"
	}
	return fmt.Sprintf("destination type '%s' cannot be encrypted", e.Type)
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

// DecryptionError wraps a failed decrypt operation.
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
