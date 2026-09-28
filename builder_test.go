package branta

import (
	"encoding/json"
	"testing"
)

func TestAddDestinationWithoutType(t *testing.T) {
	payment := NewPaymentBuilder().AddDestination("addr1", "").Build()
	if len(payment.Destinations) != 1 || payment.Destinations[0].Value != "addr1" {
		t.Fatalf("%+v", payment.Destinations)
	}
	if payment.Destinations[0].Type != "" || payment.Destinations[0].IsZk {
		t.Fatal("type/zk")
	}
}

func TestAddDestinationWithType(t *testing.T) {
	payment := NewPaymentBuilder().AddDestination("addr1", BitcoinAddress).Build()
	if payment.Destinations[0].Type != BitcoinAddress {
		t.Fatal(payment.Destinations[0].Type)
	}
}

func TestSetZkMarksOnlyTheLastAddedDestination(t *testing.T) {
	payment := NewPaymentBuilder().
		AddDestination("addr1", "").
		AddDestination("addr2", "").
		SetZk().
		Build()
	if payment.Destinations[0].IsZk || payment.Destinations[0].ZkID != "" {
		t.Fatal("first should not be zk")
	}
	if !payment.Destinations[1].IsZk || payment.Destinations[1].ZkID == "" {
		t.Fatal("second should be zk")
	}
}

func TestSetZkAssignsAFreshZkIDEachCall(t *testing.T) {
	payment := NewPaymentBuilder().
		AddDestination("addr1", "").SetZk().
		AddDestination("addr2", "").SetZk().
		Build()
	if payment.Destinations[0].ZkID == payment.Destinations[1].ZkID {
		t.Fatal("zk ids should differ")
	}
}

func TestSetDescription(t *testing.T) {
	payment := NewPaymentBuilder().SetDescription("desc").Build()
	if payment.Description != "desc" {
		t.Fatal(payment.Description)
	}
}

func TestAddMetadataMergesMultipleKeys(t *testing.T) {
	payment := NewPaymentBuilder().AddMetadata("a", "1").AddMetadata("b", "2").Build()
	var m map[string]string
	if err := json.Unmarshal([]byte(payment.Metadata), &m); err != nil {
		t.Fatal(err)
	}
	if m["a"] != "1" || m["b"] != "2" {
		t.Fatalf("%v", m)
	}
}

func TestAddMetadataOverwritesSameKeyNotDuplicates(t *testing.T) {
	payment := NewPaymentBuilder().AddMetadata("a", "1").AddMetadata("a", "2").Build()
	var m map[string]string
	if err := json.Unmarshal([]byte(payment.Metadata), &m); err != nil {
		t.Fatal(err)
	}
	if len(m) != 1 || m["a"] != "2" {
		t.Fatalf("%v", m)
	}
}

func TestSetTTL(t *testing.T) {
	payment := NewPaymentBuilder().SetTTL(600).Build()
	if payment.TTL != 600 {
		t.Fatal(payment.TTL)
	}
}

func TestSetPlatformLogoURL(t *testing.T) {
	payment := NewPaymentBuilder().SetPlatformLogoURL("https://example.com/logo.png").Build()
	if payment.PlatformLogoURL != "https://example.com/logo.png" {
		t.Fatal(payment.PlatformLogoURL)
	}
}

func TestSetChildPlatform(t *testing.T) {
	payment := NewPaymentBuilder().
		SetChildPlatform("ChildBrand", "https://example.com/logo.png", "https://example.com/logo-light.png").
		Build()
	child := payment.ChildPlatform
	if child == nil || child.Name != "ChildBrand" {
		t.Fatal("name")
	}
	if child.LogoURL != "https://example.com/logo.png" || child.LogoLightURL != "https://example.com/logo-light.png" {
		t.Fatal("logos")
	}
}

func TestFullChainBuildsExpectedPayment(t *testing.T) {
	payment := NewPaymentBuilder().
		SetDescription("Testing description").
		AddDestination("1A1zP1eP5QGefi2DMPTfTL5SLmv7DivfNa", BitcoinAddress).
		SetZk().
		SetTTL(600).
		Build()
	if payment.Description != "Testing description" || payment.TTL != 600 || !payment.Destinations[0].IsZk {
		t.Fatalf("%+v", payment)
	}
}
