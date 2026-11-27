package branta

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
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

func TestCombinedBrantaQrDecryptsAddressAndInvoice(t *testing.T) {
	client := &mockClient{getFn: func(lookup string) ([]Payment, error) {
		if lookup != "cipher-id" {
			t.Fatal(lookup)
		}
		return []Payment{{Destinations: []Destination{
			encDest("cipher-btc", "zk-btc", BitcoinAddress),
			encDest("cipher-bolt11", "zk-bolt11", Bolt11),
		}}}, nil
	}}
	aes := &mockAES{decryptFn: func(value, secret string) (string, error) {
		if value == "cipher-btc" && secret == "the-secret" {
			return addr, nil
		}
		if value == "cipher-bolt11" && secret == ToNormalizedHash(invoice) {
			return invoice, nil
		}
		return "", errors.New("miss")
	}}
	qr := "bitcoin:" + addr + "?lightning=" + invoice + "&branta_id=cipher-id&branta_secret=the-secret"
	result, err := svc(client, aes, nil).Verify(context.Background(), qr, nil)
	if err != nil || result.Status != VerifyMatch {
		t.Fatalf("%+v %v", result, err)
	}
	dests := result.Payments[0].Destinations
	if dests[0].Value != addr || dests[1].Value != invoice {
		t.Fatalf("%+v", dests)
	}
	if len(result.MatchedDestinations) != 2 ||
		result.MatchedDestinations[0].Type != BitcoinAddress ||
		result.MatchedDestinations[0].Value != addr ||
		result.MatchedDestinations[1].Type != Bolt11 ||
		result.MatchedDestinations[1].Value != invoice {
		t.Fatalf("%+v", result.MatchedDestinations)
	}
}

func TestCombinedBrantaQrSwappedAddressIsTampered(t *testing.T) {
	client := &mockClient{getFn: func(string) ([]Payment, error) {
		return []Payment{{Destinations: []Destination{
			encDest("cipher-btc", "zk-btc", BitcoinAddress),
			encDest("cipher-bolt11", "zk-bolt11", Bolt11),
		}}}, nil
	}}
	aes := &mockAES{decryptFn: func(value, secret string) (string, error) {
		if value == "cipher-btc" {
			return "bc1qother", nil
		}
		if value == "cipher-bolt11" {
			return invoice, nil
		}
		return "", errors.New("miss")
	}}
	qr := "bitcoin:" + addr + "?lightning=" + invoice + "&branta_id=cipher-id&branta_secret=the-secret"
	result, err := svc(client, aes, nil).Verify(context.Background(), qr, nil)
	if err != nil || result.Status != VerifyTampered || result.Payments != nil {
		t.Fatalf("%+v %v", result, err)
	}
}

func TestCombinedBrantaQrAttackerLightningIsUnknown(t *testing.T) {
	// Payment only has bitcoin — attacker bolt11 cannot bind (same as the other SDKs).
	client := &mockClient{getFn: func(string) ([]Payment, error) {
		return []Payment{{Destinations: []Destination{
			encDest("cipher-btc", "zk-btc", BitcoinAddress),
		}}}, nil
	}}
	aes := &mockAES{decryptFn: func(value, secret string) (string, error) {
		if value == "cipher-btc" && secret == "the-secret" {
			return addr, nil
		}
		return "", errors.New("miss")
	}}
	qr := "bitcoin:" + addr + "?lightning=lnbc1attacker&branta_id=cipher-id&branta_secret=the-secret"
	result, err := svc(client, aes, nil).Verify(context.Background(), qr, nil)
	if err != nil || result.Status != VerifyUnknown {
		t.Fatalf("%+v %v", result, err)
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

func TestAddPaymentLeavesCallerBrandUnchangedAndCopyCarriesBrand(t *testing.T) {
	cases := map[string]Payment{
		"brand": NewPaymentBuilder().AddDestination(addr, BitcoinAddress).
			SetBrand("Shop", "https://example.com/l.png", "").Build(),
		"deprecated field": {
			Destinations:  []Destination{NewDestination(addr, BitcoinAddress)},
			ChildPlatform: &Platform{Name: "Shop", LogoURL: "https://example.com/l.png"},
		},
	}
	for name, payment := range cases {
		client := &mockClient{}
		before := payment
		beforeBrand, beforeChild := payment.Brand, payment.ChildPlatform
		if _, err := svc(client, nil, nil).AddPayment(context.Background(), payment, nil); err != nil {
			t.Fatal(name, err)
		}
		if payment.Brand != beforeBrand || payment.ChildPlatform != beforeChild || payment.Destinations[0].Value != addr {
			t.Fatalf("%s: caller changed: %+v vs %+v", name, payment, before)
		}
		posted := client.posted[0]
		if posted.Brand == nil || posted.Brand.Name != "Shop" || posted.Brand.LogoURL != "https://example.com/l.png" {
			t.Fatalf("%s: posted brand %+v", name, posted.Brand)
		}
		if posted.Brand == payment.Brand || posted.Brand == payment.ChildPlatform {
			t.Fatalf("%s: posted brand must be a copy", name)
		}
		if posted.ChildPlatform == nil || posted.ChildPlatform == posted.Brand {
			t.Fatalf("%s: posted deprecated mirror %+v", name, posted.ChildPlatform)
		}
		obj := wireObject(t, posted)
		if _, ok := obj["child_platform"]; ok || obj["brand"] == nil {
			t.Fatalf("%s: wire %v", name, obj)
		}
	}
}

func TestAddPaymentBrandWinsOverChildPlatform(t *testing.T) {
	client := &mockClient{}
	payment := Payment{
		Destinations:  []Destination{NewDestination(addr, BitcoinAddress)},
		Brand:         &Platform{Name: "New"},
		ChildPlatform: &Platform{Name: "Old"},
	}
	if _, err := svc(client, nil, nil).AddPayment(context.Background(), payment, nil); err != nil {
		t.Fatal(err)
	}
	if client.posted[0].Brand.Name != "New" || payment.ChildPlatform.Name != "Old" {
		t.Fatalf("%+v", client.posted[0])
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

const (
	usdtPolygonAddr  = "0x278297528c32eCa01295B0d2135984d3442ec963"
	usdcPolygonAddr  = "0xAb5801a7D398351b8bE11C439e05C5B3259aeC9B"
	siblingUsdtAddr  = "0x2222222222222222222222222222222222222222"
	swappedEvmAddr   = "0x3333333333333333333333333333333333333333"
	usdtEthereumAddr = "0xBF5550253939187432Ef4dccE8089765d3DCb47f"
	usdcEthereumAddr = "0xA0b86991c6218b36c1d19D4a2e9Eb0cE3606eB48"
	usdtTronAddr     = "T9yD14Nj9j7xAB4dbGeiX9h8unkKHxuWwb"
	usdtTonAddr      = "EQDtFpEwcFAEcRe5mLVh2N6C0y-_tur3BYOM93pEZPw390by"
	usdcSolanaAddr   = "7EqQdEULxWcraVx3mXKFjc84LhCkMGZCkRuDofycyinf"
)

func railFor(t *testing.T, destType DestinationType) evmTokenRail {
	t.Helper()
	for _, rail := range evmTokenRails {
		if rail.destType == destType {
			return rail
		}
	}
	t.Fatalf("no rail for %s", destType)
	return evmTokenRail{}
}

// transferURI builds an ERC-681 transfer URI for one EVM rail, as a wallet would show it.
func transferURI(t *testing.T, destType DestinationType, address, encrypted, secret string) string {
	t.Helper()
	rail := railFor(t, destType)
	return "ethereum:" + rail.contract + "@" + strconv.FormatInt(rail.chainID, 10) +
		"/transfer?address=" + address + "&uint256=1000000&branta_id=" + encrypted + "&branta_secret=" + secret
}

// depositFor returns one mixed-case deposit per rail, so a lowercase scan still
// differs from the stored text.
func depositFor(destType DestinationType) string {
	for i, rail := range evmTokenRails {
		if rail.destType == destType {
			hex := strconv.FormatInt(int64(i+1), 16)
			return "0xAb" + strings.Repeat("1", 38-len(hex)) + hex
		}
	}
	return ""
}

// freshClient returns a client that builds a new payment on every GET, so a
// test can verify twice without sharing mutated destinations.
func freshClient(dests ...Destination) *mockClient {
	return &mockClient{getFn: func(string) ([]Payment, error) {
		return []Payment{{Destinations: append([]Destination(nil), dests...)}}, nil
	}}
}

// decryptTable decrypts a ciphertext to its plaintext when the secret is "the-secret".
func decryptTable(table map[string]string) *mockAES {
	return &mockAES{decryptFn: func(value, secret string) (string, error) {
		plain, ok := table[value]
		if !ok || secret != "the-secret" {
			return "", errors.New("wrong key")
		}
		return plain, nil
	}}
}

func verifyQR(t *testing.T, client *mockClient, aes *mockAES, qr string) VerifyResult {
	t.Helper()
	result, err := svc(client, aes, nil).Verify(context.Background(), qr, nil)
	if err != nil {
		t.Fatalf("%.60s: %v", qr, err)
	}
	return result
}

func wantStatus(t *testing.T, result VerifyResult, want VerifyStatus, label string) {
	t.Helper()
	if result.Status != want {
		t.Fatalf("%s: want %s got %s (%+v)", label, want, result.Status, result)
	}
}

func matchedTypes(result VerifyResult) []DestinationType {
	var out []DestinationType
	for _, d := range result.MatchedDestinations {
		out = append(out, d.Type)
	}
	return out
}

func wantTypes(t *testing.T, got []DestinationType, want ...DestinationType) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("want %v got %v", want, got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("want %v got %v", want, got)
		}
	}
}

// ---- Polygon

func TestPolygonUsdtMatchIgnoresAmountAndSetsMatchedDestinations(t *testing.T) {
	client := freshClient(encDest("cipher-usdt", "zk-usdt", UsdtPolygon))
	aes := decryptTable(map[string]string{"cipher-usdt": usdtPolygonAddr})
	result := verifyQR(t, client, aes, transferURI(t, UsdtPolygon, usdtPolygonAddr, "enc-usdt", "the-secret"))
	wantStatus(t, result, VerifyMatch, "match")
	if len(client.getCalls) != 1 || client.getCalls[0] != "enc-usdt" {
		t.Fatal(client.getCalls)
	}
	dest := result.Payments[0].Destinations[0]
	if dest.Type != UsdtPolygon || dest.Value != usdtPolygonAddr {
		t.Fatalf("%+v", dest)
	}
	wantTypes(t, matchedTypes(result), UsdtPolygon)
}

func TestAmountIsNotPartOfTheMatch(t *testing.T) {
	for _, amount := range []string{"1", "50000000", "999999999999"} {
		client := freshClient(encDest("cipher-usdt", "zk-usdt", UsdtPolygon))
		aes := decryptTable(map[string]string{"cipher-usdt": usdtPolygonAddr})
		qr := strings.Replace(transferURI(t, UsdtPolygon, usdtPolygonAddr, "enc-usdt", "the-secret"), "uint256=1000000", "uint256="+amount, 1)
		wantStatus(t, verifyQR(t, client, aes, qr), VerifyMatch, amount)
	}
}

func TestPolygonUsdtCaseInsensitiveAnd0XPrefixMatch(t *testing.T) {
	for _, address := range []string{strings.ToLower(usdtPolygonAddr), "0X" + usdtPolygonAddr[2:]} {
		client := freshClient(encDest("cipher-usdt", "zk-usdt", UsdtPolygon))
		aes := decryptTable(map[string]string{"cipher-usdt": usdtPolygonAddr})
		wantStatus(t, verifyQR(t, client, aes, transferURI(t, UsdtPolygon, address, "enc-usdt", "the-secret")), VerifyMatch, address)
	}
}

func TestPolygonUsdtSwappedIsTampered(t *testing.T) {
	client := freshClient(encDest("cipher-usdt", "zk-usdt", UsdtPolygon))
	aes := decryptTable(map[string]string{"cipher-usdt": usdtPolygonAddr})
	result := verifyQR(t, client, aes, transferURI(t, UsdtPolygon, swappedEvmAddr, "enc-usdt", "the-secret"))
	wantStatus(t, result, VerifyTampered, "swapped")
	if len(result.MatchedDestinations) != 0 {
		t.Fatalf("%+v", result.MatchedDestinations)
	}
}

func TestPolygonScanIgnoresSiblingBitcoinRail(t *testing.T) {
	client := freshClient(
		encDest("cipher-btc", "zk-btc", BitcoinAddress),
		encDest("cipher-usdt", "zk-usdt", UsdtPolygon),
	)
	aes := decryptTable(map[string]string{"cipher-btc": addr, "cipher-usdt": usdtPolygonAddr})
	result := verifyQR(t, client, aes, transferURI(t, UsdtPolygon, usdtPolygonAddr, "enc-usdt", "the-secret"))
	wantStatus(t, result, VerifyMatch, "match")
	dests := result.Payments[0].Destinations
	if !dests[0].IsEncrypted || dests[0].Type != BitcoinAddress {
		t.Fatalf("bitcoin should stay encrypted: %+v", dests[0])
	}
	if dests[1].IsEncrypted || dests[1].Value != usdtPolygonAddr {
		t.Fatalf("%+v", dests[1])
	}
}

func TestBitcoinScanIgnoresSiblingUsdtRail(t *testing.T) {
	client := freshClient(
		encDest("cipher-btc", "zk-btc", BitcoinAddress),
		encDest("cipher-usdt", "zk-usdt", UsdtPolygon),
	)
	aes := decryptTable(map[string]string{"cipher-btc": addr, "cipher-usdt": usdtPolygonAddr})
	result := verifyQR(t, client, aes, "bitcoin:"+addr+"?branta_id=cipher-id&branta_secret=the-secret")
	wantStatus(t, result, VerifyMatch, "match")
	dests := result.Payments[0].Destinations
	if dests[0].IsEncrypted || dests[0].Value != addr {
		t.Fatalf("%+v", dests[0])
	}
	if !dests[1].IsEncrypted || dests[1].Type != UsdtPolygon {
		t.Fatalf("usdt should stay encrypted: %+v", dests[1])
	}
}

func TestPolygonSwappedIsStillTamperedWithSiblingBitcoin(t *testing.T) {
	client := freshClient(
		encDest("cipher-btc", "zk-btc", BitcoinAddress),
		encDest("cipher-usdt", "zk-usdt", UsdtPolygon),
	)
	aes := decryptTable(map[string]string{"cipher-btc": addr, "cipher-usdt": usdtPolygonAddr})
	wantStatus(t, verifyQR(t, client, aes, transferURI(t, UsdtPolygon, swappedEvmAddr, "enc-usdt", "the-secret")), VerifyTampered, "swapped")
}

func TestASecondUsdtDestinationWithAnotherAddressStaysEncrypted(t *testing.T) {
	client := freshClient(
		encDest("cipher-sib", "zk-sib", UsdtPolygon),
		encDest("cipher-usdt", "zk-usdt", UsdtPolygon),
	)
	aes := decryptTable(map[string]string{"cipher-sib": siblingUsdtAddr, "cipher-usdt": usdtPolygonAddr})
	result := verifyQR(t, client, aes, transferURI(t, UsdtPolygon, usdtPolygonAddr, "enc-usdt", "the-secret"))
	wantStatus(t, result, VerifyMatch, "match")
	dests := result.Payments[0].Destinations
	if !dests[0].IsEncrypted || dests[0].Value != "cipher-sib" {
		t.Fatalf("%+v", dests[0])
	}
	if dests[1].IsEncrypted || dests[1].Value != usdtPolygonAddr {
		t.Fatalf("%+v", dests[1])
	}
}

func TestSwappedIsTamperedWhenNoDestinationOfTheTypeMatches(t *testing.T) {
	client := freshClient(
		encDest("cipher-sib", "zk-sib", UsdtPolygon),
		encDest("cipher-usdt", "zk-usdt", UsdtPolygon),
	)
	aes := decryptTable(map[string]string{"cipher-sib": siblingUsdtAddr, "cipher-usdt": usdtPolygonAddr})
	wantStatus(t, verifyQR(t, client, aes, transferURI(t, UsdtPolygon, swappedEvmAddr, "enc-usdt", "the-secret")), VerifyTampered, "swapped")
}

func TestPolygonUsdcMatchAndSwapped(t *testing.T) {
	aes := decryptTable(map[string]string{"cipher-usdc": usdcPolygonAddr})
	match := verifyQR(t, freshClient(encDest("cipher-usdc", "zk-usdc", UsdcPolygon)), aes,
		transferURI(t, UsdcPolygon, usdcPolygonAddr, "enc-usdc", "the-secret"))
	wantStatus(t, match, VerifyMatch, "match")
	wantTypes(t, matchedTypes(match), UsdcPolygon)
	swapped := verifyQR(t, freshClient(encDest("cipher-usdc", "zk-usdc", UsdcPolygon)), aes,
		transferURI(t, UsdcPolygon, swappedEvmAddr, "enc-usdc", "the-secret"))
	wantStatus(t, swapped, VerifyTampered, "swapped")
}

func TestAScanBindsTheScannedTokenAndLeavesTheOtherEncrypted(t *testing.T) {
	aes := decryptTable(map[string]string{"cipher-usdt": usdtPolygonAddr, "cipher-usdc": usdcPolygonAddr})
	cases := []struct {
		scan       DestinationType
		address    string
		boundIndex int
	}{
		{UsdtPolygon, usdtPolygonAddr, 0},
		{UsdcPolygon, usdcPolygonAddr, 1},
	}
	for _, tc := range cases {
		client := freshClient(
			encDest("cipher-usdt", "zk-usdt", UsdtPolygon),
			encDest("cipher-usdc", "zk-usdc", UsdcPolygon),
		)
		result := verifyQR(t, client, aes, transferURI(t, tc.scan, tc.address, "enc-scan", "the-secret"))
		wantStatus(t, result, VerifyMatch, string(tc.scan))
		for i, d := range result.Payments[0].Destinations {
			if d.IsEncrypted == (i == tc.boundIndex) {
				t.Fatalf("%s #%d: %+v", tc.scan, i, d)
			}
		}
	}
}

func TestPolygonWrongSecretIsUnknown(t *testing.T) {
	client := freshClient(encDest("cipher-usdt", "zk-usdt", UsdtPolygon))
	aes := decryptTable(map[string]string{"cipher-usdt": usdtPolygonAddr})
	wantStatus(t, verifyQR(t, client, aes, transferURI(t, UsdtPolygon, usdtPolygonAddr, "enc-usdt", "wrong-secret")), VerifyUnknown, "wrong secret")
}

func TestTheSameAddressUnderTwoTokensBindsOnlyTheScannedToken(t *testing.T) {
	aes := decryptTable(map[string]string{"cipher-usdt": usdtPolygonAddr, "cipher-usdc": usdtPolygonAddr})
	for _, scan := range []DestinationType{UsdtPolygon, UsdcPolygon} {
		client := freshClient(
			encDest("cipher-usdt", "zk-usdt", UsdtPolygon),
			encDest("cipher-usdc", "zk-usdc", UsdcPolygon),
		)
		result := verifyQR(t, client, aes, transferURI(t, scan, usdtPolygonAddr, "enc-both", "the-secret"))
		wantStatus(t, result, VerifyMatch, string(scan))
		wantTypes(t, matchedTypes(result), scan)
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

// ---- Every EVM rail

func TestEveryEVMRailMatchesAndReturnsTheScannedType(t *testing.T) {
	for _, rail := range evmTokenRails {
		destType := rail.destType
		address := depositFor(destType)
		aes := decryptTable(map[string]string{"cipher": address})

		client := freshClient(encDest("cipher", "zk", destType))
		result := verifyQR(t, client, aes, transferURI(t, destType, address, "enc-"+string(destType), "the-secret"))
		wantStatus(t, result, VerifyMatch, string(destType))
		if len(client.getCalls) != 1 || client.getCalls[0] != "enc-"+string(destType) {
			t.Fatalf("%s: %v", destType, client.getCalls)
		}
		wantTypes(t, matchedTypes(result), destType)
		if result.MatchedDestinations[0].Value != address {
			t.Fatalf("%s: %+v", destType, result.MatchedDestinations)
		}

		lower := verifyQR(t, freshClient(encDest("cipher", "zk", destType)), aes,
			transferURI(t, destType, strings.ToLower(address), "enc", "the-secret"))
		wantStatus(t, lower, VerifyMatch, string(destType)+" lowercase")

		tampered := verifyQR(t, freshClient(encDest("cipher", "zk", destType)), aes,
			transferURI(t, destType, swappedEvmAddr, "enc", "the-secret"))
		wantStatus(t, tampered, VerifyTampered, string(destType)+" swapped")
		if len(tampered.MatchedDestinations) != 0 {
			t.Fatalf("%s: %+v", destType, tampered.MatchedDestinations)
		}
	}
}

func TestEveryEVMRailOnTheWrongChainOrWithoutBrantaParamsIsUnknownWithoutAGet(t *testing.T) {
	client := &mockClient{getFn: func(string) ([]Payment, error) {
		t.Fatal("must not GET")
		return nil, nil
	}}
	for _, rail := range evmTokenRails {
		address := depositFor(rail.destType)
		wrongChain := verifyQR(t, client, nil,
			"ethereum:"+rail.contract+"@999999/transfer?address="+address+"&branta_id=enc&branta_secret=the-secret")
		wantStatus(t, wrongChain, VerifyUnknown, string(rail.destType)+" wrong chain")
		noParams := verifyQR(t, client, nil,
			"ethereum:"+rail.contract+"@"+strconv.FormatInt(rail.chainID, 10)+"/transfer?address="+address+"&uint256=1000000")
		wantStatus(t, noParams, VerifyUnknown, string(rail.destType)+" no params")
	}
}

func TestAnEthereumScanIgnoresASiblingBitcoinRail(t *testing.T) {
	aes := decryptTable(map[string]string{"cipher-btc": addr, "cipher-usdt": usdtEthereumAddr})
	dests := []Destination{
		encDest("cipher-btc", "zk-btc", BitcoinAddress),
		encDest("cipher-usdt", "zk-usdt", UsdtEthereum),
	}
	result := verifyQR(t, freshClient(dests...), aes, transferURI(t, UsdtEthereum, usdtEthereumAddr, "enc-usdt", "the-secret"))
	wantStatus(t, result, VerifyMatch, "match")
	if !result.Payments[0].Destinations[0].IsEncrypted {
		t.Fatal("bitcoin should stay encrypted")
	}
	wantTypes(t, matchedTypes(result), UsdtEthereum)
	swapped := verifyQR(t, freshClient(dests...), aes, transferURI(t, UsdtEthereum, swappedEvmAddr, "enc-usdt", "the-secret"))
	wantStatus(t, swapped, VerifyTampered, "swapped")
}

func TestADualTokenPaymentBindsOnlyTheScannedToken(t *testing.T) {
	aes := decryptTable(map[string]string{"cipher-usdt": usdtEthereumAddr, "cipher-usdc": usdcEthereumAddr})
	dual := []Destination{
		encDest("cipher-usdt", "zk-usdt", UsdtEthereum),
		encDest("cipher-usdc", "zk-usdc", UsdcEthereum),
	}

	usdt := verifyQR(t, freshClient(dual...), aes, transferURI(t, UsdtEthereum, usdtEthereumAddr, "enc-usdt", "the-secret"))
	wantStatus(t, usdt, VerifyMatch, "usdt")
	wantTypes(t, matchedTypes(usdt), UsdtEthereum)
	if usdt.Payments[0].Destinations[0].IsEncrypted || !usdt.Payments[0].Destinations[1].IsEncrypted {
		t.Fatalf("%+v", usdt.Payments[0].Destinations)
	}

	usdc := verifyQR(t, freshClient(dual...), aes, transferURI(t, UsdcEthereum, usdcEthereumAddr, "enc-usdt", "the-secret"))
	wantStatus(t, usdc, VerifyMatch, "usdc")
	wantTypes(t, matchedTypes(usdc), UsdcEthereum)
	if !usdc.Payments[0].Destinations[0].IsEncrypted || usdc.Payments[0].Destinations[1].IsEncrypted {
		t.Fatalf("%+v", usdc.Payments[0].Destinations)
	}

	swapped := verifyQR(t, freshClient(dual...), aes, transferURI(t, UsdtEthereum, swappedEvmAddr, "enc-usdt", "the-secret"))
	wantStatus(t, swapped, VerifyTampered, "swapped")
}

func TestAUsdcScanAgainstAUsdtOnlyPaymentIsUnknown(t *testing.T) {
	client := freshClient(encDest("cipher-usdt", "zk-usdt", UsdtEthereum))
	aes := decryptTable(map[string]string{"cipher-usdt": usdtEthereumAddr})
	result := verifyQR(t, client, aes, transferURI(t, UsdcEthereum, usdcEthereumAddr, "enc-usdt", "the-secret"))
	wantStatus(t, result, VerifyUnknown, "usdc vs usdt-only")
	if len(result.MatchedDestinations) != 0 {
		t.Fatalf("%+v", result.MatchedDestinations)
	}
}

func TestAnEthereumScanDoesNotMatchAPolygonOnlyPayment(t *testing.T) {
	client := freshClient(encDest("cipher-pol", "zk-pol", UsdtPolygon))
	aes := decryptTable(map[string]string{"cipher-pol": usdtPolygonAddr})
	result := verifyQR(t, client, aes, transferURI(t, UsdtEthereum, usdtEthereumAddr, "enc-chain", "the-secret"))
	wantStatus(t, result, VerifyUnknown, "eth vs polygon-only")
	if len(result.MatchedDestinations) != 0 {
		t.Fatalf("%+v", result.MatchedDestinations)
	}
}

func TestAnEthereumScanLeavesAPolygonRailInTheSamePaymentEncrypted(t *testing.T) {
	client := freshClient(
		encDest("cipher-pol", "zk-pol", UsdtPolygon),
		encDest("cipher-eth", "zk-eth", UsdtEthereum),
	)
	aes := decryptTable(map[string]string{"cipher-pol": usdtPolygonAddr, "cipher-eth": usdtEthereumAddr})
	result := verifyQR(t, client, aes, transferURI(t, UsdtEthereum, usdtEthereumAddr, "enc-eth", "the-secret"))
	wantStatus(t, result, VerifyMatch, "match")
	dests := result.Payments[0].Destinations
	if !dests[0].IsEncrypted || dests[1].IsEncrypted || len(result.MatchedDestinations) != 1 {
		t.Fatalf("%+v", dests)
	}
}

// ---- Forms that do not verify

func TestOtherFunctionsAndTheAuthorityFormAreUnknownWithoutAGet(t *testing.T) {
	client := &mockClient{getFn: func(string) ([]Payment, error) {
		t.Fatal("must not GET")
		return nil, nil
	}}
	rail := railFor(t, UsdtBase)
	prefix := "ethereum:" + rail.contract + "@" + strconv.FormatInt(rail.chainID, 10)
	tail := "?address=" + swappedEvmAddr + "&uint256=1&branta_id=abc&branta_secret=the-secret"
	for _, qr := range []string{
		prefix + "/approve" + tail,
		prefix + "/transferFrom" + tail,
		prefix + "/transfer;x=1" + tail,
		"ethereum://" + rail.contract + "@" + strconv.FormatInt(rail.chainID, 10) + "/transfer" + tail,
	} {
		wantStatus(t, verifyQR(t, client, nil, qr), VerifyUnknown, qr)
	}
}

func TestUnparseableInputIsUnknownAndDoesNotFail(t *testing.T) {
	client := &mockClient{getFn: func(string) ([]Payment, error) {
		t.Fatal("must not GET")
		return nil, nil
	}}
	for _, qr := range []string{
		"ethereum://[bad/transfer",
		"bitcoin://[bad",
		"ethereum:0xc2132d05d31c914a87c6611c10748aeb04b58e8f@" + strings.Repeat("9", 5000) +
			"/transfer?address=" + swappedEvmAddr,
	} {
		wantStatus(t, verifyQR(t, client, nil, qr), VerifyUnknown, fmt.Sprintf("%.20s", qr))
	}
}

func TestRemovedSchemesAreUnknownWithoutAGetEvenWithValidParams(t *testing.T) {
	client := &mockClient{getFn: func(string) ([]Payment, error) {
		t.Fatal("must not GET")
		return nil, nil
	}}
	for _, scheme := range []string{"pol", "eth", "bsc", "tron", "ton", "sol"} {
		qr := scheme + ":" + usdtPolygonAddr + "?value=50&branta_id=enc-usdt&branta_secret=the-secret"
		wantStatus(t, verifyQR(t, client, nil, qr), VerifyUnknown, scheme)
	}
}

func TestBareChainAddressesAreUnknownWithoutAGet(t *testing.T) {
	client := &mockClient{getFn: func(string) ([]Payment, error) {
		t.Fatal("must not GET")
		return nil, nil
	}}
	for _, address := range []string{usdtPolygonAddr, usdtEthereumAddr, usdtTronAddr, usdtTonAddr, usdcSolanaAddr} {
		wantStatus(t, verifyQR(t, client, nil, address), VerifyUnknown, address)
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
