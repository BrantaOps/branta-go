package branta

import "testing"

func TestBitcoinURIWithoutQuery(t *testing.T) {
	p := NewQRParser("bitcoin:1A1zP1eP5QGefi2DMPTfTL5SLmv7DivfNa")
	dest, _ := p.Destination()
	if dest != "1A1zP1eP5QGefi2DMPTfTL5SLmv7DivfNa" || p.DestinationType() != BitcoinAddress {
		t.Fatalf("%+v", p)
	}
	if p.OnChainEncryptionText != "" || p.OnChainEncryptionSecret != "" {
		t.Fatal("zk params")
	}
}

func TestBitcoinURIWithBrantaZkParams(t *testing.T) {
	p := NewQRParser("bitcoin:1A1zP1eP5QGefi2DMPTfTL5SLmv7DivfNa?branta_id=abc%2Bdef%3D&branta_secret=1234")
	dest, _ := p.Destination()
	if dest != "1A1zP1eP5QGefi2DMPTfTL5SLmv7DivfNa" || p.DestinationType() != BitcoinAddress {
		t.Fatal("dest")
	}
	if p.OnChainEncryptionText != "abc+def=" || p.OnChainEncryptionSecret != "1234" || !p.IsOnChainZk() {
		t.Fatalf("text=%q secret=%q", p.OnChainEncryptionText, p.OnChainEncryptionSecret)
	}
}

func TestBitcoinURIWithLightningQueryParamPercentDecoded(t *testing.T) {
	p := NewQRParser("bitcoin:1A1zP1eP5QGefi2DMPTfTL5SLmv7DivfNa?lightning=lnbc100n1ptest%3Dpadded")
	if len(p.Destinations) != 2 {
		t.Fatalf("len %d", len(p.Destinations))
	}
	if p.Destinations[1].Value != "lnbc100n1ptest=padded" || p.Destinations[1].Type != Bolt11 {
		t.Fatalf("%+v", p.Destinations[1])
	}
}

func TestPlainBitcoinAddress(t *testing.T) {
	p := NewQRParser("1A1zP1eP5QGefi2DMPTfTL5SLmv7DivfNa")
	dest, _ := p.Destination()
	if dest != "1A1zP1eP5QGefi2DMPTfTL5SLmv7DivfNa" || p.DestinationType() != BitcoinAddress {
		t.Fatal("plain btc")
	}
}

func TestLightningURIBolt11(t *testing.T) {
	p := NewQRParser("lightning:lnbc100n1ptest")
	dest, _ := p.Destination()
	if dest != "lnbc100n1ptest" || p.DestinationType() != Bolt11 {
		t.Fatal("ln bolt11")
	}
}

func TestPlainBolt11(t *testing.T) {
	p := NewQRParser("lnbc100n1ptest")
	dest, _ := p.Destination()
	if dest != "lnbc100n1ptest" || p.DestinationType() != Bolt11 {
		t.Fatal("plain bolt11")
	}
}

func TestLightningURIBolt12IsUntyped(t *testing.T) {
	p := NewQRParser("lightning:lno1qcptest")
	dest, _ := p.Destination()
	if dest != "lno1qcptest" || p.DestinationType() != "" || !p.HasUnverifiableInstruction {
		t.Fatalf("%+v", p)
	}
}

func TestPlainBolt12HasNoType(t *testing.T) {
	p := NewQRParser("lno1qcptest")
	if p.DestinationType() != "" || p.HasUnverifiableInstruction {
		t.Fatalf("%+v", p)
	}
}

func TestLightningURILNURLIsUntyped(t *testing.T) {
	p := NewQRParser("lightning:LNURL1DP68GURN8GHJ")
	dest, _ := p.Destination()
	if dest != "LNURL1DP68GURN8GHJ" || p.DestinationType() != "" || !p.HasUnverifiableInstruction {
		t.Fatalf("%+v", p)
	}
}

func TestPlainLNURLHasNoType(t *testing.T) {
	p := NewQRParser("LNURL1DP68GURN8GHJ")
	if p.DestinationType() != "" {
		t.Fatal("plain lnurl")
	}
}

func TestPlainEthereumAddressHasNoType(t *testing.T) {
	p := NewQRParser("0x742d35Cc6634C0532925a3b844Bc454e4438f44e")
	dest, _ := p.Destination()
	if dest != "0x742d35Cc6634C0532925a3b844Bc454e4438f44e" || p.DestinationType() != "" {
		t.Fatal("eth")
	}
}

func TestPlainTronAddressHasNoType(t *testing.T) {
	p := NewQRParser("TJmUNSGV6b1CCVXN1KkABY49nUJGWDH3Hd")
	if p.DestinationType() != "" {
		t.Fatal("tron")
	}
}

func TestPlainArkAddress(t *testing.T) {
	p := NewQRParser("ark1qqjqtest")
	if p.DestinationType() != ArkAddress {
		t.Fatal("ark")
	}
}

func TestPlainSignetArkAddress(t *testing.T) {
	p := NewQRParser("tark1qqjqtest")
	if p.DestinationType() != ArkAddress {
		t.Fatal("tark")
	}
}

func TestPlainSilentPaymentSp1(t *testing.T) {
	p := NewQRParser("sp1qqwl5p9jhz0000h5zkvlf9gfqv9dl9qjp5ggq5x3fw")
	if p.DestinationType() != SilentPayment {
		t.Fatal("sp1")
	}
}

func TestPlainSilentPaymentTsp1(t *testing.T) {
	p := NewQRParser("tsp1qqwl5p9jhz0000h5zkvlf9gfqv9dl9qjp5ggq5x3fw")
	if p.DestinationType() != SilentPayment {
		t.Fatal("tsp1")
	}
}

func TestLnAddressHasNoType(t *testing.T) {
	p := NewQRParser("user@example.com")
	if p.DestinationType() != "" {
		t.Fatal("ln address")
	}
}

func TestTestnetAndRegtestAddressesAreBitcoin(t *testing.T) {
	if NewQRParser("tb1qshra2mlujc9wfscvn4d8aqpyqgxzl55853xg7s").DestinationType() != BitcoinAddress {
		t.Fatal("tb1")
	}
	if NewQRParser("bcrt1qexample").DestinationType() != BitcoinAddress {
		t.Fatal("bcrt1")
	}
}

func TestUnrecognizedPlainTextHasNoType(t *testing.T) {
	p := NewQRParser("not-any-known-format")
	dest, _ := p.Destination()
	if dest != "not-any-known-format" || p.DestinationType() != "" {
		t.Fatal("untyped")
	}
}

func TestWhitespaceIsTrimmed(t *testing.T) {
	p := NewQRParser("  lnbc100n1ptest  ")
	dest, _ := p.Destination()
	if dest != "lnbc100n1ptest" || p.DestinationType() != Bolt11 {
		t.Fatal("trim")
	}
}

func TestCombinedBitcoinAndLightningQRWithEmptyFirstQuerySegment(t *testing.T) {
	p := NewQRParser("bitcoin:1A1zP1eP5QGefi2DMPTfTL5SLmv7DivfNa?&lightning=lnbc100n1ptest")
	if len(p.Destinations) != 2 {
		t.Fatalf("len %d", len(p.Destinations))
	}
	dest, _ := p.Destination()
	if dest != "1A1zP1eP5QGefi2DMPTfTL5SLmv7DivfNa" || p.DestinationType() != BitcoinAddress {
		t.Fatal("primary")
	}
	if p.Destinations[1].Value != "lnbc100n1ptest" || p.Destinations[1].Type != Bolt11 {
		t.Fatal("lightning")
	}
	if p.IsOnChainZk() {
		t.Fatal("not zk")
	}
}

func TestCombinedBitcoinLightningAndArkQR(t *testing.T) {
	p := NewQRParser("bitcoin:1A1zP1eP5QGefi2DMPTfTL5SLmv7DivfNa?&lightning=lnbc100n1ptest&ark=ark100testaddress")
	if len(p.Destinations) != 3 {
		t.Fatalf("len %d", len(p.Destinations))
	}
	if p.Destinations[1].Value != "lnbc100n1ptest" || p.Destinations[1].Type != Bolt11 {
		t.Fatal("ln")
	}
	if p.Destinations[2].Value != "ark100testaddress" || p.Destinations[2].Type != ArkAddress {
		t.Fatal("ark")
	}
}

func TestCombinedBitcoinAndSilentPaymentQR(t *testing.T) {
	p := NewQRParser("bitcoin:1A1zP1eP5QGefi2DMPTfTL5SLmv7DivfNa?silent_payment=sp1qqwl5p9jhz0000h5zkvlf9gfqv9dl9qjp5ggq5x3fw")
	if len(p.Destinations) != 2 {
		t.Fatalf("len %d", len(p.Destinations))
	}
	if p.Destinations[1].Type != SilentPayment {
		t.Fatal("sp")
	}
}

func TestOtherURISchemeFallsBackToWholeTextUntyped(t *testing.T) {
	p := NewQRParser("http://example.com/pay?amount=1")
	dest, _ := p.Destination()
	if dest != "http://example.com/pay?amount=1" || p.DestinationType() != "" {
		t.Fatal("http fallback")
	}
}

func TestEmptyInputDoesNotPanic(t *testing.T) {
	p := NewQRParser("")
	if p.DestinationType() != "" || p.IsOnChainZk() {
		t.Fatal("empty")
	}
}

func TestBip321BcAndTbAreOnChainAddresses(t *testing.T) {
	p := NewQRParser("bitcoin:?bc=bc1qmerchant&tb=tb1qtest")
	if len(p.Destinations) != 3 {
		t.Fatalf("len %d", len(p.Destinations))
	}
	if p.Destinations[0].Value != "" || p.Destinations[1].Value != "bc1qmerchant" || p.Destinations[2].Value != "tb1qtest" {
		t.Fatalf("%+v", p.Destinations)
	}
	if p.Destinations[1].Type != BitcoinAddress || p.Destinations[2].Type != BitcoinAddress {
		t.Fatalf("%+v", p.Destinations)
	}
	if p.HasUnverifiableInstruction {
		t.Fatal("bc/tb are verifiable")
	}
}

func TestSpParamAddsSilentPayment(t *testing.T) {
	p := NewQRParser("bitcoin:bc1qmerchant?sp=sp1qqexample")
	if len(p.Destinations) != 2 || p.Destinations[1].Value != "sp1qqexample" || p.Destinations[1].Type != SilentPayment {
		t.Fatalf("%+v", p.Destinations)
	}
	if p.HasUnverifiableInstruction {
		t.Fatal("sp is hash-zk")
	}
}

func TestLnoAndPayAreUnverifiable(t *testing.T) {
	lno := NewQRParser("bitcoin:bc1qmerchant?lno=lno1qqexample")
	if len(lno.Destinations) != 1 || lno.Destinations[0].Value != "bc1qmerchant" {
		t.Fatalf("%+v", lno.Destinations)
	}
	if !lno.HasUnverifiableInstruction {
		t.Fatal("lno")
	}

	pay := NewQRParser("bitcoin:bc1qmerchant?pay=bitcoin:bc1qother")
	if !pay.HasUnverifiableInstruction {
		t.Fatal("pay")
	}
	if len(pay.Destinations) != 1 || pay.Destinations[0].Value != "bc1qmerchant" {
		t.Fatalf("%+v", pay.Destinations)
	}
}

func TestPolUriIsSecretKeyWithoutTokenType(t *testing.T) {
	p := NewQRParser("pol:0x278297528c32eCa01295B0d2135984d3442ec963?value=50&branta_id=enc&branta_secret=sec")
	dest, ok := p.Destination()
	if !ok || dest != "0x278297528c32eCa01295B0d2135984d3442ec963" {
		t.Fatal(dest)
	}
	if p.DestinationType() != "" || !p.Destinations[0].SecretKey {
		t.Fatalf("%+v", p.Destinations[0])
	}
	if p.OnChainEncryptionText != "enc" || p.OnChainEncryptionSecret != "sec" {
		t.Fatal(p.OnChainEncryptionText, p.OnChainEncryptionSecret)
	}
	if len(p.ExpectedSecretKeyTypes) != 2 || p.ExpectedSecretKeyTypes[0] != UsdtPolygon || p.ExpectedSecretKeyTypes[1] != UsdcPolygon {
		t.Fatalf("%v", p.ExpectedSecretKeyTypes)
	}
}

func TestBitcoinUriExpectsBitcoinSecretKeyFamily(t *testing.T) {
	p := NewQRParser("bitcoin:" + "bc1qmerchant" + "?branta_id=enc&branta_secret=sec")
	if !p.Destinations[0].SecretKey {
		t.Fatal("secret key")
	}
	if len(p.ExpectedSecretKeyTypes) != 1 || p.ExpectedSecretKeyTypes[0] != BitcoinAddress {
		t.Fatalf("%v", p.ExpectedSecretKeyTypes)
	}
}

func TestBolt12AndLnurlAreUnverifiable(t *testing.T) {
	for _, qr := range []string{
		"lightning:lno1qqexample",
		"lightning:lnurl1qqexample",
		"bitcoin:bc1qmerchant?bolt12=lno1qqexample",
	} {
		if !NewQRParser(qr).HasUnverifiableInstruction {
			t.Fatalf("expected unverifiable: %s", qr)
		}
	}
}

func TestControlCharInSchemeTypesBitcoinAddress(t *testing.T) {
	for _, prefix := range []string{"bitcoi\tn:", "bitcoi\nn:", "bitcoi\rn:", "\x01bitcoin:"} {
		p := NewQRParser(prefix + "1attackeraddr?branta_id=ID1&branta_secret=SEC1")
		dest, _ := p.Destination()
		if !p.IsOnChainZk() || dest != "1attackeraddr" || p.DestinationType() != BitcoinAddress {
			t.Fatalf("prefix %q: %+v", prefix, p)
		}
		if p.OnChainEncryptionText != "ID1" || p.OnChainEncryptionSecret != "SEC1" {
			t.Fatalf("prefix %q: text=%q secret=%q", prefix, p.OnChainEncryptionText, p.OnChainEncryptionSecret)
		}
	}
}

func TestTabInsideAddressUsesTheAddressAUrlParserWouldPay(t *testing.T) {
	p := NewQRParser("bitcoin:1attac\tkeraddr?branta_id=ID1&branta_secret=SEC1")
	dest, _ := p.Destination()
	if dest != "1attackeraddr" || p.DestinationType() != BitcoinAddress {
		t.Fatalf("%+v", p)
	}
}

func TestTabInsideLightningSchemeStillTypesBolt11(t *testing.T) {
	p := NewQRParser("lightnin\tg:lnbc100n1ptest")
	dest, _ := p.Destination()
	if dest != "lnbc100n1ptest" || p.DestinationType() != Bolt11 {
		t.Fatalf("%+v", p)
	}
}

func TestDuplicateQueryKeyKeepsTheFirstValue(t *testing.T) {
	p := NewQRParser("bitcoin:1attackeraddr?branta_id=ID1&branta_secret=SEC1&branta_secret=ATTACKER")
	if p.OnChainEncryptionText != "ID1" || p.OnChainEncryptionSecret != "SEC1" {
		t.Fatalf("text=%q secret=%q", p.OnChainEncryptionText, p.OnChainEncryptionSecret)
	}
}

func TestDerivedKeyAndBitcoinInstructionsAreVerifiable(t *testing.T) {
	for _, qr := range []string{
		"bitcoin:bc1qmerchant",
		"lightning:lnbc100n1ptest",
		"bitcoin:bc1qmerchant?lightning=lnbc100n1ptest&ark=ark1qq",
		"bitcoin:?bc=bc1qmerchant",
		"lnbc100n1ptest",
		"lno1plainTextNotUri",
	} {
		if NewQRParser(qr).HasUnverifiableInstruction {
			t.Fatalf("expected verifiable: %s", qr)
		}
	}
}

func TestChainSchemesMapToFamilies(t *testing.T) {
	cases := []struct {
		scheme  string
		address string
		want    []DestinationType
	}{
		{"eth", "0xBF5550253939187432Ef4dccE8089765d3DCb47f", []DestinationType{UsdtEthereum, UsdcEthereum}},
		{"bsc", "0xBF5550253939187432Ef4dccE8089765d3DCb47f", []DestinationType{UsdtBsc, UsdcBsc}},
		{"tron", "T9yD14Nj9j7xAB4dbGeiX9h8unkKHxuWwb", []DestinationType{UsdtTron}},
		{"ton", "UQBexample", []DestinationType{UsdtTon}},
		{"sol", "7EqQdEULxWcraVx3mXKFjc84LhCkMGZCkRuDofycyinf", []DestinationType{UsdtSolana, UsdcSolana}},
	}
	for _, tc := range cases {
		p := NewQRParser(tc.scheme + ":" + tc.address + "?value=1&branta_id=enc&branta_secret=sec")
		dest, ok := p.Destination()
		if !ok || dest != tc.address {
			t.Fatalf("%s: %q", tc.scheme, dest)
		}
		if p.DestinationType() != "" || !p.Destinations[0].SecretKey {
			t.Fatalf("%s: %+v", tc.scheme, p.Destinations[0])
		}
		if p.OnChainEncryptionText != "enc" || p.OnChainEncryptionSecret != "sec" {
			t.Fatalf("%s: %q %q", tc.scheme, p.OnChainEncryptionText, p.OnChainEncryptionSecret)
		}
		if len(p.ExpectedSecretKeyTypes) != len(tc.want) {
			t.Fatalf("%s: %v", tc.scheme, p.ExpectedSecretKeyTypes)
		}
		for i := range tc.want {
			if p.ExpectedSecretKeyTypes[i] != tc.want[i] {
				t.Fatalf("%s: %v", tc.scheme, p.ExpectedSecretKeyTypes)
			}
		}
	}
}
