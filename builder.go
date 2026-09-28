package branta

import (
	"encoding/json"

	"github.com/google/uuid"
)

// PaymentBuilder is a fluent builder for constructing a Payment to post.
type PaymentBuilder struct {
	payment Payment
}

// NewPaymentBuilder returns an empty builder.
func NewPaymentBuilder() *PaymentBuilder {
	return &PaymentBuilder{payment: Payment{Destinations: []Destination{}}}
}

// AddDestination adds a new destination. Call SetZk immediately after to mark it as ZK.
// An empty destType leaves the destination untyped.
func (b *PaymentBuilder) AddDestination(address string, destType DestinationType) *PaymentBuilder {
	b.payment.Destinations = append(b.payment.Destinations, NewDestination(address, destType))
	return b
}

// SetZk marks the most-recently-added destination as ZK and assigns it a fresh zk_id.
func (b *PaymentBuilder) SetZk() *PaymentBuilder {
	if n := len(b.payment.Destinations); n > 0 {
		b.payment.Destinations[n-1].IsZk = true
		b.payment.Destinations[n-1].ZkID = uuid.New().String()
	}
	return b
}

// SetDescription sets the payment description.
func (b *PaymentBuilder) SetDescription(description string) *PaymentBuilder {
	b.payment.Description = description
	return b
}

// AddMetadata merges key/value into the payment's metadata, which is stored as
// a JSON-object string (i.e. double-encoded: metadata is itself the serialized
// form of a {key: value} map).
func (b *PaymentBuilder) AddMetadata(key, value string) *PaymentBuilder {
	metadataMap := map[string]string{}
	if b.payment.Metadata != "" {
		_ = json.Unmarshal([]byte(b.payment.Metadata), &metadataMap)
	}
	metadataMap[key] = value
	encoded, err := json.Marshal(metadataMap)
	if err != nil {
		return b
	}
	b.payment.Metadata = string(encoded)
	return b
}

// SetTTL sets the payment time-to-live in seconds.
func (b *PaymentBuilder) SetTTL(ttl int) *PaymentBuilder {
	b.payment.TTL = ttl
	return b
}

// SetPlatformLogoURL sets the platform logo URL.
func (b *PaymentBuilder) SetPlatformLogoURL(platformLogoURL string) *PaymentBuilder {
	b.payment.PlatformLogoURL = platformLogoURL
	return b
}

// SetChildPlatform tags the payment with child-platform branding.
func (b *PaymentBuilder) SetChildPlatform(name, logoURL, logoLightURL string) *PaymentBuilder {
	b.payment.ChildPlatform = &Platform{
		Name:         name,
		LogoURL:      logoURL,
		LogoLightURL: logoLightURL,
	}
	return b
}

// Build returns the constructed Payment.
func (b *PaymentBuilder) Build() Payment {
	return b.payment
}
