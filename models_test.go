package branta

import (
	"encoding/json"
	"errors"
	"testing"
)

func TestDestinationWireFormat(t *testing.T) {
	d := Destination{
		Value:        "abc",
		IsPrimary:    true,
		IsZk:         true,
		IsEncrypted:  true, // must not appear in the JSON
		Type:         BitcoinAddress,
		ZkID:         "zk-1",
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

func TestDestinationOptionalFieldsOmittedWhenEmpty(t *testing.T) {
	d := NewDestination("abc", "")
	raw, err := json.Marshal(d)
	if err != nil {
		t.Fatal(err)
	}
	var obj map[string]any
	if err := json.Unmarshal(raw, &obj); err != nil {
		t.Fatal(err)
	}
	if obj["value"] != "abc" || obj["primary"] != false || obj["zk"] != false {
		t.Fatalf("got %s", raw)
	}
	if _, ok := obj["type"]; ok {
		t.Fatal("type should be omitted")
	}
}

func TestDestinationDeserializeDefaults(t *testing.T) {
	var d Destination
	if err := json.Unmarshal([]byte(`{"value": "abc"}`), &d); err != nil {
		t.Fatal(err)
	}
	if d.IsPrimary || d.IsZk || d.IsEncrypted || d.Type != "" {
		t.Fatalf("%+v", d)
	}
}

func TestPaymentWireFormatFieldNames(t *testing.T) {
	payment := Payment{
		Description:               "desc",
		Destinations:              []Destination{NewDestination("abc", "")},
		CreatedAt:                 "2026-01-01T00:00:00Z",
		TTL:                       600,
		Metadata:                  "{}",
		Platform:                  "Acme",
		PlatformLogoURL:           "https://example.com/logo.png",
		PlatformLogoLightURL:      "https://example.com/logo-light.png",
		ParentPlatform:            &Platform{Name: "Parent"},
		ChildPlatform:             &Platform{Name: "Child"},
		BtcPayServerPluginVersion: "1.0.0",
		IsMetadataDecrypted:       true,
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
	if obj["btc_pay_server_plugin_version"] != "1.0.0" {
		t.Fatal("btc_pay_server_plugin_version")
	}
	child := obj["child_platform"].(map[string]any)
	if child["name"] != "Child" {
		t.Fatal("child name")
	}
	if _, ok := obj["is_metadata_decrypted"]; ok {
		t.Fatal("is_metadata_decrypted must not appear")
	}
	if _, ok := obj["parent_platform"]; ok {
		t.Fatal("parent_platform must not appear on serialize")
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

func TestDefaultValueErrorsOnNoDestinations(t *testing.T) {
	_, err := Payment{}.DefaultValue()
	if !errors.Is(err, ErrNoDestinations) {
		t.Fatalf("got %v", err)
	}
}

func TestDefaultValueReturnsFirstDestination(t *testing.T) {
	payment := Payment{
		Destinations: []Destination{
			NewDestination("first", ""),
			NewDestination("second", ""),
		},
	}
	got, err := payment.DefaultValue()
	if err != nil || got != "first" {
		t.Fatalf("got %q %v", got, err)
	}
}
