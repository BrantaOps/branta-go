package branta

import (
	"net/url"
	"strings"
)

// QRDestination is a single destination parsed from QR text.
type QRDestination struct {
	Value string
	Type  DestinationType
}

// QRParser parses raw QR text into one or more payment destinations.
//
// Infallible by design — no sibling SDK ever raises a parse error; unrecognized
// input just yields a destination with an empty Type.
type QRParser struct {
	Destinations            []QRDestination
	OnChainEncryptionText   string
	OnChainEncryptionSecret string
	// HasUnverifiableInstruction is true when the QR carries a payable
	// instruction this SDK cannot bind to a decrypted destination (bolt12,
	// lno=, pay=, LNURL, or any other non-hash-ZK, non-bitcoin value).
	HasUnverifiableInstruction bool
}

// NewQRParser parses qrText.
func NewQRParser(qrText string) QRParser {
	text := strings.TrimSpace(qrText)
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
		Value: destValue,
		Type:  destType,
	})
	if destValue != "" && destType != BitcoinAddress && HashZkType(destValue) == "" {
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
		parser.Destinations = append(parser.Destinations, QRDestination{Value: value, Type: destType})
		if value != "" && !onChain && HashZkType(value) == "" {
			parser.HasUnverifiableInstruction = true
		}
	}

	addBindable("lightning", false)
	addBindable("bolt12", false)
	addBindable("ark", false)
	addBindable("silent_payment", false)
	addBindable("bc", true)
	addBindable("tb", true)
	addBindable("sp", false)
	addBindable("lno", false)
	if query["pay"] != "" {
		parser.HasUnverifiableInstruction = true
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
		m[strings.ToLower(decodedKey)] = decodedValue
	}
	return m
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
