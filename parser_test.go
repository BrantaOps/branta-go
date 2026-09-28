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

func TestLightningURIBolt12(t *testing.T) {
	p := NewQRParser("lightning:lno1qcptest")
	dest, _ := p.Destination()
	if dest != "lno1qcptest" || p.DestinationType() != Bolt12 {
		t.Fatal("bolt12 uri")
	}
}

func TestPlainBolt12(t *testing.T) {
	p := NewQRParser("lno1qcptest")
	if p.DestinationType() != Bolt12 {
		t.Fatal("plain bolt12")
	}
}

func TestLightningURILNURL(t *testing.T) {
	p := NewQRParser("lightning:LNURL1DP68GURN8GHJ")
	dest, _ := p.Destination()
	if dest != "LNURL1DP68GURN8GHJ" || p.DestinationType() != LnUrl {
		t.Fatal("lnurl uri")
	}
}

func TestPlainLNURL(t *testing.T) {
	p := NewQRParser("LNURL1DP68GURN8GHJ")
	if p.DestinationType() != LnUrl {
		t.Fatal("plain lnurl")
	}
}

func TestPlainEthereumStyleTetherAddress(t *testing.T) {
	p := NewQRParser("0x742d35Cc6634C0532925a3b844Bc454e4438f44e")
	dest, _ := p.Destination()
	if dest != "0x742d35Cc6634C0532925a3b844Bc454e4438f44e" || p.DestinationType() != TetherAddress {
		t.Fatal("eth tether")
	}
}

func TestPlainTronStyleTetherAddress(t *testing.T) {
	p := NewQRParser("TJmUNSGV6b1CCVXN1KkABY49nUJGWDH3Hd")
	if p.DestinationType() != TetherAddress {
		t.Fatal("tron tether")
	}
}

func TestPlainArkAddress(t *testing.T) {
	p := NewQRParser("ark1qqjqtest")
	if p.DestinationType() != ArkAddress {
		t.Fatal("ark")
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

func TestLnAddressDetected(t *testing.T) {
	p := NewQRParser("user@example.com")
	if p.DestinationType() != LnAddress {
		t.Fatal("ln address")
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
