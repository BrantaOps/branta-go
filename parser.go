package branta

import (
	"net/url"
	"strings"
)

// QRDestination is a single destination parsed from QR text.
type QRDestination struct {
	Value string
	// Type is set when the scheme or payload names a rail (e.g. bitcoin).
	// Not set for pol: — token comes from the matched payment destination.
	Type DestinationType
	// SecretKey is true for a visible address that participates in the
	// secret-key swap check (bitcoin path / bc / tb, or pol: address).
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
	// ExpectedSecretKeyTypes is the rail family verify may bind for this QR's
	// scheme (match-if-any). Set from the URI scheme — not from a token stamped
	// on the address.
	ExpectedSecretKeyTypes []DestinationType
	// HasUnverifiableInstruction is true when the QR carries a payable
	// instruction this SDK cannot bind to a decrypted destination. That is any
	// value other than a bitcoin address, a pol: address, or a derived key
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
	if schemeLower == "pol" {
		destValue := extractDestination(text)
		parser.Destinations = append(parser.Destinations, QRDestination{
			Value:     destValue,
			SecretKey: true,
		})
		parser.ExpectedSecretKeyTypes = append([]DestinationType(nil), PolygonEvmDestinationTypes...)
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
