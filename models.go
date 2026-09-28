package branta

import "encoding/json"

// Destination is a payment destination exchanged with the Branta API.
//
// Field names on the wire are snake_case (with `primary`/`zk` rather than
// `is_primary`/`is_zk`) and must match every other Branta SDK exactly.
// IsEncrypted is client-side-only derived state and is never serialized.
type Destination struct {
	Value        string          `json:"value"`
	IsPrimary    bool            `json:"primary"`
	IsZk         bool            `json:"zk"`
	IsEncrypted  bool            `json:"-"`
	Type         DestinationType `json:"type,omitempty"`
	ZkID         string          `json:"zk_id,omitempty"`
	EncryptedDEK string          `json:"encrypted_dek,omitempty"`
}

// NewDestination constructs a destination with the given value and optional type.
func NewDestination(value string, destType DestinationType) Destination {
	return Destination{Value: value, Type: destType}
}

// Platform holds branding for a parent or child platform.
type Platform struct {
	Name         string `json:"name,omitempty"`
	LogoURL      string `json:"logo_url,omitempty"`
	LogoLightURL string `json:"logo_light_url,omitempty"`
}

// Payment is the wire model for a Branta payment.
//
// ParentPlatform is deserialize-only: set by the server, never sent on a POST.
// IsMetadataDecrypted is client-side-only and is never serialized.
//
// NOTE: the majority wire name across sibling SDKs (dotnet/js/python/dart/rust)
// is `btc_pay_server_plugin_version`. branta-kotlin alone uses
// `btcpay_server_plugin_version` — a pre-existing cross-SDK inconsistency, not
// something to replicate here.
type Payment struct {
	Description               string        `json:"description,omitempty"`
	Destinations              []Destination `json:"destinations"`
	CreatedAt                 string        `json:"created_at,omitempty"`
	TTL                       int           `json:"ttl"`
	Metadata                  string        `json:"metadata,omitempty"`
	Platform                  string        `json:"platform,omitempty"`
	PlatformLogoURL           string        `json:"platform_logo_url,omitempty"`
	PlatformLogoLightURL      string        `json:"platform_logo_light_url,omitempty"`
	ParentPlatform            *Platform     `json:"parent_platform,omitempty"`
	ChildPlatform             *Platform     `json:"child_platform,omitempty"`
	BtcPayServerPluginVersion string        `json:"btc_pay_server_plugin_version,omitempty"`
	IsMetadataDecrypted       bool          `json:"-"`
}

// MarshalJSON omits parent_platform (deserialize-only) and client-side flags.
func (p Payment) MarshalJSON() ([]byte, error) {
	type paymentWire struct {
		Description               string        `json:"description,omitempty"`
		Destinations              []Destination `json:"destinations"`
		CreatedAt                 string        `json:"created_at,omitempty"`
		TTL                       int           `json:"ttl"`
		Metadata                  string        `json:"metadata,omitempty"`
		Platform                  string        `json:"platform,omitempty"`
		PlatformLogoURL           string        `json:"platform_logo_url,omitempty"`
		PlatformLogoLightURL      string        `json:"platform_logo_light_url,omitempty"`
		ChildPlatform             *Platform     `json:"child_platform,omitempty"`
		BtcPayServerPluginVersion string        `json:"btc_pay_server_plugin_version,omitempty"`
	}
	dests := p.Destinations
	if dests == nil {
		dests = []Destination{}
	}
	return json.Marshal(paymentWire{
		Description:               p.Description,
		Destinations:              dests,
		CreatedAt:                 p.CreatedAt,
		TTL:                       p.TTL,
		Metadata:                  p.Metadata,
		Platform:                  p.Platform,
		PlatformLogoURL:           p.PlatformLogoURL,
		PlatformLogoLightURL:      p.PlatformLogoLightURL,
		ChildPlatform:             p.ChildPlatform,
		BtcPayServerPluginVersion: p.BtcPayServerPluginVersion,
	})
}

// DefaultValue returns the value of the first destination, or ErrNoDestinations.
func (p Payment) DefaultValue() (string, error) {
	if len(p.Destinations) == 0 {
		return "", ErrNoDestinations
	}
	return p.Destinations[0].Value, nil
}

// PaymentsResult is the result of GetPayments / GetPaymentsByQRCode.
//
// VerifyURL is always populated, even when Payments is empty.
type PaymentsResult struct {
	Payments  []Payment
	VerifyURL string
}

// AddPaymentResult is the result of AddPayment.
type AddPaymentResult struct {
	Payment   Payment
	Secret    string
	VerifyURL string
}
