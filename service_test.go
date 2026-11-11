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
	if len(result.MatchedDestinations) != 1 || result.MatchedDestinations[0].Value != addr {
		t.Fatalf("%+v", result.MatchedDestinations)
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

const usdtPolygonAddr = "0x278297528c32eCa01295B0d2135984d3442ec963"
const usdcPolygonAddr = "0xAb5801a7D398351b8bE11C439e05C5B3259aeC9B"
const siblingUsdtAddr = "0x2222222222222222222222222222222222222222"
const swappedPolygonAddr = "0x1111111111111111111111111111111111111111"

func polURI(address, encrypted, secret string) string {
	return "pol:" + address + "?value=50&user=send&currency=USD&branta_id=" + encrypted + "&branta_secret=" + secret
}

func TestPolUsdtMatchIgnoresAmountQuery(t *testing.T) {
	client := &mockClient{getFn: func(lookup string) ([]Payment, error) {
		if lookup != "enc-usdt" {
			t.Fatal(lookup)
		}
		return []Payment{{Destinations: []Destination{encDest("cipher-usdt", "zk-usdt", UsdtPolygon)}}}, nil
	}}
	aes := &mockAES{decryptFn: func(value, secret string) (string, error) {
		if value != "cipher-usdt" || secret != "the-secret" {
			return "", errors.New("wrong key")
		}
		return usdtPolygonAddr, nil
	}}
	result, err := svc(client, aes, nil).Verify(context.Background(), polURI(usdtPolygonAddr, "enc-usdt", "the-secret"), nil)
	if err != nil || result.Status != VerifyMatch {
		t.Fatalf("%+v %v", result, err)
	}
	if result.Payments[0].Destinations[0].Type != UsdtPolygon || result.Payments[0].Destinations[0].Value != usdtPolygonAddr {
		t.Fatalf("%+v", result.Payments[0].Destinations[0])
	}
	if len(result.MatchedDestinations) != 1 || result.MatchedDestinations[0].Type != UsdtPolygon {
		t.Fatalf("%+v", result.MatchedDestinations)
	}
}

func TestPolUsdtCaseInsensitiveMatch(t *testing.T) {
	client := &mockClient{getFn: func(string) ([]Payment, error) {
		return []Payment{{Destinations: []Destination{encDest("cipher-usdt", "zk-usdt", UsdtPolygon)}}}, nil
	}}
	aes := &mockAES{decryptFn: func(string, string) (string, error) { return usdtPolygonAddr, nil }}
	result, err := svc(client, aes, nil).Verify(context.Background(), polURI(strings.ToLower(usdtPolygonAddr), "enc-usdt", "the-secret"), nil)
	if err != nil || result.Status != VerifyMatch {
		t.Fatalf("%+v %v", result, err)
	}
}

func TestPolUsdt0xPrefixCaseInsensitiveMatch(t *testing.T) {
	client := &mockClient{getFn: func(string) ([]Payment, error) {
		return []Payment{{Destinations: []Destination{encDest("cipher-usdt", "zk-usdt", UsdtPolygon)}}}, nil
	}}
	aes := &mockAES{decryptFn: func(string, string) (string, error) { return usdtPolygonAddr, nil }}
	mixed := "0X" + usdtPolygonAddr[2:]
	result, err := svc(client, aes, nil).Verify(context.Background(), polURI(mixed, "enc-usdt", "the-secret"), nil)
	if err != nil || result.Status != VerifyMatch {
		t.Fatalf("%+v %v", result, err)
	}
}

func TestPolUsdtSwappedIsTampered(t *testing.T) {
	client := &mockClient{getFn: func(string) ([]Payment, error) {
		return []Payment{{Destinations: []Destination{encDest("cipher-usdt", "zk-usdt", UsdtPolygon)}}}, nil
	}}
	aes := &mockAES{decryptFn: func(string, string) (string, error) { return usdtPolygonAddr, nil }}
	result, err := svc(client, aes, nil).Verify(context.Background(), polURI(swappedPolygonAddr, "enc-usdt", "the-secret"), nil)
	if err != nil || result.Status != VerifyTampered {
		t.Fatalf("%+v %v", result, err)
	}
}

func TestBareUsdtAddressIsUnknown(t *testing.T) {
	client := &mockClient{}
	result, err := svc(client, nil, nil).Verify(context.Background(), usdtPolygonAddr, nil)
	if err != nil || result.Status != VerifyUnknown || len(client.getCalls) != 0 {
		t.Fatalf("%+v %v", result, err)
	}
}

func TestPolWithoutBrantaParamsIsUnknown(t *testing.T) {
	client := &mockClient{}
	result, err := svc(client, nil, nil).Verify(context.Background(), "pol:"+usdtPolygonAddr+"?value=50", nil)
	if err != nil || result.Status != VerifyUnknown || len(client.getCalls) != 0 {
		t.Fatalf("%+v %v", result, err)
	}
}

func TestPolIgnoresSiblingBitcoinRail(t *testing.T) {
	client := &mockClient{getFn: func(string) ([]Payment, error) {
		return []Payment{{Destinations: []Destination{
			encDest("cipher-btc", "zk-btc", BitcoinAddress),
			encDest("cipher-usdt", "zk-usdt", UsdtPolygon),
		}}}, nil
	}}
	aes := &mockAES{decryptFn: func(value, secret string) (string, error) {
		if value == "cipher-usdt" && secret == "the-secret" {
			return usdtPolygonAddr, nil
		}
		if value == "cipher-btc" {
			return addr, nil
		}
		return "", errors.New("no")
	}}
	result, err := svc(client, aes, nil).Verify(context.Background(), polURI(usdtPolygonAddr, "enc-usdt", "the-secret"), nil)
	if err != nil || result.Status != VerifyMatch {
		t.Fatalf("%+v %v", result, err)
	}
	dests := result.Payments[0].Destinations
	if !dests[0].IsEncrypted || dests[0].Type != BitcoinAddress {
		t.Fatalf("bitcoin should stay encrypted: %+v", dests[0])
	}
	if dests[1].IsEncrypted || dests[1].Value != usdtPolygonAddr {
		t.Fatalf("%+v", dests[1])
	}
}

func TestBitcoinIgnoresSiblingUsdtRail(t *testing.T) {
	client := &mockClient{getFn: func(string) ([]Payment, error) {
		return []Payment{{Destinations: []Destination{
			encDest("cipher-btc", "zk-btc", BitcoinAddress),
			encDest("cipher-usdt", "zk-usdt", UsdtPolygon),
		}}}, nil
	}}
	aes := &mockAES{decryptFn: func(value, secret string) (string, error) {
		if value == "cipher-btc" && secret == "the-secret" {
			return addr, nil
		}
		if value == "cipher-usdt" {
			return usdtPolygonAddr, nil
		}
		return "", errors.New("no")
	}}
	qr := "bitcoin:" + addr + "?branta_id=cipher-id&branta_secret=the-secret"
	result, err := svc(client, aes, nil).Verify(context.Background(), qr, nil)
	if err != nil || result.Status != VerifyMatch {
		t.Fatalf("%+v %v", result, err)
	}
	dests := result.Payments[0].Destinations
	if dests[0].IsEncrypted || dests[0].Value != addr {
		t.Fatalf("%+v", dests[0])
	}
	if !dests[1].IsEncrypted || dests[1].Type != UsdtPolygon {
		t.Fatalf("usdt should stay encrypted: %+v", dests[1])
	}
}

func TestPolSwappedStillTamperedWithSiblingBitcoin(t *testing.T) {
	client := &mockClient{getFn: func(string) ([]Payment, error) {
		return []Payment{{Destinations: []Destination{
			encDest("cipher-btc", "zk-btc", BitcoinAddress),
			encDest("cipher-usdt", "zk-usdt", UsdtPolygon),
		}}}, nil
	}}
	aes := &mockAES{decryptFn: func(value, _ string) (string, error) {
		if value == "cipher-usdt" {
			return usdtPolygonAddr, nil
		}
		return addr, nil
	}}
	result, err := svc(client, aes, nil).Verify(context.Background(), polURI(swappedPolygonAddr, "enc-usdt", "the-secret"), nil)
	if err != nil || result.Status != VerifyTampered {
		t.Fatalf("%+v %v", result, err)
	}
}

func TestPolMatchIfAnyDualUsdt(t *testing.T) {
	client := &mockClient{getFn: func(string) ([]Payment, error) {
		return []Payment{{Destinations: []Destination{
			encDest("cipher-sib", "zk-sib", UsdtPolygon),
			encDest("cipher-usdt", "zk-usdt", UsdtPolygon),
		}}}, nil
	}}
	aes := &mockAES{decryptFn: func(value, _ string) (string, error) {
		if value == "cipher-sib" {
			return siblingUsdtAddr, nil
		}
		return usdtPolygonAddr, nil
	}}
	result, err := svc(client, aes, nil).Verify(context.Background(), polURI(usdtPolygonAddr, "enc-usdt", "the-secret"), nil)
	if err != nil || result.Status != VerifyMatch {
		t.Fatalf("%+v %v", result, err)
	}
	dests := result.Payments[0].Destinations
	if !dests[0].IsEncrypted || dests[0].Value != "cipher-sib" {
		t.Fatalf("sibling should stay encrypted: %+v", dests[0])
	}
	if dests[1].IsEncrypted || dests[1].Value != usdtPolygonAddr {
		t.Fatalf("%+v", dests[1])
	}
}

func TestPolSwappedTamperedWhenNoPolygonFamilyRailMatches(t *testing.T) {
	client := &mockClient{getFn: func(string) ([]Payment, error) {
		return []Payment{{Destinations: []Destination{
			encDest("cipher-sib", "zk-sib", UsdtPolygon),
			encDest("cipher-usdt", "zk-usdt", UsdtPolygon),
		}}}, nil
	}}
	aes := &mockAES{decryptFn: func(value, _ string) (string, error) {
		if value == "cipher-sib" {
			return siblingUsdtAddr, nil
		}
		return usdtPolygonAddr, nil
	}}
	result, err := svc(client, aes, nil).Verify(context.Background(), polURI(swappedPolygonAddr, "enc-usdt", "the-secret"), nil)
	if err != nil || result.Status != VerifyTampered {
		t.Fatalf("%+v %v", result, err)
	}
}

func TestPolUsdcMatch(t *testing.T) {
	client := &mockClient{getFn: func(string) ([]Payment, error) {
		return []Payment{{Destinations: []Destination{encDest("cipher-usdc", "zk-usdc", UsdcPolygon)}}}, nil
	}}
	aes := &mockAES{decryptFn: func(string, string) (string, error) { return usdcPolygonAddr, nil }}
	result, err := svc(client, aes, nil).Verify(context.Background(), polURI(usdcPolygonAddr, "enc-usdc", "the-secret"), nil)
	if err != nil || result.Status != VerifyMatch {
		t.Fatalf("%+v %v", result, err)
	}
	if result.MatchedDestinations[0].Type != UsdcPolygon {
		t.Fatalf("%+v", result.MatchedDestinations)
	}
}

func TestPolUsdcSwappedIsTampered(t *testing.T) {
	client := &mockClient{getFn: func(string) ([]Payment, error) {
		return []Payment{{Destinations: []Destination{encDest("cipher-usdc", "zk-usdc", UsdcPolygon)}}}, nil
	}}
	aes := &mockAES{decryptFn: func(string, string) (string, error) { return usdcPolygonAddr, nil }}
	result, err := svc(client, aes, nil).Verify(context.Background(), polURI(swappedPolygonAddr, "enc-usdc", "the-secret"), nil)
	if err != nil || result.Status != VerifyTampered {
		t.Fatalf("%+v %v", result, err)
	}
}

func TestPolBindsUsdtLeavesUsdcEncrypted(t *testing.T) {
	client := &mockClient{getFn: func(string) ([]Payment, error) {
		return []Payment{{Destinations: []Destination{
			encDest("cipher-usdt", "zk-usdt", UsdtPolygon),
			encDest("cipher-usdc", "zk-usdc", UsdcPolygon),
		}}}, nil
	}}
	aes := &mockAES{decryptFn: func(value, _ string) (string, error) {
		if value == "cipher-usdt" {
			return usdtPolygonAddr, nil
		}
		return usdcPolygonAddr, nil
	}}
	result, err := svc(client, aes, nil).Verify(context.Background(), polURI(usdtPolygonAddr, "enc-usdt", "the-secret"), nil)
	if err != nil || result.Status != VerifyMatch {
		t.Fatalf("%+v %v", result, err)
	}
	dests := result.Payments[0].Destinations
	if dests[0].IsEncrypted || dests[0].Type != UsdtPolygon {
		t.Fatalf("%+v", dests[0])
	}
	if !dests[1].IsEncrypted || dests[1].Type != UsdcPolygon {
		t.Fatalf("%+v", dests[1])
	}
}

func TestPolBindsUsdcLeavesUsdtEncrypted(t *testing.T) {
	client := &mockClient{getFn: func(string) ([]Payment, error) {
		return []Payment{{Destinations: []Destination{
			encDest("cipher-usdt", "zk-usdt", UsdtPolygon),
			encDest("cipher-usdc", "zk-usdc", UsdcPolygon),
		}}}, nil
	}}
	aes := &mockAES{decryptFn: func(value, _ string) (string, error) {
		if value == "cipher-usdt" {
			return usdtPolygonAddr, nil
		}
		return usdcPolygonAddr, nil
	}}
	result, err := svc(client, aes, nil).Verify(context.Background(), polURI(usdcPolygonAddr, "enc-usdc", "the-secret"), nil)
	if err != nil || result.Status != VerifyMatch {
		t.Fatalf("%+v %v", result, err)
	}
	dests := result.Payments[0].Destinations
	if !dests[0].IsEncrypted || dests[0].Type != UsdtPolygon {
		t.Fatalf("%+v", dests[0])
	}
	if dests[1].IsEncrypted || dests[1].Type != UsdcPolygon {
		t.Fatalf("%+v", dests[1])
	}
}

func TestPolWrongSecretIsUnknown(t *testing.T) {
	client := &mockClient{getFn: func(string) ([]Payment, error) {
		return []Payment{{Destinations: []Destination{encDest("cipher-usdt", "zk-usdt", UsdtPolygon)}}}, nil
	}}
	aes := &mockAES{decryptFn: func(string, string) (string, error) {
		return "", errors.New("wrong key")
	}}
	result, err := svc(client, aes, nil).Verify(context.Background(), polURI(usdtPolygonAddr, "enc-usdt", "wrong-secret"), nil)
	if err != nil || result.Status != VerifyUnknown {
		t.Fatalf("%+v %v", result, err)
	}
}

func TestSameAddressUsdtAndUsdcBothMatch(t *testing.T) {
	client := &mockClient{getFn: func(string) ([]Payment, error) {
		return []Payment{{Destinations: []Destination{
			encDest("cipher-usdt", "zk-usdt", UsdtPolygon),
			encDest("cipher-usdc", "zk-usdc", UsdcPolygon),
		}}}, nil
	}}
	aes := &mockAES{decryptFn: func(string, string) (string, error) {
		return usdtPolygonAddr, nil
	}}
	result, err := svc(client, aes, nil).Verify(context.Background(), polURI(usdtPolygonAddr, "enc-both", "the-secret"), nil)
	if err != nil || result.Status != VerifyMatch {
		t.Fatalf("%+v %v", result, err)
	}
	if len(result.MatchedDestinations) != 2 {
		t.Fatalf("%+v", result.MatchedDestinations)
	}
}

func TestAddPaymentEncryptsUsdtPolygon(t *testing.T) {
	client := &mockClient{}
	payment := Payment{Destinations: []Destination{{Value: usdtPolygonAddr, Type: UsdtPolygon, ID: "zk-1"}}}
	added, err := svc(client, nil, &mockSecrets{values: []string{"polygon-secret"}}).AddPayment(context.Background(), payment, nil)
	if err != nil {
		t.Fatal(err)
	}
	if payment.Destinations[0].Value != usdtPolygonAddr {
		t.Fatal("caller mutated")
	}
	if client.posted[0].Destinations[0].Value == usdtPolygonAddr || client.posted[0].Destinations[0].Type != UsdtPolygon {
		t.Fatalf("%+v", client.posted[0].Destinations[0])
	}
	if added.Secret != "polygon-secret" {
		t.Fatal(added.Secret)
	}
}

func TestAddPaymentEncryptsUsdcPolygon(t *testing.T) {
	client := &mockClient{}
	payment := Payment{Destinations: []Destination{{Value: usdcPolygonAddr, Type: UsdcPolygon, ID: "zk-1"}}}
	_, err := svc(client, nil, &mockSecrets{values: []string{"polygon-secret"}}).AddPayment(context.Background(), payment, nil)
	if err != nil {
		t.Fatal(err)
	}
	if client.posted[0].Destinations[0].Type != UsdcPolygon {
		t.Fatalf("%+v", client.posted[0].Destinations[0])
	}
}

func TestAddPaymentBareOxWithoutTypeUnsupported(t *testing.T) {
	client := &mockClient{}
	payment := Payment{Destinations: []Destination{{Value: usdtPolygonAddr}}}
	_, err := svc(client, nil, nil).AddPayment(context.Background(), payment, nil)
	var unsupported *UnsupportedDestinationTypeError
	if !errors.As(err, &unsupported) {
		t.Fatal(err)
	}
}

const usdtEthereumAddr = "0xBF5550253939187432Ef4dccE8089765d3DCb47f"
const usdtTronAddr = "T9yD14Nj9j7xAB4dbGeiX9h8unkKHxuWwb"
const usdcSolanaAddr = "7EqQdEULxWcraVx3mXKFjc84LhCkMGZCkRuDofycyinf"

func chainURI(scheme, address, encrypted, secret string) string {
	return scheme + ":" + address + "?value=50&branta_id=" + encrypted + "&branta_secret=" + secret
}

func verifyChainMatch(t *testing.T, scheme, address string, destType DestinationType) VerifyResult {
	t.Helper()
	client := &mockClient{getFn: func(lookup string) ([]Payment, error) {
		if lookup != "enc-chain" {
			t.Fatal(lookup)
		}
		return []Payment{{Destinations: []Destination{encDest("cipher-chain", "zk-chain", destType)}}}, nil
	}}
	aes := &mockAES{decryptFn: func(value, secret string) (string, error) {
		if value != "cipher-chain" || secret != "the-secret" {
			return "", errors.New("wrong key")
		}
		return address, nil
	}}
	result, err := svc(client, aes, nil).Verify(context.Background(), chainURI(scheme, address, "enc-chain", "the-secret"), nil)
	if err != nil || result.Status != VerifyMatch {
		t.Fatalf("%+v %v", result, err)
	}
	if len(result.MatchedDestinations) != 1 ||
		result.MatchedDestinations[0].Type != destType ||
		result.MatchedDestinations[0].Value != address {
		t.Fatalf("%+v", result.MatchedDestinations)
	}
	return result
}

func TestEthUsdtMatch(t *testing.T) {
	verifyChainMatch(t, "eth", usdtEthereumAddr, UsdtEthereum)
}

func TestTronUsdtMatch(t *testing.T) {
	verifyChainMatch(t, "tron", usdtTronAddr, UsdtTron)
}

func TestSolUsdcMatch(t *testing.T) {
	verifyChainMatch(t, "sol", usdcSolanaAddr, UsdcSolana)
}

func TestTronSwappedTampered(t *testing.T) {
	client := &mockClient{getFn: func(string) ([]Payment, error) {
		return []Payment{{Destinations: []Destination{encDest("cipher-chain", "zk-chain", UsdtTron)}}}, nil
	}}
	aes := &mockAES{decryptFn: func(string, string) (string, error) { return usdtTronAddr, nil }}
	swapped := usdtTronAddr[:len(usdtTronAddr)-1] + "x"
	result, err := svc(client, aes, nil).Verify(context.Background(), chainURI("tron", swapped, "enc-chain", "the-secret"), nil)
	if err != nil || result.Status != VerifyTampered {
		t.Fatalf("%+v %v", result, err)
	}
}

func TestEthSchemeIgnoresPolygonFamilyRail(t *testing.T) {
	client := &mockClient{getFn: func(string) ([]Payment, error) {
		return []Payment{{Destinations: []Destination{
			encDest("cipher-poly", "zk-poly", UsdtPolygon),
			encDest("cipher-eth", "zk-eth", UsdtEthereum),
		}}}, nil
	}}
	aes := &mockAES{decryptFn: func(value, _ string) (string, error) {
		if value == "cipher-poly" {
			return usdtPolygonAddr, nil
		}
		return usdtEthereumAddr, nil
	}}
	result, err := svc(client, aes, nil).Verify(context.Background(), chainURI("eth", usdtEthereumAddr, "enc-chain", "the-secret"), nil)
	if err != nil || result.Status != VerifyMatch {
		t.Fatalf("%+v %v", result, err)
	}
	dests := result.Payments[0].Destinations
	if !dests[0].IsEncrypted || dests[1].IsEncrypted || len(result.MatchedDestinations) != 1 {
		t.Fatalf("%+v", dests)
	}
}

func TestBareStablecoinAddressesAreUnknown(t *testing.T) {
	for _, address := range []string{usdtEthereumAddr, usdtTronAddr, usdcSolanaAddr} {
		client := &mockClient{getFn: func(string) ([]Payment, error) {
			t.Fatal("must not GET")
			return nil, nil
		}}
		result, err := svc(client, nil, nil).Verify(context.Background(), address, nil)
		if err != nil || result.Status != VerifyUnknown {
			t.Fatalf("%s: %+v %v", address, result, err)
		}
	}
}

func TestAddPaymentEncryptsUsdtTronWithSecret(t *testing.T) {
	client := &mockClient{}
	payment := Payment{Destinations: []Destination{{Value: usdtTronAddr, Type: UsdtTron, ID: "zk-1"}}}
	added, err := svc(client, nil, &mockSecrets{values: []string{"tron-secret"}}).AddPayment(context.Background(), payment, nil)
	if err != nil {
		t.Fatal(err)
	}
	if payment.Destinations[0].Value != usdtTronAddr {
		t.Fatal("caller mutated")
	}
	posted := client.posted[0].Destinations[0]
	if posted.Value == usdtTronAddr || posted.Type != UsdtTron {
		t.Fatalf("%+v", posted)
	}
	if added.Secret != "tron-secret" {
		t.Fatal(added.Secret)
	}
}
