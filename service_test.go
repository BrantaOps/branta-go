package branta

import (
	"context"
	"errors"
	"strings"
	"testing"
)

type mockClient struct {
	getFn     func(destinationValue string) ([]Payment, error)
	postFn    func(payment Payment) (*Payment, error)
	validFn   func() (bool, error)
	getCalls  []string
	postCalls int
}

func (m *mockClient) GetPayments(_ context.Context, destinationValue string, _ *BrantaClientOptions) ([]Payment, error) {
	m.getCalls = append(m.getCalls, destinationValue)
	if m.getFn == nil {
		return nil, nil
	}
	return m.getFn(destinationValue)
}

func (m *mockClient) PostPayment(_ context.Context, payment Payment, _ *BrantaClientOptions) (*Payment, error) {
	m.postCalls++
	if m.postFn == nil {
		p := payment
		return &p, nil
	}
	return m.postFn(payment)
}

func (m *mockClient) IsAPIKeyValid(_ context.Context, _ *BrantaClientOptions) (bool, error) {
	if m.validFn == nil {
		return false, nil
	}
	return m.validFn()
}

type mockAES struct {
	encryptFn func(value, secret string, det bool) (string, error)
	decryptFn func(value, secret string) (string, error)
}

func (m *mockAES) Encrypt(value, secret string, det bool) (string, error) {
	if m.encryptFn == nil {
		panic("unexpected Encrypt(" + value + ")")
	}
	return m.encryptFn(value, secret, det)
}

func (m *mockAES) Decrypt(value, secret string) (string, error) {
	if m.decryptFn == nil {
		panic("unexpected Decrypt(" + value + ")")
	}
	return m.decryptFn(value, secret)
}

type mockSecrets struct {
	values []string
	i      int
	det    bool
}

func (m *mockSecrets) Generate() string {
	if m.i >= len(m.values) {
		return "secret"
	}
	v := m.values[m.i]
	m.i++
	return v
}

func (m *mockSecrets) DeterministicNonce() bool { return m.det }

func strictOptions() BrantaClientOptions { return NewBrantaClientOptions(Staging) }

func looseOptions() BrantaClientOptions {
	opts := NewBrantaClientOptions(Staging)
	opts.Privacy = PrivacyLoose
	return opts
}

func svc(opts BrantaClientOptions, client *mockClient, aes *mockAES, secrets *mockSecrets) *BrantaService {
	if client == nil {
		client = &mockClient{}
	}
	if aes == nil {
		aes = &mockAES{}
	}
	if secrets == nil {
		secrets = &mockSecrets{}
	}
	return NewBrantaServiceWithDeps(opts, client, aes, secrets)
}

func zkDest(value, zkID string, destType DestinationType) Destination {
	return Destination{Value: value, IsZk: true, Type: destType, ZkID: zkID}
}

func TestStrictModePlainBitcoinLookupWithoutKeyErrors(t *testing.T) {
	_, err := svc(strictOptions(), &mockClient{}, &mockAES{}, &mockSecrets{}).
		GetPayments(context.Background(), "1A1zP1eP5QGefi2DMPTfTL5SLmv7DivfNa", "", nil)
	if !errors.Is(err, ErrPrivacyModeViolation) {
		t.Fatalf("got %v", err)
	}
}

func TestStrictModePlainBitcoinLookupWithKeySucceeds(t *testing.T) {
	client := &mockClient{getFn: func(string) ([]Payment, error) { return nil, nil }}
	result, err := svc(strictOptions(), client, &mockAES{}, &mockSecrets{}).
		GetPayments(context.Background(), "1A1zP1eP5QGefi2DMPTfTL5SLmv7DivfNa", "some-key", nil)
	if err != nil || len(result.Payments) != 0 {
		t.Fatalf("%v %v", result, err)
	}
}

func TestStrictModeHashZkLookupDoesNotErrorAndUsesEncryptedLookup(t *testing.T) {
	client := &mockClient{getFn: func(value string) ([]Payment, error) {
		if value != "ENCRYPTED-LOOKUP" {
			t.Errorf("lookup %s", value)
		}
		return nil, nil
	}}
	aes := &mockAES{encryptFn: func(value, secret string, det bool) (string, error) {
		if value != "lnbc1qsomething" || !det {
			t.Errorf("encrypt %s det=%v", value, det)
		}
		return "ENCRYPTED-LOOKUP", nil
	}}
	result, err := svc(strictOptions(), client, aes, &mockSecrets{}).
		GetPayments(context.Background(), "lnbc1qsomething", "", nil)
	if err != nil || len(result.Payments) != 0 {
		t.Fatalf("%v %v", result, err)
	}
}

func TestStrictModeHashZkNotFoundNeverFallsBackToPlain(t *testing.T) {
	client := &mockClient{getFn: func(string) ([]Payment, error) { return nil, nil }}
	aes := &mockAES{encryptFn: func(string, string, bool) (string, error) { return "ENCRYPTED-LOOKUP", nil }}
	_, _ = svc(strictOptions(), client, aes, &mockSecrets{}).
		GetPayments(context.Background(), "lnbc1qsomething", "", nil)
	if len(client.getCalls) != 1 {
		t.Fatalf("calls %d", len(client.getCalls))
	}
}

func TestLooseModeHashZkNotFoundFallsBackToPlainLookup(t *testing.T) {
	client := &mockClient{getFn: func(value string) ([]Payment, error) {
		if value == "lnbc1qsomething" {
			return []Payment{{}}, nil
		}
		return nil, nil
	}}
	aes := &mockAES{encryptFn: func(string, string, bool) (string, error) { return "ENCRYPTED-LOOKUP", nil }}
	result, err := svc(looseOptions(), client, aes, &mockSecrets{}).
		GetPayments(context.Background(), "lnbc1qsomething", "", nil)
	if err != nil || len(result.Payments) != 1 {
		t.Fatalf("%v %v", result, err)
	}
	if len(client.getCalls) != 2 {
		t.Fatalf("calls %d", len(client.getCalls))
	}
}

func TestZkBitcoinDestinationDecryptsWithCorrectKey(t *testing.T) {
	client := &mockClient{getFn: func(string) ([]Payment, error) {
		return []Payment{{Destinations: []Destination{zkDest("CIPHERTEXT", "zk-1", BitcoinAddress)}}}, nil
	}}
	aes := &mockAES{decryptFn: func(value, key string) (string, error) {
		if value != "CIPHERTEXT" || key != "correct-key" {
			t.Errorf("decrypt %s %s", value, key)
		}
		return "1A1zP1eP5QGefi2DMPTfTL5SLmv7DivfNa", nil
	}}
	result, err := svc(looseOptions(), client, aes, &mockSecrets{}).
		GetPayments(context.Background(), "CIPHERTEXT", "correct-key", nil)
	if err != nil {
		t.Fatal(err)
	}
	dest := result.Payments[0].Destinations[0]
	if dest.Value != "1A1zP1eP5QGefi2DMPTfTL5SLmv7DivfNa" || dest.IsEncrypted {
		t.Fatalf("%+v", dest)
	}
}

func TestZkBitcoinDestinationWrongKeyStaysEncryptedNoError(t *testing.T) {
	client := &mockClient{getFn: func(string) ([]Payment, error) {
		return []Payment{{Destinations: []Destination{zkDest("CIPHERTEXT", "zk-1", BitcoinAddress)}}}, nil
	}}
	aes := &mockAES{decryptFn: func(string, string) (string, error) {
		return "", &DecryptionError{Err: errors.New("bad key")}
	}}
	result, err := svc(looseOptions(), client, aes, &mockSecrets{}).
		GetPayments(context.Background(), "CIPHERTEXT", "wrong-key", nil)
	if err != nil {
		t.Fatal(err)
	}
	dest := result.Payments[0].Destinations[0]
	if dest.Value != "CIPHERTEXT" || !dest.IsEncrypted {
		t.Fatalf("%+v", dest)
	}
}

func TestZkBitcoinDestinationNoKeySuppliedStaysEncrypted(t *testing.T) {
	client := &mockClient{getFn: func(string) ([]Payment, error) {
		return []Payment{{Destinations: []Destination{zkDest("CIPHERTEXT", "zk-1", BitcoinAddress)}}}, nil
	}}
	result, err := svc(looseOptions(), client, &mockAES{}, &mockSecrets{}).
		GetPayments(context.Background(), "CIPHERTEXT", "", nil)
	if err != nil {
		t.Fatal(err)
	}
	if !result.Payments[0].Destinations[0].IsEncrypted {
		t.Fatal("should stay encrypted")
	}
}

func TestNonZkDestinationIsLeftCompletelyUntouched(t *testing.T) {
	client := &mockClient{getFn: func(string) ([]Payment, error) {
		return []Payment{{Destinations: []Destination{NewDestination("plain-value", BitcoinAddress)}}}, nil
	}}
	result, err := svc(looseOptions(), client, &mockAES{}, &mockSecrets{}).
		GetPayments(context.Background(), "plain-value", "", nil)
	if err != nil {
		t.Fatal(err)
	}
	if result.Payments[0].Destinations[0].Value != "plain-value" || result.Payments[0].Destinations[0].IsEncrypted {
		t.Fatal("untouched")
	}
}

func TestHashZkDestinationDecryptsByHashDerivedKey(t *testing.T) {
	client := &mockClient{getFn: func(string) ([]Payment, error) {
		return []Payment{{Destinations: []Destination{zkDest("CIPHERTEXT", "zk-1", Bolt11)}}}, nil
	}}
	aes := &mockAES{
		encryptFn: func(string, string, bool) (string, error) { return "LOOKUP", nil },
		decryptFn: func(value, key string) (string, error) {
			if value != "CIPHERTEXT" || key != ToNormalizedHash("lnbc1qsomething") {
				t.Errorf("decrypt %s key=%s", value, key)
			}
			return "lnbc1qsomething", nil
		},
	}
	result, err := svc(looseOptions(), client, aes, &mockSecrets{}).
		GetPayments(context.Background(), "lnbc1qsomething", "", nil)
	if err != nil {
		t.Fatal(err)
	}
	if result.Payments[0].Destinations[0].Value != "lnbc1qsomething" || result.Payments[0].Destinations[0].IsEncrypted {
		t.Fatal("hash zk")
	}
}

func TestVerifyURLIncludesFragmentWhenAKeyResolved(t *testing.T) {
	client := &mockClient{getFn: func(string) ([]Payment, error) {
		return []Payment{{Destinations: []Destination{zkDest("CIPHERTEXT", "zk-1", BitcoinAddress)}}}, nil
	}}
	aes := &mockAES{decryptFn: func(string, string) (string, error) { return "1A1zP...", nil }}
	result, err := svc(looseOptions(), client, aes, &mockSecrets{}).
		GetPayments(context.Background(), "CIPHERTEXT", "key", nil)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(result.VerifyURL, "#k-zk-1=key") {
		t.Fatal(result.VerifyURL)
	}
}

func TestVerifyURLHasNoFragmentWhenNoKeyResolved(t *testing.T) {
	client := &mockClient{getFn: func(string) ([]Payment, error) { return nil, nil }}
	result, err := svc(looseOptions(), client, &mockAES{}, &mockSecrets{}).
		GetPayments(context.Background(), "plain-value", "", nil)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(result.VerifyURL, "#") {
		t.Fatal(result.VerifyURL)
	}
	if !strings.Contains(result.VerifyURL, "/v2/verify/plain-value") {
		t.Fatal(result.VerifyURL)
	}
}

func TestQRStrictPlainBitcoinShortCircuitsWithNoNetworkCall(t *testing.T) {
	client := &mockClient{getFn: func(string) ([]Payment, error) {
		t.Fatal("should not hit network")
		return nil, nil
	}}
	result, err := svc(strictOptions(), client, &mockAES{}, &mockSecrets{}).
		GetPaymentsByQRCode(context.Background(), "bitcoin:1A1zP1eP5QGefi2DMPTfTL5SLmv7DivfNa", nil)
	if err != nil || len(result.Payments) != 0 || result.VerifyURL == "" {
		t.Fatalf("%+v %v", result, err)
	}
}

func TestQRStrictHashZkDestinationSucceedsOverNetwork(t *testing.T) {
	client := &mockClient{getFn: func(string) ([]Payment, error) { return nil, nil }}
	aes := &mockAES{encryptFn: func(string, string, bool) (string, error) { return "LOOKUP", nil }}
	result, err := svc(strictOptions(), client, aes, &mockSecrets{}).
		GetPaymentsByQRCode(context.Background(), "lnbc1qsomething", nil)
	if err != nil || len(result.Payments) != 0 {
		t.Fatalf("%v %v", result, err)
	}
}

func TestQROnChainZkDecryptsBitcoinWithSecret(t *testing.T) {
	client := &mockClient{getFn: func(value string) ([]Payment, error) {
		if value != "onchain-id" {
			t.Errorf("lookup %s", value)
		}
		return []Payment{{Destinations: []Destination{zkDest("CIPHERTEXT", "zk-1", BitcoinAddress)}}}, nil
	}}
	aes := &mockAES{decryptFn: func(value, key string) (string, error) {
		if value != "CIPHERTEXT" || key != "onchain-secret" {
			t.Errorf("decrypt %s %s", value, key)
		}
		return "1A1zP...", nil
	}}
	qr := "bitcoin:1A1zP...?branta_id=onchain-id&branta_secret=onchain-secret"
	result, err := svc(looseOptions(), client, aes, &mockSecrets{}).
		GetPaymentsByQRCode(context.Background(), qr, nil)
	if err != nil {
		t.Fatal(err)
	}
	if result.Payments[0].Destinations[0].Value != "1A1zP..." {
		t.Fatal(result.Payments[0].Destinations[0].Value)
	}
}

func TestQRCombinedOnChainZkAlsoDecryptsHashZkDestination(t *testing.T) {
	client := &mockClient{getFn: func(string) ([]Payment, error) {
		return []Payment{{Destinations: []Destination{
			zkDest("BTC-CIPHERTEXT", "zk-1", BitcoinAddress),
			zkDest("LN-CIPHERTEXT", "zk-2", Bolt11),
		}}}, nil
	}}
	aes := &mockAES{decryptFn: func(value, key string) (string, error) {
		switch {
		case value == "BTC-CIPHERTEXT" && key == "onchain-secret":
			return "1A1zP...", nil
		case value == "LN-CIPHERTEXT" && key == ToNormalizedHash("lnbc1qsomething"):
			return "lnbc1qsomething", nil
		default:
			return "", &DecryptionError{Err: errors.New("unexpected")}
		}
	}}
	qr := "bitcoin:1A1zP...?branta_id=onchain-id&branta_secret=onchain-secret&lightning=lnbc1qsomething"
	result, err := svc(looseOptions(), client, aes, &mockSecrets{}).
		GetPaymentsByQRCode(context.Background(), qr, nil)
	if err != nil {
		t.Fatal(err)
	}
	if result.Payments[0].Destinations[0].Value != "1A1zP..." {
		t.Fatal("btc")
	}
	if result.Payments[0].Destinations[1].Value != "lnbc1qsomething" {
		t.Fatal("ln")
	}
}

func TestQROnChainZkLeavesUnrelatedDestinationEncrypted(t *testing.T) {
	client := &mockClient{getFn: func(string) ([]Payment, error) {
		return []Payment{{Destinations: []Destination{
			zkDest("BTC-CIPHERTEXT", "zk-1", BitcoinAddress),
			zkDest("ARK-CIPHERTEXT", "zk-2", ArkAddress),
		}}}, nil
	}}
	aes := &mockAES{decryptFn: func(value, key string) (string, error) {
		if value == "BTC-CIPHERTEXT" {
			return "1A1zP...", nil
		}
		return "", &DecryptionError{Err: errors.New("no")}
	}}
	qr := "bitcoin:1A1zP...?branta_id=onchain-id&branta_secret=onchain-secret"
	result, err := svc(looseOptions(), client, aes, &mockSecrets{}).
		GetPaymentsByQRCode(context.Background(), qr, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !result.Payments[0].Destinations[1].IsEncrypted || result.Payments[0].Destinations[1].Value != "ARK-CIPHERTEXT" {
		t.Fatal("ark should stay encrypted")
	}
}

func TestAddPaymentStrictModeAllNonZkErrors(t *testing.T) {
	payment := Payment{Destinations: []Destination{NewDestination("addr", BitcoinAddress)}}
	_, err := svc(strictOptions(), &mockClient{}, &mockAES{}, &mockSecrets{}).
		AddPayment(context.Background(), payment, nil)
	if !errors.Is(err, ErrNonZkDestinationInStrictMode) {
		t.Fatalf("got %v", err)
	}
}

func TestAddPaymentStrictModeMixedZkAndNonZkErrors(t *testing.T) {
	payment := Payment{Destinations: []Destination{
		zkDest("addr1", "zk-1", BitcoinAddress),
		NewDestination("addr2", BitcoinAddress),
	}}
	_, err := svc(strictOptions(), &mockClient{}, &mockAES{}, &mockSecrets{}).
		AddPayment(context.Background(), payment, nil)
	if !errors.Is(err, ErrNonZkDestinationInStrictMode) {
		t.Fatalf("got %v", err)
	}
}

func TestAddPaymentPlainDestinationUntouchedInLooseMode(t *testing.T) {
	client := &mockClient{postFn: func(p Payment) (*Payment, error) { cp := p; return &cp, nil }}
	payment := Payment{Destinations: []Destination{NewDestination("addr", BitcoinAddress)}}
	result, err := svc(looseOptions(), client, &mockAES{}, &mockSecrets{values: []string{"generated-secret"}}).
		AddPayment(context.Background(), payment, nil)
	if err != nil {
		t.Fatal(err)
	}
	if result.Payment.Destinations[0].Value != "addr" {
		t.Fatal(result.Payment.Destinations[0].Value)
	}
}

func TestAddPaymentZkBitcoinEncryptsWithSharedSecret(t *testing.T) {
	client := &mockClient{postFn: func(p Payment) (*Payment, error) { cp := p; return &cp, nil }}
	aes := &mockAES{encryptFn: func(value, secret string, det bool) (string, error) {
		if value != "addr" || secret != "the-secret" {
			t.Errorf("encrypt %s %s", value, secret)
		}
		return "CIPHERTEXT", nil
	}}
	payment := Payment{Destinations: []Destination{zkDest("addr", "zk-1", BitcoinAddress)}}
	result, err := svc(looseOptions(), client, aes, &mockSecrets{values: []string{"the-secret"}}).
		AddPayment(context.Background(), payment, nil)
	if err != nil {
		t.Fatal(err)
	}
	if result.Secret != "the-secret" || result.Payment.Destinations[0].Value != "CIPHERTEXT" {
		t.Fatalf("%+v", result)
	}
}

func TestAddPaymentZkHashTypeEncryptsWithHashDerivedKey(t *testing.T) {
	client := &mockClient{postFn: func(p Payment) (*Payment, error) { cp := p; return &cp, nil }}
	aes := &mockAES{encryptFn: func(value, key string, det bool) (string, error) {
		if value != "lnbc1qsomething" || key != ToNormalizedHash("lnbc1qsomething") || !det {
			t.Errorf("encrypt %s key=%s det=%v", value, key, det)
		}
		return "CIPHERTEXT", nil
	}}
	payment := Payment{Destinations: []Destination{zkDest("lnbc1qsomething", "zk-1", Bolt11)}}
	result, err := svc(looseOptions(), client, aes, &mockSecrets{values: []string{"unused-secret"}}).
		AddPayment(context.Background(), payment, nil)
	if err != nil {
		t.Fatal(err)
	}
	if result.Payment.Destinations[0].Value != "CIPHERTEXT" {
		t.Fatal(result.Payment.Destinations[0].Value)
	}
}

func TestAddPaymentUnsupportedZkTypeErrors(t *testing.T) {
	payment := Payment{Destinations: []Destination{zkDest("not-a-known-format", "zk-1", LnAddress)}}
	_, err := svc(looseOptions(), &mockClient{}, &mockAES{}, &mockSecrets{values: []string{"secret"}}).
		AddPayment(context.Background(), payment, nil)
	var unsup *UnsupportedZkDestinationTypeError
	if !errors.As(err, &unsup) {
		t.Fatalf("got %v", err)
	}
}

func TestAddPaymentNoPaymentReturnedErrors(t *testing.T) {
	client := &mockClient{postFn: func(Payment) (*Payment, error) { return nil, nil }}
	payment := Payment{Destinations: []Destination{NewDestination("addr", BitcoinAddress)}}
	_, err := svc(looseOptions(), client, &mockAES{}, &mockSecrets{values: []string{"secret"}}).
		AddPayment(context.Background(), payment, nil)
	if !errors.Is(err, ErrNoPaymentReturned) {
		t.Fatalf("got %v", err)
	}
}

func TestAddPaymentWithMetadataAndZkDestinationSetsEncryptedDEK(t *testing.T) {
	client := &mockClient{postFn: func(p Payment) (*Payment, error) { cp := p; return &cp, nil }}
	aes := &mockAES{encryptFn: func(value, secret string, det bool) (string, error) {
		switch {
		case value == `{"a":"b"}` && secret == "dek-value" && !det:
			return "ENCRYPTED-METADATA", nil
		case value == "addr" && secret == "shared-secret":
			return "CIPHERTEXT", nil
		case value == "dek-value" && secret == "shared-secret" && !det:
			return "ENCRYPTED-DEK", nil
		default:
			t.Errorf("unexpected encrypt %q %q det=%v", value, secret, det)
			return "", &EncryptionError{}
		}
	}}
	payment := Payment{
		Destinations: []Destination{zkDest("addr", "zk-1", BitcoinAddress)},
		Metadata:     `{"a":"b"}`,
	}
	result, err := svc(looseOptions(), client, aes, &mockSecrets{values: []string{"dek-value", "shared-secret"}}).
		AddPayment(context.Background(), payment, nil)
	if err != nil {
		t.Fatal(err)
	}
	if result.Payment.Metadata != "ENCRYPTED-METADATA" {
		t.Fatal(result.Payment.Metadata)
	}
	if result.Payment.Destinations[0].EncryptedDEK != "ENCRYPTED-DEK" {
		t.Fatal(result.Payment.Destinations[0].EncryptedDEK)
	}
}

func TestAddPaymentWithMetadataButNoZkDestinationDoesNotEncryptMetadata(t *testing.T) {
	client := &mockClient{postFn: func(p Payment) (*Payment, error) { cp := p; return &cp, nil }}
	payment := Payment{
		Destinations: []Destination{NewDestination("addr", BitcoinAddress)},
		Metadata:     "plain-metadata",
	}
	result, err := svc(looseOptions(), client, &mockAES{}, &mockSecrets{values: []string{"secret"}}).
		AddPayment(context.Background(), payment, nil)
	if err != nil {
		t.Fatal(err)
	}
	if result.Payment.Metadata != "plain-metadata" || result.Payment.Destinations[0].EncryptedDEK != "" {
		t.Fatal("metadata should stay plain")
	}
}

func TestGetPaymentsDecryptsMetadataWhenEncryptedDEKPresent(t *testing.T) {
	client := &mockClient{getFn: func(string) ([]Payment, error) {
		d := zkDest("CIPHERTEXT", "zk-1", BitcoinAddress)
		d.EncryptedDEK = "ENCRYPTED-DEK"
		return []Payment{{Destinations: []Destination{d}, Metadata: "ENCRYPTED-METADATA"}}, nil
	}}
	aes := &mockAES{decryptFn: func(value, key string) (string, error) {
		switch {
		case value == "CIPHERTEXT" && key == "the-key":
			return "plain-address", nil
		case value == "ENCRYPTED-DEK" && key == "the-key":
			return "dek-value", nil
		case value == "ENCRYPTED-METADATA" && key == "dek-value":
			return "plain-metadata", nil
		default:
			return "", &DecryptionError{Err: errors.New("unexpected")}
		}
	}}
	result, err := svc(looseOptions(), client, aes, &mockSecrets{}).
		GetPayments(context.Background(), "CIPHERTEXT", "the-key", nil)
	if err != nil {
		t.Fatal(err)
	}
	if result.Payments[0].Metadata != "plain-metadata" || !result.Payments[0].IsMetadataDecrypted {
		t.Fatalf("%+v", result.Payments[0])
	}
}

func TestGetPaymentsMetadataDecryptFailureLeavesMetadataUntouched(t *testing.T) {
	client := &mockClient{getFn: func(string) ([]Payment, error) {
		d := zkDest("CIPHERTEXT", "zk-1", BitcoinAddress)
		d.EncryptedDEK = "ENCRYPTED-DEK"
		return []Payment{{Destinations: []Destination{d}, Metadata: "ENCRYPTED-METADATA"}}, nil
	}}
	aes := &mockAES{decryptFn: func(value, key string) (string, error) {
		if value == "CIPHERTEXT" {
			return "plain-address", nil
		}
		if value == "ENCRYPTED-DEK" {
			return "", &DecryptionError{Err: errors.New("bad dek")}
		}
		return "", &DecryptionError{Err: errors.New("unexpected")}
	}}
	result, err := svc(looseOptions(), client, aes, &mockSecrets{}).
		GetPayments(context.Background(), "CIPHERTEXT", "the-key", nil)
	if err != nil {
		t.Fatal(err)
	}
	if result.Payments[0].Metadata != "ENCRYPTED-METADATA" || result.Payments[0].IsMetadataDecrypted {
		t.Fatal("metadata should stay encrypted")
	}
}

func TestGetPaymentsDecryptsMetadataOnlyOncePerPayment(t *testing.T) {
	client := &mockClient{getFn: func(string) ([]Payment, error) {
		d1 := zkDest("CIPHER-1", "zk-1", BitcoinAddress)
		d1.EncryptedDEK = "DEK-1"
		d2 := zkDest("CIPHER-1", "zk-2", BitcoinAddress)
		d2.EncryptedDEK = "DEK-2"
		return []Payment{{Destinations: []Destination{d1, d2}, Metadata: "ENCRYPTED-METADATA"}}, nil
	}}
	dek1Calls, metaCalls := 0, 0
	aes := &mockAES{decryptFn: func(value, key string) (string, error) {
		switch value {
		case "CIPHER-1":
			return "plain", nil
		case "DEK-1":
			dek1Calls++
			return "dek", nil
		case "ENCRYPTED-METADATA":
			metaCalls++
			return "decrypted", nil
		case "DEK-2":
			t.Fatal("DEK-2 should not be decrypted")
		}
		return "", &DecryptionError{Err: errors.New("unexpected " + value)}
	}}
	result, err := svc(looseOptions(), client, aes, &mockSecrets{}).
		GetPayments(context.Background(), "CIPHER-1", "shared-key", nil)
	if err != nil {
		t.Fatal(err)
	}
	if result.Payments[0].Metadata != "decrypted" || dek1Calls != 1 || metaCalls != 1 {
		t.Fatalf("meta=%s dek1=%d metaCalls=%d", result.Payments[0].Metadata, dek1Calls, metaCalls)
	}
}

const swappedAddress = "1BvBMSEYstWetqTFn5Au4m4GFg7xJaNVN2"
const bech32Address = "bc1qw508d6qejxtdg4y5r3zarvary0c5xw7kv8f3t4"

func TestQROnChainZkSwappedAddressRejects(t *testing.T) {
	client := &mockClient{getFn: func(string) ([]Payment, error) {
		return []Payment{{Destinations: []Destination{zkDest("CIPHERTEXT", "zk-1", BitcoinAddress)}}}, nil
	}}
	aes := &mockAES{decryptFn: func(string, string) (string, error) {
		return "1A1zP1eP5QGefi2DMPTfTL5SLmv7DivfNa", nil
	}}
	qr := "bitcoin:" + swappedAddress + "?branta_id=onchain-id&branta_secret=onchain-secret"
	_, err := svc(looseOptions(), client, aes, &mockSecrets{}).
		GetPaymentsByQRCode(context.Background(), qr, nil)
	if !errors.Is(err, ErrTampered) {
		t.Fatalf("got %v", err)
	}
}

func TestQROnChainZkMatchingAddressDoesNotError(t *testing.T) {
	client := &mockClient{getFn: func(string) ([]Payment, error) {
		return []Payment{{Destinations: []Destination{zkDest("CIPHERTEXT", "zk-1", BitcoinAddress)}}}, nil
	}}
	aes := &mockAES{decryptFn: func(string, string) (string, error) {
		return "1A1zP1eP5QGefi2DMPTfTL5SLmv7DivfNa", nil
	}}
	qr := "bitcoin:1A1zP1eP5QGefi2DMPTfTL5SLmv7DivfNa?branta_id=onchain-id&branta_secret=onchain-secret"
	result, err := svc(looseOptions(), client, aes, &mockSecrets{}).
		GetPaymentsByQRCode(context.Background(), qr, nil)
	if err != nil {
		t.Fatal(err)
	}
	if result.Payments[0].Destinations[0].Value != "1A1zP1eP5QGefi2DMPTfTL5SLmv7DivfNa" {
		t.Fatal(result.Payments[0].Destinations[0].Value)
	}
}

func TestQRUppercaseBech32MatchesLowercaseRegisteredAddress(t *testing.T) {
	client := &mockClient{getFn: func(string) ([]Payment, error) {
		return []Payment{{Destinations: []Destination{zkDest("CIPHERTEXT", "zk-1", BitcoinAddress)}}}, nil
	}}
	aes := &mockAES{decryptFn: func(string, string) (string, error) { return bech32Address, nil }}
	qr := "bitcoin:" + strings.ToUpper(bech32Address) + "?branta_id=onchain-id&branta_secret=onchain-secret"
	result, err := svc(looseOptions(), client, aes, &mockSecrets{}).
		GetPaymentsByQRCode(context.Background(), qr, nil)
	if err != nil {
		t.Fatal(err)
	}
	if result.Payments[0].Destinations[0].Value != bech32Address {
		t.Fatal(result.Payments[0].Destinations[0].Value)
	}
}

func TestQRBase58CaseMismatchRejects(t *testing.T) {
	client := &mockClient{getFn: func(string) ([]Payment, error) {
		return []Payment{{Destinations: []Destination{zkDest("CIPHERTEXT", "zk-1", BitcoinAddress)}}}, nil
	}}
	aes := &mockAES{decryptFn: func(string, string) (string, error) {
		return "1A1zP1eP5QGefi2DMPTfTL5SLmv7DivfNa", nil
	}}
	qr := "bitcoin:1a1zp1ep5qgefi2dmptftl5slmv7divfna?branta_id=onchain-id&branta_secret=onchain-secret"
	_, err := svc(looseOptions(), client, aes, &mockSecrets{}).
		GetPaymentsByQRCode(context.Background(), qr, nil)
	if !errors.Is(err, ErrTampered) {
		t.Fatalf("got %v", err)
	}
}

func TestQRLightningWithZkParamsNoPlainAddressDecryptsWithoutComparison(t *testing.T) {
	client := &mockClient{getFn: func(string) ([]Payment, error) {
		return []Payment{{Destinations: []Destination{zkDest("CIPHERTEXT", "zk-1", BitcoinAddress)}}}, nil
	}}
	aes := &mockAES{decryptFn: func(string, string) (string, error) {
		return "1A1zP1eP5QGefi2DMPTfTL5SLmv7DivfNa", nil
	}}
	qr := "lightning:lnbc1qsomething?branta_id=onchain-id&branta_secret=onchain-secret"
	result, err := svc(looseOptions(), client, aes, &mockSecrets{}).
		GetPaymentsByQRCode(context.Background(), qr, nil)
	if err != nil {
		t.Fatal(err)
	}
	if result.Payments[0].Destinations[0].Value != "1A1zP1eP5QGefi2DMPTfTL5SLmv7DivfNa" {
		t.Fatal(result.Payments[0].Destinations[0].Value)
	}
}

func TestQRCombinedZkSwappedAddressRejects(t *testing.T) {
	client := &mockClient{getFn: func(string) ([]Payment, error) {
		return []Payment{{Destinations: []Destination{
			zkDest("BTC-CIPHERTEXT", "zk-1", BitcoinAddress),
			zkDest("LN-CIPHERTEXT", "zk-2", Bolt11),
		}}}, nil
	}}
	aes := &mockAES{decryptFn: func(value, key string) (string, error) {
		if value == "BTC-CIPHERTEXT" {
			return "1A1zP1eP5QGefi2DMPTfTL5SLmv7DivfNa", nil
		}
		return "", &DecryptionError{Err: errors.New("no")}
	}}
	qr := "bitcoin:" + swappedAddress + "?branta_id=onchain-id&branta_secret=onchain-secret&lightning=lnbc1qsomething"
	_, err := svc(looseOptions(), client, aes, &mockSecrets{}).
		GetPaymentsByQRCode(context.Background(), qr, nil)
	if !errors.Is(err, ErrTampered) {
		t.Fatalf("got %v", err)
	}
}

func TestIsAPIKeyValidPassesThrough(t *testing.T) {
	trueClient := &mockClient{validFn: func() (bool, error) { return true, nil }}
	ok, err := svc(looseOptions(), trueClient, &mockAES{}, &mockSecrets{}).IsAPIKeyValid(context.Background(), nil)
	if err != nil || !ok {
		t.Fatal("true")
	}
	falseClient := &mockClient{validFn: func() (bool, error) { return false, nil }}
	ok, err = svc(looseOptions(), falseClient, &mockAES{}, &mockSecrets{}).IsAPIKeyValid(context.Background(), nil)
	if err != nil || ok {
		t.Fatal("false")
	}
}
