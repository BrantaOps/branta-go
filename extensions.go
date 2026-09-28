package branta

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"unicode"
)

// IsBolt11 reports whether value looks like a BOLT-11 invoice.
func IsBolt11(value string) bool {
	lower := strings.ToLower(value)
	return strings.HasPrefix(lower, "lnbc") || strings.HasPrefix(lower, "lntb") || strings.HasPrefix(lower, "lnbcrt")
}

// IsArk reports whether value looks like an Ark address.
func IsArk(value string) bool {
	return strings.HasPrefix(strings.ToLower(value), "ark1")
}

// IsSilentPayment reports whether value looks like a silent payment address.
func IsSilentPayment(value string) bool {
	lower := strings.ToLower(value)
	return strings.HasPrefix(lower, "sp1") || strings.HasPrefix(lower, "tsp1")
}

// HashZkType returns the hash-ZK DestinationType for value, or empty if it
// isn't one of the three hash-ZK types (bolt11, ark, silent payment).
//
// Intentionally returns empty for Bolt12 / LnUrl / LnAddress / TetherAddress —
// those are valid DestinationTypes but are NOT hash-ZK types in any sibling
// SDK. Do not "fix" this.
func HashZkType(value string) DestinationType {
	if IsBolt11(value) {
		return Bolt11
	}
	if IsArk(value) {
		return ArkAddress
	}
	if IsSilentPayment(value) {
		return SilentPayment
	}
	return ""
}

// ToNormalizedHash returns SHA-256(lowercase(value)) as 64-char uppercase hex,
// matching .NET's Convert.ToHexString. This value doubles as the AES secret
// for hash-ZK destinations, so a case mismatch here silently breaks cross-SDK
// payment lookups.
func ToNormalizedHash(value string) string {
	sum := sha256.Sum256([]byte(strings.ToLower(value)))
	return strings.ToUpper(hex.EncodeToString(sum[:]))
}

// ToURLFragment builds a `#k-{zk_id}={key}&k-{zk_id2}={key2}...` URL fragment
// from resolved decryption keys, iterating in insertion order.
func ToURLFragment(keys *orderedKeys) string {
	if keys == nil || len(keys.keys) == 0 {
		return "#"
	}
	parts := make([]string, 0, len(keys.keys))
	for _, k := range keys.keys {
		parts = append(parts, "k-"+k+"="+keys.m[k])
	}
	return "#" + strings.Join(parts, "&")
}

// orderedKeys is an insertion-ordered string map, matching sibling SDKs whose
// native map types preserve insertion order (IndexMap, LinkedHashMap, etc).
type orderedKeys struct {
	keys []string
	m    map[string]string
}

func newOrderedKeys() *orderedKeys {
	return &orderedKeys{m: make(map[string]string)}
}

func (o *orderedKeys) set(k, v string) {
	if _, exists := o.m[k]; !exists {
		o.keys = append(o.keys, k)
	}
	o.m[k] = v
}

func (o *orderedKeys) get(k string) (string, bool) {
	v, ok := o.m[k]
	return v, ok
}

func (o *orderedKeys) empty() bool {
	return o == nil || len(o.keys) == 0
}

func hasWhitespace(value string) bool {
	for _, r := range value {
		if unicode.IsSpace(r) {
			return true
		}
	}
	return false
}
