package domain

import (
	"time"

	"github.com/google/uuid"
)

// Partner represents a partner store
type Partner struct {
	ID         uuid.UUID
	Name       string
	APIKeyHash string
	WebhookURL *string
	IsActive   bool
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

// SupplierOrder represents an order from a partner
type SupplierOrder struct {
	ID                  uuid.UUID
	PartnerID           uuid.UUID
	PartnerOrderID      string
	Status              OrderStatus
	ShopifyDraftOrderID *int64
	CustomerName        string
	CustomerPhone       string
	ShippingAddress     map[string]interface{} // JSONB
	CartTotal           float64
	PaymentStatus       string
	RejectionReason     *string
	TrackingCarrier     *string
	TrackingNumber      *string
	TrackingURL         *string
	CreatedAt           time.Time
	UpdatedAt           time.Time
}

// SupplierOrderItem represents an item in a supplier order
type SupplierOrderItem struct {
	ID              uuid.UUID
	SupplierOrderID uuid.UUID
	SKU             string
	Title           string
	Price           float64
	Quantity        int
	ProductURL      *string
	IsSupplierItem  bool
	ShopifyVariantID *int64
	CreatedAt       time.Time
}

// IdempotencyKey stores idempotency information
type IdempotencyKey struct {
	Key             string
	PartnerID       uuid.UUID
	SupplierOrderID uuid.UUID
	RequestHash     string
	CreatedAt       time.Time
}

// SKUMapping maps SKUs to Shopify variants
type SKUMapping struct {
	ID              uuid.UUID
	SKU             string
	ShopifyProductID  int64
	ShopifyVariantID  int64
	IsActive        bool
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

// OrderEvent represents an audit event for an order
type OrderEvent struct {
	ID              uuid.UUID
	SupplierOrderID uuid.UUID
	EventType       string
	EventData       map[string]interface{} // JSONB
	CreatedAt       time.Time
}
