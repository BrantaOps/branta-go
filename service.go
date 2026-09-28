package branta

import (
	"context"
	"strings"
)

// BrantaService orchestrates zero-knowledge encrypt/decrypt around the raw
// HTTP calls in BrantaClient.
//
// The single most important behavioral rule: decryption failures (wrong key,
// mismatched destination, DEK decrypt failure) are always swallowed internally
// — a destination that fails to decrypt is simply left encrypted
// (IsEncrypted = true), never surfaced as an error. The one deliberate
// exception: when a QR code carries both a plaintext on-chain address and
// branta_id/branta_secret params, a successful decrypt of the ZK Bitcoin-address
// destination is compared against that plaintext address, and a mismatch
// propagates ErrTampered.
type BrantaService struct {
	defaultOptions  BrantaClientOptions
	client          Client
	aes             AESEncryption
	secretGenerator SecretGenerator
}

// NewBrantaService is the production constructor: real HTTP client, real
// AES-GCM, real UUID generator.
func NewBrantaService(defaultOptions BrantaClientOptions) *BrantaService {
	return &BrantaService{
		defaultOptions:  defaultOptions,
		client:          NewBrantaClient(defaultOptions),
		aes:             AESEncryptionService{},
		secretGenerator: GuidSecretGenerator{},
	}
}

// NewBrantaServiceWithDeps is the test/advanced constructor: inject alternate
// implementations (e.g. mocks).
func NewBrantaServiceWithDeps(
	defaultOptions BrantaClientOptions,
	client Client,
	aes AESEncryption,
	secretGenerator SecretGenerator,
) *BrantaService {
	return &BrantaService{
		defaultOptions:  defaultOptions,
		client:          client,
		aes:             aes,
		secretGenerator: secretGenerator,
	}
}

func addressesMatch(a, b string) bool {
	isBech32 := func(v string) bool {
		return strings.HasPrefix(strings.ToLower(v), "bc1")
	}
	if isBech32(a) && isBech32(b) {
		return strings.EqualFold(a, b)
	}
	return a == b
}

// GetPayments looks up a single destination value.
func (s *BrantaService) GetPayments(ctx context.Context, destinationValue, destinationEncryptionKey string, options *BrantaClientOptions) (PaymentsResult, error) {
	hashZkType := HashZkType(destinationValue)
	privacy := s.defaultOptions.GetPrivacy(options)

	// Supplying a key at all signals ZK intent and bypasses this check, even
	// for a non-hash-ZK value (e.g. a Bitcoin address looked up with its
	// secret in Strict mode).
	if hashZkType == "" && destinationEncryptionKey == "" && privacy == PrivacyStrict {
		return PaymentsResult{}, ErrPrivacyModeViolation
	}

	normalized := destinationValue
	if hashZkType != "" {
		normalized = strings.ToLower(destinationValue)
	}

	lookupValue := destinationValue
	if hashZkType != "" {
		enc, err := s.aes.Encrypt(normalized, ToNormalizedHash(normalized), true)
		if err != nil {
			return PaymentsResult{}, err
		}
		lookupValue = enc
	}

	payments, err := s.client.GetPayments(ctx, lookupValue, options)
	if err != nil {
		return PaymentsResult{}, err
	}

	// Loose mode only: fall back to a plain-value lookup if the encrypted
	// lookup missed. Strict mode never falls back to plain.
	if len(payments) == 0 && hashZkType != "" && privacy != PrivacyStrict {
		lookupValue = normalized
		payments, err = s.client.GetPayments(ctx, lookupValue, options)
		if err != nil {
			return PaymentsResult{}, err
		}
	}

	keys := newOrderedKeys()
	for i := range payments {
		if err := s.decryptDestinations(ctx, &payments[i], normalized, destinationEncryptionKey, hashZkType, keys, ""); err != nil {
			return PaymentsResult{}, err
		}
	}

	return PaymentsResult{
		Payments:  payments,
		VerifyURL: s.buildVerifyURL(options, lookupValue, keys),
	}, nil
}

// GetPaymentsByQRCode parses raw QR text and looks up matching payments.
func (s *BrantaService) GetPaymentsByQRCode(ctx context.Context, qrText string, options *BrantaClientOptions) (PaymentsResult, error) {
	parser := NewQRParser(qrText)

	if parser.IsOnChainZk() {
		var additionalValues []string
		var onChainAddress string
		for _, d := range parser.Destinations {
			if HashZkType(d.Value) != "" {
				additionalValues = append(additionalValues, d.Value)
			}
			if d.Type == BitcoinAddress && onChainAddress == "" {
				onChainAddress = d.Value
			}
		}
		return s.getPaymentsForZk(ctx, parser.OnChainEncryptionText, parser.OnChainEncryptionSecret, additionalValues, onChainAddress, options)
	}

	destination, ok := parser.Destination()
	if !ok {
		return PaymentsResult{
			Payments:  nil,
			VerifyURL: s.buildVerifyURL(options, "", newOrderedKeys()),
		}, nil
	}

	if s.defaultOptions.GetPrivacy(options) == PrivacyStrict && HashZkType(destination) == "" {
		return PaymentsResult{
			Payments:  nil,
			VerifyURL: s.buildVerifyURL(options, destination, newOrderedKeys()),
		}, nil
	}

	return s.GetPayments(ctx, destination, "", options)
}

func (s *BrantaService) getPaymentsForZk(ctx context.Context, lookupValue, encryptionKey string, additionalHashValues []string, expectedOnChainAddress string, options *BrantaClientOptions) (PaymentsResult, error) {
	payments, err := s.client.GetPayments(ctx, lookupValue, options)
	if err != nil {
		return PaymentsResult{}, err
	}

	keys := newOrderedKeys()
	for i := range payments {
		if err := s.decryptDestinations(ctx, &payments[i], lookupValue, encryptionKey, "", keys, expectedOnChainAddress); err != nil {
			return PaymentsResult{}, err
		}
		for _, value := range additionalHashValues {
			s.decryptHashZkDestinations(ctx, &payments[i], value, keys)
		}
	}

	return PaymentsResult{
		Payments:  payments,
		VerifyURL: s.buildVerifyURL(options, lookupValue, keys),
	}, nil
}

// AddPayment encrypts ZK destinations and posts the payment.
func (s *BrantaService) AddPayment(ctx context.Context, payment Payment, options *BrantaClientOptions) (AddPaymentResult, error) {
	if s.defaultOptions.GetPrivacy(options) == PrivacyStrict {
		for _, d := range payment.Destinations {
			if !d.IsZk {
				return AddPaymentResult{}, ErrNonZkDestinationInStrictMode
			}
		}
	}

	var dek string
	hasZk := false
	for _, d := range payment.Destinations {
		if d.IsZk {
			hasZk = true
			break
		}
	}
	if payment.Metadata != "" && hasZk {
		generated := s.secretGenerator.Generate()
		enc, err := s.aes.Encrypt(payment.Metadata, generated, false)
		if err != nil {
			return AddPaymentResult{}, err
		}
		payment.Metadata = enc
		dek = generated
	}

	secret := s.secretGenerator.Generate()
	encryptedToKey := newOrderedKeys()

	for i := range payment.Destinations {
		destination := &payment.Destinations[i]
		if !destination.IsZk {
			continue
		}

		if destination.Type == BitcoinAddress {
			ciphertext, err := s.aes.Encrypt(destination.Value, secret, s.secretGenerator.DeterministicNonce())
			if err != nil {
				return AddPaymentResult{}, err
			}
			destination.Value = ciphertext
			encryptedToKey.set(ciphertext, secret)
			if dek != "" {
				encDEK, err := s.aes.Encrypt(dek, secret, false)
				if err != nil {
					return AddPaymentResult{}, err
				}
				destination.EncryptedDEK = encDEK
			}
			continue
		}

		if HashZkType(destination.Value) == "" {
			return AddPaymentResult{}, &UnsupportedZkDestinationTypeError{Type: destination.Type}
		}
		normalized := strings.ToLower(destination.Value)
		key := ToNormalizedHash(normalized)
		ciphertext, err := s.aes.Encrypt(normalized, key, true)
		if err != nil {
			return AddPaymentResult{}, err
		}
		destination.Value = ciphertext
		encryptedToKey.set(ciphertext, key)
		if dek != "" {
			encDEK, err := s.aes.Encrypt(dek, key, false)
			if err != nil {
				return AddPaymentResult{}, err
			}
			destination.EncryptedDEK = encDEK
		}
	}

	primaryValue := ""
	if len(payment.Destinations) > 0 {
		primaryValue = payment.Destinations[0].Value
	}

	responsePayment, err := s.client.PostPayment(ctx, payment, options)
	if err != nil {
		return AddPaymentResult{}, err
	}
	if responsePayment == nil {
		return AddPaymentResult{}, ErrNoPaymentReturned
	}

	keys := newOrderedKeys()
	for _, destination := range responsePayment.Destinations {
		if destination.ZkID == "" {
			continue
		}
		if key, ok := encryptedToKey.get(destination.Value); ok {
			keys.set(destination.ZkID, key)
		}
	}

	return AddPaymentResult{
		Payment:   *responsePayment,
		Secret:    secret,
		VerifyURL: s.buildVerifyURL(options, primaryValue, keys),
	}, nil
}

// IsAPIKeyValid proxies to the HTTP client health-check.
func (s *BrantaService) IsAPIKeyValid(ctx context.Context, options *BrantaClientOptions) (bool, error) {
	return s.client.IsAPIKeyValid(ctx, options)
}

func (s *BrantaService) decryptDestinations(
	ctx context.Context,
	payment *Payment,
	destinationValue, encryptionKey string,
	hashZkType DestinationType,
	keys *orderedKeys,
	expectedOnChainAddress string,
) error {
	_ = ctx
	for i := range payment.Destinations {
		isZk := payment.Destinations[i].IsZk
		payment.Destinations[i].IsEncrypted = isZk
		if !isZk {
			continue
		}

		destType := payment.Destinations[i].Type

		if destType == BitcoinAddress {
			if encryptionKey == "" {
				continue
			}
			value := payment.Destinations[i].Value
			plaintext, err := s.aes.Decrypt(value, encryptionKey)
			if err != nil {
				continue
			}
			if expectedOnChainAddress != "" && !addressesMatch(plaintext, expectedOnChainAddress) {
				return ErrTampered
			}
			payment.Destinations[i].Value = plaintext
			payment.Destinations[i].IsEncrypted = false
			if payment.Destinations[i].ZkID != "" {
				if _, exists := keys.get(payment.Destinations[i].ZkID); !exists {
					keys.set(payment.Destinations[i].ZkID, encryptionKey)
				}
			}
			s.tryDecryptMetadata(payment, i, encryptionKey)
			continue
		}

		if hashZkType != "" && destType == hashZkType {
			key := ToNormalizedHash(destinationValue)
			value := payment.Destinations[i].Value
			plaintext, err := s.aes.Decrypt(value, key)
			if err != nil {
				continue
			}
			payment.Destinations[i].Value = plaintext
			payment.Destinations[i].IsEncrypted = false
			if payment.Destinations[i].ZkID != "" {
				if _, exists := keys.get(payment.Destinations[i].ZkID); !exists {
					keys.set(payment.Destinations[i].ZkID, key)
				}
			}
			s.tryDecryptMetadata(payment, i, key)
		}
	}
	return nil
}

func (s *BrantaService) decryptHashZkDestinations(ctx context.Context, payment *Payment, plainValue string, keys *orderedKeys) {
	_ = ctx
	hashZkType := HashZkType(plainValue)
	if hashZkType == "" {
		return
	}
	key := ToNormalizedHash(plainValue)

	for i := range payment.Destinations {
		if !payment.Destinations[i].IsZk || payment.Destinations[i].Type != hashZkType {
			continue
		}
		value := payment.Destinations[i].Value
		plaintext, err := s.aes.Decrypt(value, key)
		if err != nil {
			continue
		}
		payment.Destinations[i].Value = plaintext
		payment.Destinations[i].IsEncrypted = false
		if payment.Destinations[i].ZkID != "" {
			if _, exists := keys.get(payment.Destinations[i].ZkID); !exists {
				keys.set(payment.Destinations[i].ZkID, key)
			}
		}
		s.tryDecryptMetadata(payment, i, key)
	}
}

func (s *BrantaService) tryDecryptMetadata(payment *Payment, index int, keyUsed string) {
	if payment.IsMetadataDecrypted {
		return
	}
	encryptedDEK := payment.Destinations[index].EncryptedDEK
	if encryptedDEK == "" || payment.Metadata == "" {
		return
	}
	dek, err := s.aes.Decrypt(encryptedDEK, keyUsed)
	if err != nil {
		return
	}
	plaintext, err := s.aes.Decrypt(payment.Metadata, dek)
	if err != nil {
		return
	}
	payment.Metadata = plaintext
	payment.IsMetadataDecrypted = true
}

func (s *BrantaService) buildVerifyURL(options *BrantaClientOptions, paymentLookup string, keys *orderedKeys) string {
	baseURL := s.defaultOptions.GetBaseURL(options)
	encoded := pathEscape(paymentLookup)
	u := strings.TrimRight(baseURL, "/") + "/v2/verify/" + encoded
	if !keys.empty() {
		u += ToURLFragment(keys)
	}
	return u
}
