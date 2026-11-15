package branta

import (
	"strconv"
	"strings"
	"testing"
)

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

const (
	baseUsdcContract = "0x833589fCD6eDb6E08f4c7C32D4f71b54bdA02913"
	erc681Deposit    = "0x1111111111111111111111111111111111111111"
)

func TestERC681TransferBindsTheDepositAndOneToken(t *testing.T) {
	p := NewQRParser("ethereum:" + baseUsdcContract + "@8453/transfer?address=" + erc681Deposit +
		"&uint256=1000000&branta_id=abc%2Bdef%3D&branta_secret=1234")
	dest, ok := p.Destination()
	if !ok || dest != erc681Deposit {
		t.Fatal(dest)
	}
	if p.DestinationType() != UsdcBase || len(p.Destinations) != 1 || !p.Destinations[0].SecretKey {
		t.Fatalf("%+v", p.Destinations)
	}
	if len(p.ExpectedSecretKeyTypes) != 1 || p.ExpectedSecretKeyTypes[0] != UsdcBase {
		t.Fatalf("%v", p.ExpectedSecretKeyTypes)
	}
	if p.OnChainEncryptionText != "abc+def=" || p.OnChainEncryptionSecret != "1234" {
		t.Fatal(p.OnChainEncryptionText, p.OnChainEncryptionSecret)
	}
}

func TestEveryRailBindsTheDepositAndOneType(t *testing.T) {
	for _, rail := range evmTokenRails {
		p := NewQRParser("ethereum:" + rail.contract + "@" + strconv.FormatInt(rail.chainID, 10) +
			"/transfer?address=" + erc681Deposit + "&branta_id=a&branta_secret=b")
		dest, _ := p.Destination()
		if dest != erc681Deposit || p.DestinationType() != rail.destType ||
			len(p.ExpectedSecretKeyTypes) != 1 || p.ExpectedSecretKeyTypes[0] != rail.destType {
			t.Fatalf("%s: %+v", rail.destType, p)
		}
	}
}

func TestERC681ContractMatchesCaseInsensitively(t *testing.T) {
	p := NewQRParser("ethereum:" + strings.ToLower(baseUsdcContract) + "@8453/transfer?address=" + erc681Deposit)
	if p.DestinationType() != UsdcBase {
		t.Fatalf("%+v", p)
	}
}

func TestERC681Uint256IsNotPartOfTheBinding(t *testing.T) {
	for _, amount := range []string{"1", "50000000", "999999999999999999999"} {
		p := NewQRParser("ethereum:" + baseUsdcContract + "@8453/transfer?address=" + erc681Deposit + "&uint256=" + amount)
		dest, _ := p.Destination()
		if dest != erc681Deposit || p.DestinationType() != UsdcBase {
			t.Fatalf("%s: %+v", amount, p)
		}
	}
}

func TestERC681ToleratesOneLeadingSlashAndAnUpperCaseScheme(t *testing.T) {
	for _, qr := range []string{
		"ethereum:/" + baseUsdcContract + "@8453/transfer?address=" + erc681Deposit,
		"ETHEREUM:" + baseUsdcContract + "@8453/transfer?address=" + erc681Deposit,
	} {
		if NewQRParser(qr).DestinationType() != UsdcBase {
			t.Fatal(qr)
		}
	}
}

func TestERC681AcceptsAnUpperCase0XDepositPrefix(t *testing.T) {
	p := NewQRParser("ethereum:" + baseUsdcContract + "@8453/transfer?address=0X" + erc681Deposit[2:])
	if p.DestinationType() != UsdcBase {
		t.Fatalf("%+v", p)
	}
}

func TestERC681FirstAddressParamWins(t *testing.T) {
	p := NewQRParser("ethereum:" + baseUsdcContract + "@8453/transfer?address=" + erc681Deposit +
		"&address=0x" + strings.Repeat("f", 40))
	if dest, _ := p.Destination(); dest != erc681Deposit {
		t.Fatal(dest)
	}
}

func TestERC681PercentEncodedAmpersandStaysInsideBrantaID(t *testing.T) {
	p := NewQRParser("ethereum:" + baseUsdcContract + "@8453/transfer?address=" + erc681Deposit +
		"&branta_id=a%26b&branta_secret=s")
	if p.OnChainEncryptionText != "a&b" || p.OnChainEncryptionSecret != "s" {
		t.Fatal(p.OnChainEncryptionText, p.OnChainEncryptionSecret)
	}
}

func TestERC681ParamsInTheFragmentAreNotRead(t *testing.T) {
	p := NewQRParser("ethereum:" + baseUsdcContract + "@8453/transfer?address=" + erc681Deposit +
		"#&branta_id=abc&branta_secret=1234")
	if p.DestinationType() != UsdcBase || p.OnChainEncryptionText != "" {
		t.Fatalf("%+v", p)
	}
}

func TestERC681QueryAfterAFragmentIsNotRead(t *testing.T) {
	p := NewQRParser("ethereum:" + baseUsdcContract + "@8453/transfer#x?address=" + erc681Deposit +
		"&branta_id=a&branta_secret=b")
	if p.DestinationType() != "" || p.OnChainEncryptionText != "" || len(p.ExpectedSecretKeyTypes) != 0 {
		t.Fatalf("%+v", p)
	}
}

func TestERC681FragmentAfterTheQueryIsDropped(t *testing.T) {
	p := NewQRParser("ethereum:" + baseUsdcContract + "@8453/transfer?address=" + erc681Deposit +
		"&branta_id=a&branta_secret=b#address=0x2")
	dest, _ := p.Destination()
	if dest != erc681Deposit || p.OnChainEncryptionText != "a" || p.OnChainEncryptionSecret != "b" {
		t.Fatalf("%+v", p)
	}
}

func TestBitcoinFragmentDoesNotBleedIntoTheQuery(t *testing.T) {
	if p := NewQRParser("bitcoin:bc1qtest#x?branta_id=a&branta_secret=b"); p.OnChainEncryptionText != "" {
		t.Fatalf("%+v", p)
	}
	if p := NewQRParser("bitcoin:bc1qtest?branta_id=a&branta_secret=b#frag"); p.OnChainEncryptionSecret != "b" {
		t.Fatalf("%+v", p)
	}
}

func TestERC681ChainIDBeyond15DigitsIsUntyped(t *testing.T) {
	p := NewQRParser("ethereum:" + baseUsdcContract + "@" + strings.Repeat("9", 5000) +
		"/transfer?address=" + erc681Deposit)
	if p.DestinationType() != "" {
		t.Fatal("expected untyped")
	}
}

func TestERC681LeadingZeroChainIDStillSelectsTheRail(t *testing.T) {
	p := NewQRParser("ethereum:" + baseUsdcContract + "@" + strings.Repeat("0", 4500) +
		"8453/transfer?address=" + erc681Deposit)
	if p.DestinationType() != UsdcBase {
		t.Fatalf("%+v", p)
	}
}

func TestIncompleteERC681FormsDoNotBind(t *testing.T) {
	tail := "?address=" + erc681Deposit + "&branta_id=abc&branta_secret=1234"
	forms := []string{
		// No chain id.
		"ethereum:" + baseUsdcContract + "/transfer" + tail,
		// Payment-request prefix and ENS-style forms.
		"ethereum:pay-" + baseUsdcContract + "@8453/transfer" + tail,
		"ethereum:vitalik.eth@8453/transfer" + tail,
		// // makes the contract an authority, not a path.
		"ethereum://" + baseUsdcContract + "@8453/transfer" + tail,
		// Only transfer is a payment. approve and transferFrom are not.
		"ethereum:" + baseUsdcContract + "@8453/approve" + tail,
		"ethereum:" + baseUsdcContract + "@8453/transferFrom" + tail,
		// Path parameters after the function name.
		"ethereum:" + baseUsdcContract + "@8453/transfer;x=1" + tail,
		// Chain id forms that are not plain digits.
		"ethereum:" + baseUsdcContract + "@0x2105/transfer" + tail,
		"ethereum:" + baseUsdcContract + "@-8453/transfer" + tail,
		"ethereum:" + baseUsdcContract + "@+8453/transfer" + tail,
		"ethereum:" + baseUsdcContract + "@/transfer" + tail,
		// No address, a truncated address, or an ENS recipient.
		"ethereum:" + baseUsdcContract + "@8453/transfer?branta_id=abc&branta_secret=1234",
		"ethereum:" + baseUsdcContract + "@8453/transfer?address=0x1234&branta_id=abc&branta_secret=1234",
		"ethereum:" + baseUsdcContract + "@8453/transfer?address=vitalik.eth&branta_id=abc&branta_secret=1234",
	}
	for _, qr := range forms {
		p := NewQRParser(qr)
		if p.DestinationType() != "" || len(p.ExpectedSecretKeyTypes) != 0 ||
			p.OnChainEncryptionText != "" || p.Destinations[0].SecretKey {
			t.Fatalf("%s: %+v", qr, p)
		}
	}
}

func TestUnknownContractAndWrongChainDoNotBind(t *testing.T) {
	for _, qr := range []string{
		"ethereum:0x" + strings.Repeat("2", 40) + "@8453/transfer?address=" + erc681Deposit + "&branta_id=a&branta_secret=b",
		"ethereum:" + baseUsdcContract + "@1/transfer?address=" + erc681Deposit + "&branta_id=a&branta_secret=b",
	} {
		p := NewQRParser(qr)
		if p.DestinationType() != "" || p.OnChainEncryptionText != "" {
			t.Fatalf("%s: %+v", qr, p)
		}
	}
}

func TestUnparseableAuthorityIsAnUntypedDestination(t *testing.T) {
	p := NewQRParser("ethereum://[bad/transfer")
	if dest, _ := p.Destination(); dest != "ethereum://[bad/transfer" || p.DestinationType() != "" {
		t.Fatalf("%+v", p)
	}
	// Other schemes never fail either. They carry no Branta params.
	if NewQRParser("bitcoin://[bad").OnChainEncryptionText != "" {
		t.Fatal("bitcoin")
	}
}

func TestRemovedChainSchemesDoNotBind(t *testing.T) {
	for _, scheme := range []string{"pol", "eth", "bsc", "tron", "ton", "sol"} {
		text := scheme + ":0x278297528c32eCa01295B0d2135984d3442ec963?value=50&branta_id=abc&branta_secret=1234"
		p := NewQRParser(text)
		dest, _ := p.Destination()
		if dest != text || p.DestinationType() != "" || p.Destinations[0].SecretKey ||
			len(p.ExpectedSecretKeyTypes) != 0 || p.IsOnChainZk() {
			t.Fatalf("%s: %+v", scheme, p)
		}
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
