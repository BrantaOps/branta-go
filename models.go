package branta

import "encoding/json"

// Destination is a payment destination exchanged with the Branta API.
//
// Wire fields are snake_case (`primary`, `zk`, `zk_id`). IsEncrypted is
// client-side only: true when a GET row has zk true and still needs decrypt.
type Destination struct {
	Value        string
	IsPrimary    bool
	IsEncrypted  bool
	Type         DestinationType
	ID           string
	EncryptedDEK string
}

// NewDestination constructs a destination with the given value and optional type.
func NewDestination(value string, destType DestinationType) Destination {
	return Destination{Value: value, Type: destType}
}

// MarshalJSON always sends zk true. AddPayment encrypts every destination.
func (d Destination) MarshalJSON() ([]byte, error) {
	wire := struct {
		Value        string          `json:"value"`
		IsPrimary    bool            `json:"primary"`
		Zk           bool            `json:"zk"`
		Type         DestinationType `json:"type,omitempty"`
		ID           string          `json:"zk_id,omitempty"`
		EncryptedDEK string          `json:"encrypted_dek,omitempty"`
	}{
		Value:        d.Value,
		IsPrimary:    d.IsPrimary,
		Zk:           true,
		Type:         d.Type,
		ID:           d.ID,
		EncryptedDEK: d.EncryptedDEK,
	}
	return json.Marshal(wire)
}

// UnmarshalJSON reads a GET or POST response destination.
// zk true attempts decrypt. A missing or false zk skips decrypt.
func (d *Destination) UnmarshalJSON(data []byte) error {
	var raw struct {
		Value        string          `json:"value"`
		IsPrimary    bool            `json:"primary"`
		Zk           *bool           `json:"zk"`
		Type         DestinationType `json:"type"`
		ID           string          `json:"zk_id"`
		EncryptedDEK string          `json:"encrypted_dek"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	d.Value = raw.Value
	d.IsPrimary = raw.IsPrimary
	d.Type = raw.Type
	d.ID = raw.ID
	d.EncryptedDEK = raw.EncryptedDEK
	d.IsEncrypted = raw.Zk != nil && *raw.Zk
	return nil
}

// Platform holds branding for a parent or child platform.
type Platform struct {
	Name         string `json:"name,omitempty"`
	LogoURL      string `json:"logo_url,omitempty"`
	LogoLightURL string `json:"logo_light_url,omitempty"`
}

// Payment is the wire model for a Branta payment.
//
// Description is not a plaintext API field. AddPayment folds it into encrypted
// metadata, and Verify lifts it back after metadata decrypts.
// ParentPlatform is deserialize-only: set by the server, never sent on a POST.
// TTL is seconds. Zero is unset and is left off the POST so Guardrail applies
// its default. A posted 0 is stored for 30 seconds.
type Payment struct {
	Description          string        `json:"-"`
	Destinations         []Destination `json:"destinations"`
	CreatedAt            string        `json:"created_at,omitempty"`
	TTL                  int           `json:"ttl,omitempty"`
	Metadata             string        `json:"metadata,omitempty"`
	Platform             string        `json:"platform,omitempty"`
	PlatformLogoURL      string        `json:"platform_logo_url,omitempty"`
	PlatformLogoLightURL string        `json:"platform_logo_light_url,omitempty"`
	ParentPlatform       *Platform     `json:"parent_platform,omitempty"`
	ChildPlatform        *Platform     `json:"child_platform,omitempty"`
	IsMetadataDecrypted  bool          `json:"-"`
}

// MarshalJSON omits description and parent_platform.
func (p Payment) MarshalJSON() ([]byte, error) {
	type paymentWire struct {
		Destinations         []Destination `json:"destinations"`
		CreatedAt            string        `json:"created_at,omitempty"`
		TTL                  int           `json:"ttl,omitempty"`
		Metadata             string        `json:"metadata,omitempty"`
		Platform             string        `json:"platform,omitempty"`
		PlatformLogoURL      string        `json:"platform_logo_url,omitempty"`
		PlatformLogoLightURL string        `json:"platform_logo_light_url,omitempty"`
		ChildPlatform        *Platform     `json:"child_platform,omitempty"`
	}
	dests := p.Destinations
	if dests == nil {
		dests = []Destination{}
	}
	return json.Marshal(paymentWire{
		Destinations:         dests,
		CreatedAt:            p.CreatedAt,
		TTL:                  p.TTL,
		Metadata:             p.Metadata,
		Platform:             p.Platform,
		PlatformLogoURL:      p.PlatformLogoURL,
		PlatformLogoLightURL: p.PlatformLogoLightURL,
		ChildPlatform:        p.ChildPlatform,
	})
}

// VerifyResult is what Verify returns. Payments and MatchedDestinations are
// set only for VerifyMatch. MatchedDestinations are the decrypted secret-key
// or derived-key rails that bound the scan (token for pol: is on Type).
type VerifyResult struct {
	Status              VerifyStatus
	Payments            []Payment
	MatchedDestinations []Destination
	VerifyURL           string
}

// AddPaymentResult is the result of AddPayment.
type AddPaymentResult struct {
	Payment   Payment
	Secret    string
	VerifyURL string
}
