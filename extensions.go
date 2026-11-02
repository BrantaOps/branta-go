package branta

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
)

// IsBolt11 reports whether value looks like a BOLT-11 invoice.
func IsBolt11(value string) bool {
	lower := strings.ToLower(value)
	return strings.HasPrefix(lower, "lnbc") || strings.HasPrefix(lower, "lntb") || strings.HasPrefix(lower, "lnbcrt")
}

// IsArk reports whether value looks like an Ark address (mainnet ark1, signet/regtest tark1).
func IsArk(value string) bool {
	lower := strings.ToLower(value)
	return strings.HasPrefix(lower, "ark1") || strings.HasPrefix(lower, "tark1")
}

// IsSilentPayment reports whether value looks like a silent payment address.
func IsSilentPayment(value string) bool {
	lower := strings.ToLower(value)
	return strings.HasPrefix(lower, "sp1") || strings.HasPrefix(lower, "tsp1")
}

// DerivedKeyType returns bolt11, Ark, or silent payment for value, or empty.
// A bolt11, Ark address, or silent payment is encrypted with a key derived
// from the value. Other BIP21 extras are empty. They are not destination types.
func DerivedKeyType(value string) DestinationType {
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
// matching .NET's Convert.ToHexString. This is the AES secret for a derived-key
// destination (bolt11, Ark, or silent payment). A case mismatch here breaks
// cross-SDK payment lookups.
func ToNormalizedHash(value string) string {
	sum := sha256.Sum256([]byte(strings.ToLower(value)))
	return strings.ToUpper(hex.EncodeToString(sum[:]))
}

// ToURLFragment builds a `#k-{id}={key}&...` fragment from resolved decryption
// keys, in insertion order. Names and values are percent-encoded with the
// encodeURIComponent reserved set. An empty map returns "".
func ToURLFragment(keys *orderedKeys) string {
	if keys == nil || len(keys.keys) == 0 {
		return ""
	}
	parts := make([]string, 0, len(keys.keys))
	for _, k := range keys.keys {
		parts = append(parts, "k-"+fragmentEscape(k)+"="+fragmentEscape(keys.m[k]))
	}
	return "#" + strings.Join(parts, "&")
}

// fragmentEscape matches JavaScript encodeURIComponent: everything except
// ALPHA / DIGIT / "-" / "_" / "." / "!" / "~" / "*" / "'" / "(" / ")".
func fragmentEscape(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	for i := 0; i < len(s); i++ {
		c := s[i]
		if isFragmentSafe(c) {
			b.WriteByte(c)
			continue
		}
		fmt.Fprintf(&b, "%%%02X", c)
	}
	return b.String()
}

func isFragmentSafe(c byte) bool {
	if (c >= 'A' && c <= 'Z') || (c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') {
		return true
	}
	switch c {
	case '-', '_', '.', '!', '~', '*', '\'', '(', ')':
		return true
	}
	return false
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
