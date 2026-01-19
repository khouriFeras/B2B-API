package service

// CartSubmitRequest represents the cart submission payload
type CartSubmitRequest struct {
	PartnerOrderID string                 `json:"partner_order_id" binding:"required"`
	Items          []CartItem             `json:"items" binding:"required,min=1"`
	Customer       CustomerInfo            `json:"customer" binding:"required"`
	Shipping       ShippingAddress         `json:"shipping" binding:"required"`
	Totals         CartTotals             `json:"totals" binding:"required"`
	PaymentStatus  string                 `json:"payment_status"`
}

type CartItem struct {
	SKU        string  `json:"sku" binding:"required"`
	Title      string  `json:"title" binding:"required"`
	Price      float64 `json:"price" binding:"required,min=0"`
	Quantity   int     `json:"quantity" binding:"required,min=1"`
	ProductURL *string `json:"product_url,omitempty"`
}

type CustomerInfo struct {
	Name  string  `json:"name" binding:"required"`
	Phone *string `json:"phone,omitempty"`
}

type ShippingAddress struct {
	Street     string  `json:"street" binding:"required"`
	City       string  `json:"city" binding:"required"`
	State      *string `json:"state,omitempty"`
	PostalCode string  `json:"postal_code" binding:"required"`
	Country    string  `json:"country" binding:"required"`
}

type CartTotals struct {
	Subtotal float64 `json:"subtotal" binding:"required,min=0"`
	Tax      float64 `json:"tax" binding:"min=0"`
	Shipping float64 `json:"shipping" binding:"min=0"`
	Total    float64 `json:"total" binding:"required,min=0"`
}