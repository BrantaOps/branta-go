package branta

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
)

type mockClient struct {
	getFn    func(destinationValue string) ([]Payment, error)
	postFn   func(payment Payment) (*Payment, error)
	validFn  func() (bool, error)
	getCalls []string
	posted   []Payment
}

func (m *mockClient) GetPayments(_ context.Context, destinationValue string, _ *BrantaApiOptions) ([]Payment, error) {
	m.getCalls = append(m.getCalls, destinationValue)
	if m.getFn == nil {
		return nil, nil
	}
	return m.getFn(destinationValue)
}

func (m *mockClient) PostPayment(_ context.Context, payment Payment, _ *BrantaApiOptions) (*Payment, error) {
	m.posted = append(m.posted, payment)
	if m.postFn == nil {
		p := payment
		return &p, nil
	}
	return m.postFn(payment)
}

func (m *mockClient) IsAPIKeyValid(_ context.Context, _ *BrantaApiOptions) (bool, error) {
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
		return "enc:" + value, nil
	}
	return m.encryptFn(value, secret, det)
}

func (m *mockAES) Decrypt(value, secret string) (string, error) {
	if m.decryptFn == nil {
		return "", errors.New("no decrypt")
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

func svc(client *mockClient, aes *mockAES, secrets *mockSecrets) *BrantaService {
	if client == nil {
		client = &mockClient{}
	}
	if aes == nil {
		aes = &mockAES{}
	}
	if secrets == nil {
		secrets = &mockSecrets{values: []string{"bitcoin-secret"}}
	}
	return NewBrantaServiceWithDeps(NewBrantaApiOptions(Localhost), client, aes, secrets)
}

func encDest(value, id string, destType DestinationType) Destination {
	return Destination{Value: value, IsEncrypted: true, Type: destType, ID: id}
}

const addr = "bc1q6745z6cy3u0k9nprurh3x804c4r7u3u8vxca2n"
const invoice = "lnbc10u1p4w23tzpp5example"

func TestBareAddressIsUnknownWithoutLookup(t *testing.T) {
	client := &mockClient{}
	result, err := svc(client, nil, nil).Verify(context.Background(), "bitcoin:"+addr, nil)
	if err != nil || result.Status != VerifyUnknown || len(client.getCalls) != 0 {
		t.Fatalf("%+v %v calls=%v", result, err, client.getCalls)
	}
	if !strings.Contains(result.VerifyURL, "/v2/verify/"+addr) || strings.Contains(result.VerifyURL, "#") {
		t.Fatal(result.VerifyURL)
	}
}

func TestTestnetAddressIsUnknownWithoutLookup(t *testing.T) {
	client := &mockClient{}
	tb1 := "tb1qshra2mlujc9wfscvn4d8aqpyqgxzl55853xg7s"
	result, err := svc(client, nil, nil).Verify(context.Background(), tb1, nil)
	if err != nil || result.Status != VerifyUnknown || len(client.getCalls) != 0 {
		t.Fatalf("%+v %v", result, err)
	}
}

func TestBolt11MatchDecryptsToTheScannedInvoice(t *testing.T) {
	client := &mockClient{getFn: func(string) ([]Payment, error) {
		return []Payment{{Destinations: []Destination{encDest("cipher", "zk-1", Bolt11)}}}, nil
	}}
	aes := &mockAES{decryptFn: func(value, secret string) (string, error) {
		if value == "cipher" && secret == ToNormalizedHash(invoice) {
			return invoice, nil
		}
		return "", errors.New("miss")
	}}
	result, err := svc(client, aes, nil).Verify(context.Background(), "lightning:"+invoice, nil)
	if err != nil || result.Status != VerifyMatch {
		t.Fatalf("%+v %v", result, err)
	}
	if result.Payments[0].Destinations[0].Value != invoice {
		t.Fatal(result.Payments[0].Destinations[0].Value)
	}
	if !strings.Contains(result.VerifyURL, "#k-zk-1=") {
		t.Fatal(result.VerifyURL)
	}
	if client.getCalls[0] != "enc:"+strings.ToLower(invoice) {
		t.Fatal(client.getCalls)
	}
}

func TestBolt11DifferentPlaintextIsTampered(t *testing.T) {
	client := &mockClient{getFn: func(string) ([]Payment, error) {
		return []Payment{{Destinations: []Destination{encDest("cipher", "zk-1", Bolt11)}}}, nil
	}}
	aes := &mockAES{decryptFn: func(string, string) (string, error) {
		return "lnbc1other", nil
	}}
	result, err := svc(client, aes, nil).Verify(context.Background(), invoice, nil)
	if err != nil || result.Status != VerifyTampered || result.Payments != nil {
		t.Fatalf("%+v %v", result, err)
	}
	if result.VerifyURL != "http://localhost:3000/v2/verify/"+invoice {
		t.Fatal(result.VerifyURL)
	}
}

func TestBolt11MatchIsCaseInsensitive(t *testing.T) {
	upper := strings.ToUpper(invoice)
	client := &mockClient{getFn: func(string) ([]Payment, error) {
		return []Payment{{Destinations: []Destination{encDest("cipher", "zk-1", Bolt11)}}}, nil
	}}
	aes := &mockAES{decryptFn: func(string, string) (string, error) {
		return invoice, nil
	}}
	result, err := svc(client, aes, nil).Verify(context.Background(), upper, nil)
	if err != nil || result.Status != VerifyMatch {
		t.Fatalf("%+v %v", result, err)
	}
	if result.Payments[0].Destinations[0].Value != invoice {
		t.Fatal(result.Payments[0].Destinations[0].Value)
	}
}

func TestOnChainMatch(t *testing.T) {
	client := &mockClient{getFn: func(lookup string) ([]Payment, error) {
		if lookup != "cipher-id" {
			t.Fatal(lookup)
		}
		return []Payment{{Destinations: []Destination{encDest("cipher", "zk-1", BitcoinAddress)}}}, nil
	}}
	aes := &mockAES{decryptFn: func(string, string) (string, error) {
		return addr, nil
	}}
	qr := "bitcoin:" + addr + "?branta_id=cipher-id&branta_secret=the-secret"
	result, err := svc(client, aes, nil).Verify(context.Background(), qr, nil)
	if err != nil || result.Status != VerifyMatch {
		t.Fatalf("%+v %v", result, err)
	}
	if result.Payments[0].Destinations[0].Value != addr {
		t.Fatal(result.Payments[0].Destinations[0].Value)
	}
	if !strings.HasSuffix(result.VerifyURL, "/v2/verify/cipher-id#k-zk-1=the-secret") {
		t.Fatal(result.VerifyURL)
	}
}

func TestSwappedAddressIsTampered(t *testing.T) {
	other := "bc1qw508d6qejxtdg4y5r3zarvary0c5xw7kv8f3t4"
	client := &mockClient{getFn: func(string) ([]Payment, error) {
		return []Payment{{Destinations: []Destination{encDest("cipher", "zk-1", BitcoinAddress)}}}, nil
	}}
	aes := &mockAES{decryptFn: func(string, string) (string, error) {
		return addr, nil
	}}
	qr := "bitcoin:" + other + "?branta_id=cipher-id&branta_secret=the-secret"
	result, err := svc(client, aes, nil).Verify(context.Background(), qr, nil)
	if err != nil || result.Status != VerifyTampered || result.Payments != nil {
		t.Fatalf("%+v %v", result, err)
	}
	if strings.Contains(result.VerifyURL, "#") {
		t.Fatal(result.VerifyURL)
	}
}

func TestBrantaParamsWithoutPayableDestinationAreUnknown(t *testing.T) {
	client := &mockClient{}
	qr := "bitcoin:?branta_id=cipher-id&branta_secret=the-secret"
	result, err := svc(client, nil, nil).Verify(context.Background(), qr, nil)
	if err != nil || result.Status != VerifyUnknown || len(client.getCalls) != 0 {
		t.Fatalf("%+v %v calls=%v", result, err, client.getCalls)
	}
}

func TestUnverifiableInstructionIsUnknownWithoutLookup(t *testing.T) {
	client := &mockClient{}
	qr := "bitcoin:" + addr + "?lno=lno1qqexample"
	result, err := svc(client, nil, nil).Verify(context.Background(), qr, nil)
	if err != nil || result.Status != VerifyUnknown || len(client.getCalls) != 0 {
		t.Fatalf("%+v %v", result, err)
	}
}

func TestServerErrorIsIncomplete(t *testing.T) {
	client := &mockClient{getFn: func(string) ([]Payment, error) {
		return nil, errIncomplete
	}}
	result, err := svc(client, nil, nil).Verify(context.Background(), invoice, nil)
	if err != nil || result.Status != VerifyIncomplete {
		t.Fatalf("%+v %v", result, err)
	}
	if result.VerifyURL != "http://localhost:3000/v2/verify/"+invoice {
		t.Fatal(result.VerifyURL)
	}
}

func TestLogoMismatchStillErrors(t *testing.T) {
	client := &mockClient{getFn: func(string) ([]Payment, error) {
		return nil, &LogoURLDomainMismatchError{Field: "platform_logo_url"}
	}}
	_, err := svc(client, nil, nil).Verify(context.Background(), invoice, nil)
	var mismatch *LogoURLDomainMismatchError
	if !errors.As(err, &mismatch) {
		t.Fatal(err)
	}
}

func TestNotFoundIsUnknown(t *testing.T) {
	client := &mockClient{getFn: func(string) ([]Payment, error) {
		return []Payment{}, nil
	}}
	result, err := svc(client, nil, nil).Verify(context.Background(), invoice, nil)
	if err != nil || result.Status != VerifyUnknown || result.Payments != nil {
		t.Fatalf("%+v %v", result, err)
	}
}

func TestAddPaymentLeavesCallerPlaintext(t *testing.T) {
	client := &mockClient{}
	payment := NewPaymentBuilder().
		AddDestination(addr, BitcoinAddress).
		SetDescription("Invoice").
		Build()
	payment.Destinations[0].ID = "zk-1"
	result, err := svc(client, nil, &mockSecrets{values: []string{"dek", "bitcoin-secret"}}).AddPayment(context.Background(), payment, nil)
	if err != nil {
		t.Fatal(err)
	}
	if payment.Destinations[0].Value != addr || payment.Description != "Invoice" || payment.Metadata != "" {
		t.Fatalf("caller changed: %+v", payment)
	}
	posted := client.posted[0]
	if posted.Destinations[0].Value != "enc:"+addr || posted.Description != "" {
		t.Fatalf("posted %+v", posted)
	}
	var meta map[string]string
	if err := json.Unmarshal([]byte(posted.Metadata), &meta); err == nil {
		t.Fatal("metadata should be ciphertext, got", posted.Metadata)
	}
	if result.Secret != "bitcoin-secret" {
		t.Fatal(result.Secret)
	}
}

func TestAddPaymentRetryEncryptsTheOriginalAddress(t *testing.T) {
	calls := 0
	client := &mockClient{postFn: func(payment Payment) (*Payment, error) {
		calls++
		if calls == 1 {
			return nil, &RequestFailedError{Status: 500}
		}
		return &payment, nil
	}}
	aes := &mockAES{encryptFn: func(value, secret string, det bool) (string, error) {
		return "enc-" + secret + ":" + value, nil
	}}
	payment := Payment{Destinations: []Destination{{Value: addr, Type: BitcoinAddress, ID: "zk-1"}}}
	service := svc(client, aes, &mockSecrets{values: []string{"first", "retry"}})
	if _, err := service.AddPayment(context.Background(), payment, nil); err == nil {
		t.Fatal("expected post failure")
	}
	if payment.Destinations[0].Value != addr {
		t.Fatal(payment.Destinations[0].Value)
	}
	result, err := service.AddPayment(context.Background(), payment, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(result.Payment.Destinations[0].Value, "retry:"+addr) {
		t.Fatal(result.Payment.Destinations[0].Value)
	}
	if strings.Contains(client.posted[1].Destinations[0].Value, "enc-first") {
		t.Fatal("retried the ciphertext")
	}
}

func TestAddPaymentBolt11UsesDerivedKey(t *testing.T) {
	client := &mockClient{}
	aes := &mockAES{encryptFn: func(value, secret string, det bool) (string, error) {
		if !det || secret != ToNormalizedHash(invoice) || value != invoice {
			t.Fatalf("encrypt %q %q det=%v", value, secret, det)
		}
		return "cipher", nil
	}}
	payment := Payment{Destinations: []Destination{{Value: strings.ToUpper(invoice), Type: Bolt11, ID: "zk-1"}}}
	_, err := svc(client, aes, nil).AddPayment(context.Background(), payment, nil)
	if err != nil {
		t.Fatal(err)
	}
	if payment.Destinations[0].Value != strings.ToUpper(invoice) {
		t.Fatal("caller mutated")
	}
	if client.posted[0].Destinations[0].Value != "cipher" {
		t.Fatal(client.posted[0].Destinations[0].Value)
	}
}

func TestAddPaymentUnsupportedTypeLeavesCallerUnchanged(t *testing.T) {
	payment := Payment{Destinations: []Destination{{Value: "user@example.com", ID: "zk-1"}}}
	_, err := svc(nil, nil, nil).AddPayment(context.Background(), payment, nil)
	var unsupported *UnsupportedDestinationTypeError
	if !errors.As(err, &unsupported) {
		t.Fatal(err)
	}
	if payment.Destinations[0].Value != "user@example.com" {
		t.Fatal(payment.Destinations[0].Value)
	}
}

func TestAddPaymentEmptyResponseErrors(t *testing.T) {
	client := &mockClient{postFn: func(Payment) (*Payment, error) { return nil, nil }}
	payment := Payment{Destinations: []Destination{{Value: addr, Type: BitcoinAddress}}}
	_, err := svc(client, nil, nil).AddPayment(context.Background(), payment, nil)
	if !errors.Is(err, ErrNoPaymentReturned) {
		t.Fatal(err)
	}
	if payment.Destinations[0].Value != addr {
		t.Fatal("caller mutated")
	}
}

func TestFragmentEncodesReservedCharacters(t *testing.T) {
	client := &mockClient{getFn: func(string) ([]Payment, error) {
		return []Payment{{Destinations: []Destination{encDest("cipher", "a b", Bolt11)}}}, nil
	}}
	aes := &mockAES{
		encryptFn: func(value, secret string, det bool) (string, error) { return "lookup", nil },
		decryptFn: func(string, string) (string, error) { return invoice, nil },
	}
	result, err := svc(client, aes, nil).Verify(context.Background(), invoice, nil)
	if err != nil || result.Status != VerifyMatch {
		t.Fatalf("%+v %v", result, err)
	}
	if !strings.Contains(result.VerifyURL, "#k-a%20b=") {
		t.Fatal(result.VerifyURL)
	}
}

func TestMetadataDecryptLiftsDescription(t *testing.T) {
	meta := `{"description":"Invoice"}`
	client := &mockClient{getFn: func(string) ([]Payment, error) {
		return []Payment{{
			Metadata: "enc-meta",
			Destinations: []Destination{{
				Value: "cipher", IsEncrypted: true, Type: BitcoinAddress, ID: "zk-1", EncryptedDEK: "enc-dek",
			}},
		}}, nil
	}}
	aes := &mockAES{decryptFn: func(value, secret string) (string, error) {
		switch value {
		case "cipher":
			return addr, nil
		case "enc-dek":
			return "dek", nil
		case "enc-meta":
			return meta, nil
		}
		return "", errors.New(value)
	}}
	qr := "bitcoin:" + addr + "?branta_id=cipher-id&branta_secret=the-secret"
	result, err := svc(client, aes, nil).Verify(context.Background(), qr, nil)
	if err != nil || result.Status != VerifyMatch {
		t.Fatalf("%+v %v", result, err)
	}
	if result.Payments[0].Description != "Invoice" || result.Payments[0].Metadata != meta {
		t.Fatalf("%+v", result.Payments[0])
	}
}

func TestConflictingOnChainAddressesAreTamperedBeforeLookup(t *testing.T) {
	client := &mockClient{}
	other := "1A1zP1eP5QGefi2DMPTfTL5SLmv7DivfNa"
	qr := "bitcoin:" + addr + "?bc=" + other + "&branta_id=cipher-id&branta_secret=the-secret"
	result, err := svc(client, nil, nil).Verify(context.Background(), qr, nil)
	if err != nil || result.Status != VerifyTampered || len(client.getCalls) != 0 {
		t.Fatalf("%+v %v calls=%v", result, err, client.getCalls)
	}
}

func TestIsAPIKeyValidPassesThrough(t *testing.T) {
	client := &mockClient{validFn: func() (bool, error) { return true, nil }}
	ok, err := svc(client, nil, nil).IsAPIKeyValid(context.Background(), nil)
	if err != nil || !ok {
		t.Fatal(err)
	}
}
