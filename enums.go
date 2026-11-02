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
)

// VerifyStatus is the result of Verify.
type VerifyStatus string

const (
	VerifyMatch      VerifyStatus = "match"
	VerifyUnknown    VerifyStatus = "unknown"
	VerifyTampered   VerifyStatus = "tampered"
	VerifyIncomplete VerifyStatus = "incomplete"
)
