package branta

import (
	"net/url"
	"strings"
)

// QRDestination is a single destination parsed from QR text.
type QRDestination struct {
	Value string
	// Type is set when the scheme or payload names a rail: bitcoin, or an
	// ERC-681 transfer whose chain id and token contract name one.
	Type DestinationType
	// SecretKey is true for a visible address that participates in the
	// secret-key swap check (bitcoin path / bc / tb, or an ERC-681 deposit).
	SecretKey bool
}

// QRParser parses raw QR text into one or more payment destinations.
//
// Infallible by design — no sibling SDK ever raises a parse error; unrecognized
// input just yields a destination with an empty Type.
type QRParser struct {
	Destinations            []QRDestination
	OnChainEncryptionText   string
	OnChainEncryptionSecret string
	// ExpectedSecretKeyTypes are the destination types this scan may decrypt.
	// ERC-681 sets one type from the chain id and token contract. Bitcoin sets
	// BitcoinAddress.
	ExpectedSecretKeyTypes []DestinationType
	// HasUnverifiableInstruction is true when the QR carries a payable
	// instruction this SDK cannot bind to a decrypted destination. That is any
	// value other than a bitcoin address, an ERC-681 token transfer, or a derived key
	// (bolt11, Ark, or silent payment), including bolt12, lno, and pay.
	HasUnverifiableInstruction bool
}

// NewQRParser parses qrText.
func NewQRParser(qrText string) QRParser {
	text := normalizeQRText(qrText)
	parser := QRParser{}

	scheme, ok := uriScheme(text)
	if !ok {
		parser.Destinations = append(parser.Destinations, QRDestination{
			Value: text,
			Type:  detectPlainTextType(text),
		})
		return parser
	}

	schemeLower := strings.ToLower(scheme)
	if schemeLower == "ethereum" {
		destType, address, ok := parseERC681(text)
		if !ok {
			parser.Destinations = append(parser.Destinations, QRDestination{Value: text})
			return parser
		}
		parser.Destinations = append(parser.Destinations, QRDestination{
			Value:     address,
			Type:      destType,
			SecretKey: true,
		})
		parser.ExpectedSecretKeyTypes = []DestinationType{destType}
		query := parseQuery(extractQuery(text))
		parser.OnChainEncryptionText = query["branta_id"]
		parser.OnChainEncryptionSecret = query["branta_secret"]
		return parser
	}

	if schemeLower != "bitcoin" && schemeLower != "lightning" {
		parser.Destinations = append(parser.Destinations, QRDestination{
			Value: text,
			Type:  "",
		})
		return parser
	}

	destValue := extractDestination(text)
	var destType DestinationType
	if schemeLower == "bitcoin" {
		destType = BitcoinAddress
	} else {
		destType = lightningSchemeType(destValue)
	}
	parser.Destinations = append(parser.Destinations, QRDestination{
		Value:     destValue,
		Type:      destType,
		SecretKey: destType == BitcoinAddress,
	})
	if destValue != "" && destType != BitcoinAddress && DerivedKeyType(destValue) == "" {
		parser.HasUnverifiableInstruction = true
	}

	query := parseQuery(extractQuery(text))
	parser.OnChainEncryptionText = query["branta_id"]
	parser.OnChainEncryptionSecret = query["branta_secret"]

	addBindable := func(key string, onChain bool) {
		value, ok := query[key]
		if !ok {
			return
		}
		destType := detectPlainTextType(value)
		if onChain {
			destType = BitcoinAddress
		}
		if value != "" && !onChain && DerivedKeyType(value) == "" {
			parser.HasUnverifiableInstruction = true
			return
		}
		parser.Destinations = append(parser.Destinations, QRDestination{
			Value:     value,
			Type:      destType,
			SecretKey: onChain,
		})
	}

	addBindable("lightning", false)
	addBindable("ark", false)
	addBindable("silent_payment", false)
	addBindable("bc", true)
	addBindable("tb", true)
	addBindable("sp", false)
	if query["bolt12"] != "" {
		parser.HasUnverifiableInstruction = true
	}
	if query["lno"] != "" {
		parser.HasUnverifiableInstruction = true
	}
	if query["pay"] != "" {
		parser.HasUnverifiableInstruction = true
	}

	for _, d := range parser.Destinations {
		if d.SecretKey {
			parser.ExpectedSecretKeyTypes = []DestinationType{BitcoinAddress}
			break
		}
	}

	return parser
}

// Destination returns the first destination value, if any.
func (p QRParser) Destination() (string, bool) {
	if len(p.Destinations) == 0 {
		return "", false
	}
	return p.Destinations[0].Value, true
}

// DestinationType returns the first destination type, if any.
func (p QRParser) DestinationType() DestinationType {
	if len(p.Destinations) == 0 {
		return ""
	}
	return p.Destinations[0].Type
}

// IsOnChainZk is true iff both branta_id and branta_secret query params were present.
func (p QRParser) IsOnChainZk() bool {
	return p.OnChainEncryptionText != "" && p.OnChainEncryptionSecret != ""
}

// uriScheme returns the URI scheme if text starts with a syntactically valid
// `scheme:` prefix (RFC 3986 ALPHA *( ALPHA / DIGIT / "+" / "-" / "." ) ":").
func uriScheme(text string) (string, bool) {
	if text == "" || !isASCIIAlpha(text[0]) {
		return "", false
	}
	end := 1
	for end < len(text) {
		c := text[end]
		if isASCIIAlphanumeric(c) || c == '+' || c == '-' || c == '.' {
			end++
			continue
		}
		break
	}
	if end < len(text) && text[end] == ':' {
		return text[:end], true
	}
	return "", false
}

func extractDestination(text string) string {
	_, after, found := strings.Cut(text, ":")
	if !found {
		after = ""
	}
	if pos := strings.IndexByte(after, '?'); pos >= 0 {
		return after[:pos]
	}
	return after
}

func extractQuery(text string) string {
	// A "?" after the "#" belongs to the fragment, not the query.
	if hash := strings.IndexByte(text, '#'); hash >= 0 {
		text = text[:hash]
	}
	_, after, found := strings.Cut(text, ":")
	if !found {
		return ""
	}
	if pos := strings.IndexByte(after, '?'); pos >= 0 {
		return after[pos+1:]
	}
	return ""
}

func parseQuery(query string) map[string]string {
	m := make(map[string]string)
	for _, pair := range strings.Split(query, "&") {
		if pair == "" {
			continue
		}
		key, value, _ := strings.Cut(pair, "=")
		if key == "" {
			continue
		}
		decodedKey, err := url.PathUnescape(key)
		if err != nil {
			decodedKey = key
		}
		decodedValue, err := url.PathUnescape(value)
		if err != nil {
			decodedValue = value
		}
		lowered := strings.ToLower(decodedKey)
		if _, exists := m[lowered]; exists {
			continue
		}
		m[lowered] = decodedValue
	}
	return m
}

// normalizeQRText matches URL parsers: drop leading and trailing C0 controls
// and spaces, then drop tab, line feed, and carriage return anywhere. A control
// byte inside "bitcoin:" has to leave a typed address, or the swap check never runs.
func normalizeQRText(qrText string) string {
	value := strings.TrimSpace(qrText)
	start := 0
	end := len(value)
	for start < end && value[start] <= 0x20 {
		start++
	}
	for end > start && value[end-1] <= 0x20 {
		end--
	}
	value = value[start:end]
	value = strings.ReplaceAll(value, "\t", "")
	value = strings.ReplaceAll(value, "\n", "")
	return strings.ReplaceAll(value, "\r", "")
}

func lightningSchemeType(dest string) DestinationType {
	if IsBolt11(dest) {
		return Bolt11
	}
	return ""
}

func detectPlainTextType(value string) DestinationType {
	if IsBolt11(value) {
		return Bolt11
	}
	if IsArk(value) {
		return ArkAddress
	}
	if IsSilentPayment(value) {
		return SilentPayment
	}
	lower := strings.ToLower(value)
	if strings.HasPrefix(value, "1") || strings.HasPrefix(value, "3") ||
		strings.HasPrefix(lower, "bc1") || strings.HasPrefix(lower, "tb1") || strings.HasPrefix(lower, "bcrt1") {
		return BitcoinAddress
	}
	return ""
}

func isASCIIAlpha(c byte) bool {
	return (c >= 'A' && c <= 'Z') || (c >= 'a' && c <= 'z')
}

func isASCIIAlphanumeric(c byte) bool {
	return isASCIIAlpha(c) || (c >= '0' && c <= '9')
}

// erc681Path returns the path of an ethereum: URI, without scheme, query, or
// fragment. One leading slash is dropped.
func erc681Path(text string) string {
	_, path, _ := strings.Cut(text, ":")
	if cut := strings.IndexAny(path, "?#"); cut >= 0 {
		path = path[:cut]
	}
	return strings.TrimPrefix(path, "/")
}

func isASCIIDigit(c byte) bool { return c >= '0' && c <= '9' }

func isASCIIHex(c byte) bool {
	return isASCIIDigit(c) || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')
}

func allASCIIHex(s string) bool {
	for i := 0; i < len(s); i++ {
		if !isASCIIHex(s[i]) {
			return false
		}
	}
	return true
}

// isEVMAddress is "0x" or "0X" followed by 40 hex digits.
func isEVMAddress(value string) bool {
	return len(value) == 42 && value[0] == '0' && (value[1] == 'x' || value[1] == 'X') && allASCIIHex(value[2:])
}

// parseERC681 reads an ERC-681 token transfer. The chain id and contract select
// one destination type. The deposit address is the "address" query param.
func parseERC681(text string) (DestinationType, string, bool) {
	path := erc681Path(text)
	at := strings.IndexByte(path, '@')
	if at < 0 {
		return "", "", false
	}
	contract := path[:at]
	// The contract prefix is a lowercase "0x", then 40 hex digits.
	if len(contract) != 42 || contract[:2] != "0x" || !allASCIIHex(contract[2:]) {
		return "", "", false
	}
	const suffix = "/transfer"
	rest := path[at+1:]
	if len(rest) <= len(suffix) || !strings.HasSuffix(rest, suffix) {
		return "", "", false
	}
	chain := rest[:len(rest)-len(suffix)]
	if chain == "" {
		return "", "", false
	}
	for i := 0; i < len(chain); i++ {
		if !isASCIIDigit(chain[i]) {
			return "", "", false
		}
	}
	digits := strings.TrimLeft(chain, "0")
	// No supported chain id has more than 15 digits.
	if len(digits) > 15 {
		return "", "", false
	}
	var chainID int64
	for i := 0; i < len(digits); i++ {
		chainID = chainID*10 + int64(digits[i]-'0')
	}
	destType, ok := evmTokenType(chainID, contract)
	if !ok {
		return "", "", false
	}
	address, ok := parseQuery(extractQuery(text))["address"]
	if !ok || !isEVMAddress(address) {
		return "", "", false
	}
	return destType, address, true
}
