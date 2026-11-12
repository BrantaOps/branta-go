package branta

import (
	"context"
	"encoding/json"
	"errors"
	"strings"

	"github.com/google/uuid"
)

// BrantaService orchestrates encrypt and decrypt around BrantaClient.
//
// Verify returns a status. A swapped address or a derived-key plaintext that
// is not the scanned value is VerifyTampered. A 404 is VerifyUnknown. Any
// other non-OK response, a network failure / timeout, or a malformed GET body
// is VerifyIncomplete. A logo URL on the wrong origin still returns
// LogoURLDomainMismatchError.
//
// AddPayment encrypts a copy. A failed post leaves the caller's Payment plaintext.
type BrantaService struct {
	defaultOptions  BrantaApiOptions
	client          Client
	aes             AESEncryption
	secretGenerator SecretGenerator
}

// NewBrantaService is the production constructor.
func NewBrantaService(defaultOptions BrantaApiOptions) *BrantaService {
	return &BrantaService{
		defaultOptions:  defaultOptions,
		client:          NewBrantaClient(defaultOptions),
		aes:             AESEncryptionService{},
		secretGenerator: GuidSecretGenerator{},
	}
}

// NewBrantaServiceWithDeps injects a client, AES, and secret generator.
func NewBrantaServiceWithDeps(
	defaultOptions BrantaApiOptions,
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
		lower := strings.ToLower(v)
		return strings.HasPrefix(lower, "bc1") || strings.HasPrefix(lower, "tb1") || strings.HasPrefix(lower, "bcrt1")
	}
	if isBech32(a) && isBech32(b) {
		return strings.EqualFold(a, b)
	}
	if strings.HasPrefix(strings.ToLower(a), "0x") && strings.HasPrefix(strings.ToLower(b), "0x") {
		return strings.EqualFold(a, b)
	}
	return a == b
}

// uniformOnChainAddress returns the single secret-key address every secret-key
// instruction agrees on. Conflicting addresses are errTampered.
func uniformOnChainAddress(parser QRParser) (string, error) {
	first := ""
	for _, d := range parser.Destinations {
		if !d.SecretKey || strings.TrimSpace(d.Value) == "" {
			continue
		}
		if first == "" {
			first = d.Value
			continue
		}
		if !addressesMatch(first, d.Value) {
			return "", errTampered
		}
	}
	return first, nil
}

func expectedSecretKeyTypes(parser QRParser) map[DestinationType]struct{} {
	if len(parser.ExpectedSecretKeyTypes) == 0 {
		return nil
	}
	out := make(map[DestinationType]struct{}, len(parser.ExpectedSecretKeyTypes))
	for _, t := range parser.ExpectedSecretKeyTypes {
		out[t] = struct{}{}
	}
	return out
}

func matchedDestinationsFrom(payments []Payment) []Destination {
	var out []Destination
	for _, payment := range payments {
		for _, destination := range payment.Destinations {
			if !destination.IsEncrypted {
				out = append(out, destination)
			}
		}
	}
	return out
}

func toVerifyResult(payments []Payment, verifyURL string) VerifyResult {
	if len(payments) == 0 {
		return VerifyResult{Status: VerifyUnknown, VerifyURL: verifyURL}
	}
	return VerifyResult{
		Status:              VerifyMatch,
		Payments:            payments,
		MatchedDestinations: matchedDestinationsFrom(payments),
		VerifyURL:           verifyURL,
	}
}

// Verify parses a scanned QR or pasted text and looks it up.
func (s *BrantaService) Verify(ctx context.Context, input string, options *BrantaApiOptions) (VerifyResult, error) {
	parser := NewQRParser(input)
	lookup := parser.OnChainEncryptionText
	if lookup == "" {
		lookup, _ = parser.Destination()
	}

	result, err := s.verifyParsed(ctx, parser, options)
	if err == nil {
		return result, nil
	}
	if errors.Is(err, errTampered) {
		return VerifyResult{Status: VerifyTampered, VerifyURL: s.buildVerifyURL(options, lookup, nil)}, nil
	}
	if errors.Is(err, errIncomplete) {
		return VerifyResult{Status: VerifyIncomplete, VerifyURL: s.buildVerifyURL(options, lookup, nil)}, nil
	}
	return VerifyResult{}, err
}

func (s *BrantaService) verifyParsed(ctx context.Context, parser QRParser, options *BrantaApiOptions) (VerifyResult, error) {
	expectedOnChainAddress, err := uniformOnChainAddress(parser)
	if err != nil {
		return VerifyResult{}, err
	}

	lookup := parser.OnChainEncryptionText
	if lookup == "" {
		lookup, _ = parser.Destination()
	}

	if parser.HasUnverifiableInstruction {
		return VerifyResult{Status: VerifyUnknown, VerifyURL: s.buildVerifyURL(options, lookup, nil)}, nil
	}

	if parser.IsOnChainZk() {
		var additionalValues []string
		for _, d := range parser.Destinations {
			if DerivedKeyType(d.Value) != "" {
				additionalValues = append(additionalValues, d.Value)
			}
		}
		if expectedOnChainAddress == "" && len(additionalValues) == 0 {
			return VerifyResult{Status: VerifyUnknown, VerifyURL: s.buildVerifyURL(options, lookup, nil)}, nil
		}
		payments, verifyURL, err := s.lookupByBrantaID(
			ctx,
			parser.OnChainEncryptionText,
			parser.OnChainEncryptionSecret,
			additionalValues,
			expectedOnChainAddress,
			expectedSecretKeyTypes(parser),
			options,
		)
		if err != nil {
			return VerifyResult{}, err
		}
		return toVerifyResult(payments, verifyURL), nil
	}

	destination, _ := parser.Destination()
	if DerivedKeyType(destination) == "" {
		return VerifyResult{Status: VerifyUnknown, VerifyURL: s.buildVerifyURL(options, destination, nil)}, nil
	}

	payments, verifyURL, err := s.lookupByDerivedKey(ctx, destination, options)
	if err != nil {
		return VerifyResult{}, err
	}
	return toVerifyResult(payments, verifyURL), nil
}

func (s *BrantaService) lookupByBrantaID(
	ctx context.Context,
	lookupValue, encryptionKey string,
	additionalHashValues []string,
	expectedOnChainAddress string,
	expectedOnChainTypes map[DestinationType]struct{},
	options *BrantaApiOptions,
) ([]Payment, string, error) {
	payments, err := s.client.GetPayments(ctx, lookupValue, options)
	if err != nil {
		return nil, "", err
	}

	keys := newOrderedKeys()
	var bound []Payment
	for i := range payments {
		onChainBound, err := s.decryptDestinations(
			&payments[i],
			encryptionKey,
			keys,
			expectedOnChainAddress,
			expectedOnChainTypes,
		)
		if err != nil {
			return nil, "", err
		}
		hashesBound := true
		for _, value := range additionalHashValues {
			ok, err := s.decryptDerivedKeyDestinations(&payments[i], value, keys)
			if err != nil {
				return nil, "", err
			}
			if !ok {
				hashesBound = false
			}
		}
		hasPayable := expectedOnChainAddress != "" || len(additionalHashValues) > 0
		if hasPayable && onChainBound && hashesBound {
			bound = append(bound, payments[i])
		}
	}
	return bound, s.buildVerifyURL(options, lookupValue, keys), nil
}

func (s *BrantaService) lookupByDerivedKey(ctx context.Context, destinationValue string, options *BrantaApiOptions) ([]Payment, string, error) {
	normalized := strings.ToLower(destinationValue)
	lookupValue, err := s.aes.Encrypt(normalized, ToNormalizedHash(normalized), true)
	if err != nil {
		return nil, "", err
	}

	payments, err := s.client.GetPayments(ctx, lookupValue, options)
	if err != nil {
		return nil, "", err
	}

	keys := newOrderedKeys()
	var bound []Payment
	for i := range payments {
		ok, err := s.decryptDerivedKeyDestinations(&payments[i], normalized, keys)
		if err != nil {
			return nil, "", err
		}
		if ok {
			bound = append(bound, payments[i])
		}
	}
	return bound, s.buildVerifyURL(options, lookupValue, keys), nil
}

// AddPayment encrypts a copy of payment and posts that copy.
func (s *BrantaService) AddPayment(ctx context.Context, payment Payment, options *BrantaApiOptions) (AddPaymentResult, error) {
	submitted := copyPayment(payment)
	foldDescriptionIntoMetadata(&submitted)

	var dek string
	if submitted.Metadata != "" && len(submitted.Destinations) > 0 {
		dek = s.secretGenerator.Generate()
		enc, err := s.aes.Encrypt(submitted.Metadata, dek, false)
		if err != nil {
			return AddPaymentResult{}, err
		}
		submitted.Metadata = enc
	}

	secret := s.secretGenerator.Generate()
	encryptedToKey := newOrderedKeys()

	for i := range submitted.Destinations {
		destination := &submitted.Destinations[i]
		if destination.ID == "" {
			destination.ID = uuid.New().String()
		}

		destType := destination.Type
		if destType == "" {
			destType = detectPlainTextType(destination.Value)
			if destType != "" {
				destination.Type = destType
			}
		}

		if isSecretKeyAddressType(destType) {
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

		if DerivedKeyType(destination.Value) == "" {
			return AddPaymentResult{}, &UnsupportedDestinationTypeError{Type: destination.Type}
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
	if len(submitted.Destinations) > 0 {
		primaryValue = submitted.Destinations[0].Value
	}

	responsePayment, err := s.client.PostPayment(ctx, submitted, options)
	if err != nil {
		return AddPaymentResult{}, err
	}
	if responsePayment == nil {
		return AddPaymentResult{}, ErrNoPaymentReturned
	}

	keys := newOrderedKeys()
	for _, destination := range responsePayment.Destinations {
		if destination.ID == "" {
			continue
		}
		if key, ok := encryptedToKey.get(destination.Value); ok {
			keys.set(destination.ID, key)
		}
	}

	return AddPaymentResult{
		Payment:   *responsePayment,
		Secret:    secret,
		VerifyURL: s.buildVerifyURL(options, primaryValue, keys),
	}, nil
}

// IsAPIKeyValid proxies to the HTTP client health-check.
func (s *BrantaService) IsAPIKeyValid(ctx context.Context, options *BrantaApiOptions) (bool, error) {
	return s.client.IsAPIKeyValid(ctx, options)
}

func (s *BrantaService) decryptDestinations(
	payment *Payment,
	encryptionKey string,
	keys *orderedKeys,
	expectedOnChainAddress string,
	expectedOnChainTypes map[DestinationType]struct{},
) (bool, error) {
	onChainBound := expectedOnChainAddress == ""
	decryptedExpectedRail := false
	for i := range payment.Destinations {
		if !payment.Destinations[i].IsEncrypted {
			continue
		}
		if !isSecretKeyAddressType(payment.Destinations[i].Type) {
			continue
		}
		// Outside this QR's rail family (e.g. bitcoin when scanning pol:) — leave encrypted.
		if expectedOnChainTypes != nil {
			if _, ok := expectedOnChainTypes[payment.Destinations[i].Type]; !ok {
				continue
			}
		}
		if encryptionKey == "" {
			continue
		}
		plaintext, err := s.aes.Decrypt(payment.Destinations[i].Value, encryptionKey)
		if err != nil {
			continue
		}
		decryptedExpectedRail = true
		// Match-if-any within the family: a sibling rail with a different address stays encrypted.
		if expectedOnChainAddress != "" && !addressesMatch(plaintext, expectedOnChainAddress) {
			continue
		}
		if expectedOnChainAddress != "" {
			onChainBound = true
		}
		payment.Destinations[i].Value = plaintext
		payment.Destinations[i].IsEncrypted = false
		if payment.Destinations[i].ID != "" {
			if _, exists := keys.get(payment.Destinations[i].ID); !exists {
				keys.set(payment.Destinations[i].ID, encryptionKey)
			}
		}
		s.tryDecryptMetadata(payment, i, encryptionKey)
	}
	// Tampered only when a family rail decrypted and none matched the scanned address.
	if expectedOnChainAddress != "" && !onChainBound && decryptedExpectedRail {
		return false, errTampered
	}
	return onChainBound, nil
}

// decryptDerivedKeyDestinations reports whether a destination of plainValue's
// type decrypted to that value. A different plaintext is errTampered.
func (s *BrantaService) decryptDerivedKeyDestinations(payment *Payment, plainValue string, keys *orderedKeys) (bool, error) {
	derived := DerivedKeyType(plainValue)
	if derived == "" {
		return false, nil
	}
	key := ToNormalizedHash(plainValue)
	bound := false
	for i := range payment.Destinations {
		if !payment.Destinations[i].IsEncrypted || payment.Destinations[i].Type != derived {
			continue
		}
		plaintext, err := s.aes.Decrypt(payment.Destinations[i].Value, key)
		if err != nil {
			continue
		}
		if !strings.EqualFold(plaintext, plainValue) {
			return false, errTampered
		}
		payment.Destinations[i].Value = plaintext
		payment.Destinations[i].IsEncrypted = false
		if payment.Destinations[i].ID != "" {
			if _, exists := keys.get(payment.Destinations[i].ID); !exists {
				keys.set(payment.Destinations[i].ID, key)
			}
		}
		s.tryDecryptMetadata(payment, i, key)
		bound = true
	}
	return bound, nil
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
	liftDescriptionFromMetadata(payment)
}

func (s *BrantaService) buildVerifyURL(options *BrantaApiOptions, paymentLookup string, keys *orderedKeys) string {
	baseURL := s.defaultOptions.GetBaseURL(options)
	u := strings.TrimRight(baseURL, "/") + "/v2/verify/" + pathEscape(paymentLookup)
	if !keys.empty() {
		u += ToURLFragment(keys)
	}
	return u
}

func copyPayment(p Payment) Payment {
	out := p
	if p.Destinations != nil {
		out.Destinations = make([]Destination, len(p.Destinations))
		copy(out.Destinations, p.Destinations)
	}
	if p.ParentPlatform != nil {
		pp := *p.ParentPlatform
		out.ParentPlatform = &pp
	}
	if p.ChildPlatform != nil {
		cp := *p.ChildPlatform
		out.ChildPlatform = &cp
	}
	return out
}

func foldDescriptionIntoMetadata(payment *Payment) {
	if payment.Description == "" {
		return
	}
	metadataMap := map[string]string{}
	if payment.Metadata != "" {
		_ = json.Unmarshal([]byte(payment.Metadata), &metadataMap)
	}
	if _, ok := metadataMap["description"]; !ok {
		metadataMap["description"] = payment.Description
	}
	encoded, err := json.Marshal(metadataMap)
	if err != nil {
		return
	}
	payment.Metadata = string(encoded)
	payment.Description = ""
}

func liftDescriptionFromMetadata(payment *Payment) {
	if payment.Metadata == "" {
		return
	}
	var metadataMap map[string]any
	if err := json.Unmarshal([]byte(payment.Metadata), &metadataMap); err != nil {
		return
	}
	if description, ok := metadataMap["description"].(string); ok {
		payment.Description = description
	}
}
