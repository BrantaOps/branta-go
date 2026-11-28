package branta

import (
	"encoding/json"
	"testing"
)

func TestDestinationWireFormat(t *testing.T) {
	d := Destination{
		Value:        "abc",
		IsPrimary:    true,
		IsEncrypted:  true,
		Type:         BitcoinAddress,
		ID:           "zk-1",
		EncryptedDEK: "dek",
	}
	raw, err := json.Marshal(d)
	if err != nil {
		t.Fatal(err)
	}
	var obj map[string]any
	if err := json.Unmarshal(raw, &obj); err != nil {
		t.Fatal(err)
	}
	if obj["value"] != "abc" || obj["primary"] != true || obj["zk"] != true {
		t.Fatalf("got %s", raw)
	}
	if obj["type"] != "bitcoin_address" || obj["zk_id"] != "zk-1" || obj["encrypted_dek"] != "dek" {
		t.Fatalf("got %s", raw)
	}
	if _, ok := obj["is_encrypted"]; ok {
		t.Fatal("is_encrypted must not appear")
	}
}

func TestDestinationMarshalAlwaysSendsZkTrue(t *testing.T) {
	raw, err := json.Marshal(NewDestination("abc", ""))
	if err != nil {
		t.Fatal(err)
	}
	var obj map[string]any
	if err := json.Unmarshal(raw, &obj); err != nil {
		t.Fatal(err)
	}
	if obj["zk"] != true {
		t.Fatalf("got %s", raw)
	}
	if _, ok := obj["type"]; ok {
		t.Fatal("type should be omitted")
	}
}

func TestDestinationZkFalseSkipsDecrypt(t *testing.T) {
	var d Destination
	if err := json.Unmarshal([]byte(`{"value":"abc","zk":false}`), &d); err != nil {
		t.Fatal(err)
	}
	if d.IsEncrypted {
		t.Fatal("zk false must not decrypt")
	}
}

func TestDestinationZkTrueAttemptsDecrypt(t *testing.T) {
	var d Destination
	if err := json.Unmarshal([]byte(`{"value":"abc","zk":true,"zk_id":"zk-1"}`), &d); err != nil {
		t.Fatal(err)
	}
	if !d.IsEncrypted || d.ID != "zk-1" {
		t.Fatalf("%+v", d)
	}
}

func TestDestinationMissingZkSkipsDecrypt(t *testing.T) {
	var d Destination
	if err := json.Unmarshal([]byte(`{"value": "abc"}`), &d); err != nil {
		t.Fatal(err)
	}
	if d.IsPrimary || d.IsEncrypted || d.Type != "" {
		t.Fatalf("%+v", d)
	}
}

func TestPaymentWireOmitsDescriptionAndParent(t *testing.T) {
	payment := Payment{
		Description:          "desc",
		Destinations:         []Destination{NewDestination("abc", "")},
		CreatedAt:            "2026-01-01T00:00:00Z",
		TTL:                  600,
		Metadata:             "{}",
		Platform:             "Acme",
		PlatformLogoURL:      "https://example.com/logo.png",
		PlatformLogoLightURL: "https://example.com/logo-light.png",
		ParentPlatform:       &Platform{Name: "Parent"},
		Brand:                &Platform{Name: "Child"},
		IsMetadataDecrypted:  true,
	}
	raw, err := json.Marshal(payment)
	if err != nil {
		t.Fatal(err)
	}
	var obj map[string]any
	if err := json.Unmarshal(raw, &obj); err != nil {
		t.Fatal(err)
	}
	if obj["created_at"] != "2026-01-01T00:00:00Z" {
		t.Fatal("created_at")
	}
	if obj["ttl"] != float64(600) {
		t.Fatalf("ttl: %s", raw)
	}
	if _, ok := obj["description"]; ok {
		t.Fatal("description must not appear")
	}
	if _, ok := obj["btc_pay_server_plugin_version"]; ok {
		t.Fatal("btc pay field must not appear")
	}
	brand := obj["brand"].(map[string]any)
	if brand["name"] != "Child" {
		t.Fatal("brand name")
	}
	if _, ok := obj["child_platform"]; ok {
		t.Fatal("child_platform must not appear")
	}
	dest := obj["destinations"].([]any)[0].(map[string]any)
	if dest["zk"] != true {
		t.Fatal("destination zk")
	}
	if _, ok := obj["is_metadata_decrypted"]; ok {
		t.Fatal("is_metadata_decrypted must not appear")
	}
	if _, ok := obj["parent_platform"]; ok {
		t.Fatal("parent_platform must not appear on serialize")
	}
}

func TestPaymentOmitsUnsetTTL(t *testing.T) {
	raw, err := json.Marshal(Payment{Destinations: []Destination{NewDestination("abc", BitcoinAddress)}})
	if err != nil {
		t.Fatal(err)
	}
	var obj map[string]any
	if err := json.Unmarshal(raw, &obj); err != nil {
		t.Fatal(err)
	}
	if _, ok := obj["ttl"]; ok {
		t.Fatalf("unset ttl must be omitted, got %s", raw)
	}
}

func TestPaymentParentPlatformRoundTripsOnDeserialize(t *testing.T) {
	var payment Payment
	if err := json.Unmarshal([]byte(`{"destinations": [], "parent_platform": {"name": "Parent"}}`), &payment); err != nil {
		t.Fatal(err)
	}
	if payment.ParentPlatform == nil || payment.ParentPlatform.Name != "Parent" {
		t.Fatalf("%+v", payment.ParentPlatform)
	}
}

func wireObject(t *testing.T, payment Payment) map[string]any {
	t.Helper()
	raw, err := json.Marshal(payment)
	if err != nil {
		t.Fatal(err)
	}
	var obj map[string]any
	if err := json.Unmarshal(raw, &obj); err != nil {
		t.Fatal(err)
	}
	return obj
}

func TestDeprecatedChildPlatformFieldSerializesAsBrand(t *testing.T) {
	obj := wireObject(t, Payment{ChildPlatform: &Platform{Name: "Old", LogoURL: "https://example.com/l.png"}})
	brand, ok := obj["brand"].(map[string]any)
	if !ok || brand["name"] != "Old" || brand["logo_url"] != "https://example.com/l.png" {
		t.Fatalf("got %v", obj)
	}
	if _, ok := brand["logo_light_url"]; ok {
		t.Fatal("empty logo_light_url must be omitted")
	}
	if _, ok := obj["child_platform"]; ok {
		t.Fatal("child_platform must never be sent")
	}
}

func TestBrandWinsOverChildPlatformOnSerialize(t *testing.T) {
	obj := wireObject(t, Payment{
		Brand:         &Platform{Name: "New"},
		ChildPlatform: &Platform{Name: "Old"},
	})
	if obj["brand"].(map[string]any)["name"] != "New" {
		t.Fatalf("got %v", obj)
	}
	if _, ok := obj["child_platform"]; ok {
		t.Fatal("child_platform must never be sent")
	}
}

func TestPaymentWithoutBrandOmitsBrandKeys(t *testing.T) {
	obj := wireObject(t, Payment{})
	if _, ok := obj["brand"]; ok {
		t.Fatalf("got %v", obj)
	}
	if _, ok := obj["child_platform"]; ok {
		t.Fatalf("got %v", obj)
	}
}

func TestPaymentParsesBrand(t *testing.T) {
	var payment Payment
	if err := json.Unmarshal([]byte(`{"destinations": [], "brand": {"name": "B", "logo_url": "u"}}`), &payment); err != nil {
		t.Fatal(err)
	}
	if payment.Brand == nil || payment.Brand.Name != "B" || payment.Brand.LogoURL != "u" {
		t.Fatalf("%+v", payment.Brand)
	}
	if payment.ChildPlatform == nil || *payment.ChildPlatform != *payment.Brand {
		t.Fatalf("deprecated field must mirror brand: %+v", payment.ChildPlatform)
	}
	if payment.ChildPlatform == payment.Brand {
		t.Fatal("mirror must be an independent copy")
	}
}

func TestPaymentParsesChildPlatformAlias(t *testing.T) {
	var payment Payment
	if err := json.Unmarshal([]byte(`{"destinations": [], "child_platform": {"name": "C", "logo_light_url": "ul"}}`), &payment); err != nil {
		t.Fatal(err)
	}
	if payment.Brand == nil || payment.Brand.Name != "C" || payment.Brand.LogoLightURL != "ul" {
		t.Fatalf("%+v", payment.Brand)
	}
	if payment.ChildPlatform == nil || *payment.ChildPlatform != *payment.Brand {
		t.Fatalf("%+v", payment.ChildPlatform)
	}
}

func TestPaymentParsesBothBrandWins(t *testing.T) {
	var payment Payment
	if err := json.Unmarshal([]byte(`{"destinations": [], "brand": {"name": "B"}, "child_platform": {"name": "C"}}`), &payment); err != nil {
		t.Fatal(err)
	}
	if payment.Brand.Name != "B" || payment.ChildPlatform.Name != "B" {
		t.Fatalf("%+v %+v", payment.Brand, payment.ChildPlatform)
	}
}

func TestPaymentParsesNeitherBrand(t *testing.T) {
	var payment Payment
	if err := json.Unmarshal([]byte(`{"destinations": [], "platform": "P"}`), &payment); err != nil {
		t.Fatal(err)
	}
	if payment.Brand != nil || payment.ChildPlatform != nil || payment.Platform != "P" {
		t.Fatalf("%+v", payment)
	}
}

func TestPaymentUnmarshalRejectsMalformedJSON(t *testing.T) {
	var payment Payment
	if err := json.Unmarshal([]byte(`{"destinations": "nope"}`), &payment); err == nil {
		t.Fatal("expected error")
	}
}
