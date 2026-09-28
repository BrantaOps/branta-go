package branta

import "testing"

func TestIsBolt11CaseInsensitivePrefixes(t *testing.T) {
	for _, v := range []string{"lnbc1...", "LNBC1...", "lntb1...", "LNTB1...", "lnbcrt1...", "LNBCRT1..."} {
		if !IsBolt11(v) {
			t.Errorf("expected bolt11: %s", v)
		}
	}
	if IsBolt11("bc1qsomething") {
		t.Fatal("bc1 should not be bolt11")
	}
}

func TestIsArkCaseInsensitivePrefix(t *testing.T) {
	if !IsArk("ark1qsomething") || !IsArk("ARK1QSOMETHING") {
		t.Fatal("ark")
	}
	if IsArk("bc1qsomething") {
		t.Fatal("not ark")
	}
}

func TestIsSilentPaymentCaseInsensitivePrefixes(t *testing.T) {
	for _, v := range []string{"sp1qsomething", "SP1QSOMETHING", "tsp1qsomething", "TSP1QSOMETHING"} {
		if !IsSilentPayment(v) {
			t.Errorf("expected silent payment: %s", v)
		}
	}
	if IsSilentPayment("bc1qsomething") {
		t.Fatal("not silent payment")
	}
}

func TestHashZkTypeOnlyBolt11ArkSilentPayment(t *testing.T) {
	if HashZkType("lnbc1...") != Bolt11 {
		t.Fatal("bolt11")
	}
	if HashZkType("ark1...") != ArkAddress {
		t.Fatal("ark")
	}
	if HashZkType("sp1...") != SilentPayment {
		t.Fatal("sp")
	}
}

func TestHashZkTypeExcludesNonHashZkTypes(t *testing.T) {
	for _, v := range []string{"lno1...", "LNURL1...", "user@example.com", "0x00000000000000000000000000000000000000", "1A1zP1eP5QGefi2DMPTfTL5SLmv7DivfNa"} {
		if got := HashZkType(v); got != "" {
			t.Errorf("%s should not be hash-ZK, got %s", v, got)
		}
	}
}

func TestToNormalizedHashIsUppercaseHexOfLowercasedValue(t *testing.T) {
	hash := ToNormalizedHash("test")
	if hash != "9F86D081884C7D659A2FEAA0C55AD015A3BF4F1B2B0B822CD15D6C15B0F00A08" {
		t.Fatalf("got %s", hash)
	}
	if len(hash) != 64 {
		t.Fatalf("len %d", len(hash))
	}
}

func TestToNormalizedHashIsCaseInsensitiveOnInput(t *testing.T) {
	if ToNormalizedHash("TEST") != ToNormalizedHash("test") || ToNormalizedHash("TeSt") != ToNormalizedHash("test") {
		t.Fatal("case")
	}
}

func TestToURLFragmentEmpty(t *testing.T) {
	if ToURLFragment(newOrderedKeys()) != "#" {
		t.Fatal("empty fragment")
	}
}

func TestToURLFragmentSingleKey(t *testing.T) {
	keys := newOrderedKeys()
	keys.set("zk-1", "secret-1")
	if ToURLFragment(keys) != "#k-zk-1=secret-1" {
		t.Fatalf("got %s", ToURLFragment(keys))
	}
}

func TestToURLFragmentMultipleKeysPreservesInsertionOrder(t *testing.T) {
	keys := newOrderedKeys()
	keys.set("zk-1", "secret-1")
	keys.set("zk-2", "secret-2")
	if ToURLFragment(keys) != "#k-zk-1=secret-1&k-zk-2=secret-2" {
		t.Fatalf("got %s", ToURLFragment(keys))
	}
}
