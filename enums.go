package branta

// BrantaEnvironment identifies which Branta server to talk to.
// The zero value means unset; URL() then uses staging.
type BrantaEnvironment int

const (
	envUnset BrantaEnvironment = iota
	Staging
	Production
	Localhost
)

// URL returns the base URL for this environment (no trailing slash).
func (e BrantaEnvironment) URL() string {
	switch e {
	case Production:
		return "https://guardrail.branta.pro"
	case Localhost:
		return "http://localhost:3000"
	default:
		return "https://staging.guardrail.branta.pro"
	}
}

// DestinationType is a destination AddPayment can encrypt.
// Wire format is snake_case and must match every other Branta SDK.
type DestinationType string

const (
	BitcoinAddress DestinationType = "bitcoin_address"
	Bolt11         DestinationType = "bolt11"
	ArkAddress     DestinationType = "ark_address"
	SilentPayment  DestinationType = "silent_payment"
	UsdtPolygon    DestinationType = "usdt_polygon"
	UsdcPolygon    DestinationType = "usdc_polygon"
	UsdtEthereum   DestinationType = "usdt_ethereum"
	UsdcEthereum   DestinationType = "usdc_ethereum"
	UsdtBsc        DestinationType = "usdt_bsc"
	UsdcBsc        DestinationType = "usdc_bsc"
	UsdtTron       DestinationType = "usdt_tron"
	UsdtTon        DestinationType = "usdt_ton"
	UsdtSolana     DestinationType = "usdt_solana"
	UsdcSolana     DestinationType = "usdc_solana"
)

// polygonEvmDestinationTypes are the Polygon EVM secret-key rails (pol: URI).
var polygonEvmDestinationTypes = []DestinationType{UsdtPolygon, UsdcPolygon}

// ethereumEvmDestinationTypes are the Ethereum secret-key rails (eth: URI).
var ethereumEvmDestinationTypes = []DestinationType{UsdtEthereum, UsdcEthereum}

// bscEvmDestinationTypes are the BSC secret-key rails (bsc: URI).
var bscEvmDestinationTypes = []DestinationType{UsdtBsc, UsdcBsc}

// tronDestinationTypes are the Tron secret-key rails (tron: URI).
var tronDestinationTypes = []DestinationType{UsdtTron}

// tonDestinationTypes are the TON secret-key rails (ton: URI).
var tonDestinationTypes = []DestinationType{UsdtTon}

// solanaDestinationTypes are the Solana secret-key rails (sol: URI).
var solanaDestinationTypes = []DestinationType{UsdtSolana, UsdcSolana}

// secretKeyURIFamilies maps a lowercase URI scheme to its secret-key
// destination family for verify (match-if-any). Token is the destination type
// on the payment, not the scheme. Unexported — consumers use DestinationType
// constants and Verify / AddPayment only.
var secretKeyURIFamilies = map[string][]DestinationType{
	"pol":  polygonEvmDestinationTypes,
	"eth":  ethereumEvmDestinationTypes,
	"bsc":  bscEvmDestinationTypes,
	"tron": tronDestinationTypes,
	"ton":  tonDestinationTypes,
	"sol":  solanaDestinationTypes,
}

// isSecretKeyAddressType reports whether type is encrypted with a random
// branta_secret (not a derived key).
func isSecretKeyAddressType(t DestinationType) bool {
	if t == BitcoinAddress {
		return true
	}
	for _, family := range secretKeyURIFamilies {
		for _, member := range family {
			if member == t {
				return true
			}
		}
	}
	return false
}

// VerifyStatus is the result of Verify.
type VerifyStatus string

const (
	VerifyMatch      VerifyStatus = "match"
	VerifyUnknown    VerifyStatus = "unknown"
	VerifyTampered   VerifyStatus = "tampered"
	VerifyIncomplete VerifyStatus = "incomplete"
)
