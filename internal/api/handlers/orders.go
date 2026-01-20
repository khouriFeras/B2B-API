package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/jafarshop/b2bapi/internal/domain"
	"github.com/jafarshop/b2bapi/internal/repository"
	"github.com/jafarshop/b2bapi/internal/api/middleware"
	"github.com/jafarshop/b2bapi/pkg/errors"
)

// OrderResponse represents the order response
type OrderResponse struct {
	ID                  string                 `json:"id"`
	PartnerOrderID      string                 `json:"partner_order_id"`
	Status              domain.OrderStatus     `json:"status"`
	ShopifyDraftOrderID *int64                 `json:"shopify_draft_order_id,omitempty"`
	CustomerName        string                 `json:"customer_name"`
	CustomerPhone       string                 `json:"customer_phone,omitempty"`
	ShippingAddress     map[string]interface{} `json:"shipping_address"`
	CartTotal           float64               `json:"cart_total"`
	PaymentStatus       string                 `json:"payment_status,omitempty"`
	PaymentMethod       *string               `json:"payment_method,omitempty"`
	RejectionReason     *string               `json:"rejection_reason,omitempty"`
	TrackingCarrier     *string               `json:"tracking_carrier,omitempty"`
	TrackingNumber      *string               `json:"tracking_number,omitempty"`
	TrackingURL         *string               `json:"tracking_url,omitempty"`
	Items               []OrderItemResponse   `json:"items"`
	CreatedAt           string                 `json:"created_at"`
	UpdatedAt           string                 `json:"updated_at"`
}

type OrderItemResponse struct {
	SKU             string  `json:"sku"`
	Title           string  `json:"title"`
	Price           float64 `json:"price"`
	Quantity        int     `json:"quantity"`
	ProductURL      *string `json:"product_url,omitempty"`
	IsSupplierItem  bool    `json:"is_supplier_item"`
	ShopifyVariantID *int64 `json:"shopify_variant_id,omitempty"`
}

// HandleGetOrder handles GET /v1/orders/:id
func HandleGetOrder(repos *repository.Repositories, logger *zap.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get partner from context
		partner, ok := middleware.GetPartnerFromContext(c)
		if !ok {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}

		// Parse order ID
		orderIDStr := c.Param("id")
		orderID, err := uuid.Parse(orderIDStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid order ID"})
			return
		}

		// Get order
		order, err := repos.SupplierOrder.GetByID(c.Request.Context(), orderID)
		if err != nil {
			if _, ok := err.(*errors.ErrNotFound); ok {
				c.JSON(http.StatusNotFound, gin.H{"error": "order not found"})
				return
			}
			logger.Error("Failed to get order", zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
			return
		}

		// Verify partner owns this order
		if order.PartnerID != partner.ID {
			c.JSON(http.StatusForbidden, gin.H{"error": "access denied"})
			return
		}

		// Get order items
		items, err := repos.SupplierOrderItem.GetByOrderID(c.Request.Context(), orderID)
		if err != nil {
			logger.Error("Failed to get order items", zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
			return
		}

		// Build response
		itemResponses := make([]OrderItemResponse, len(items))
		for i, item := range items {
			itemResponses[i] = OrderItemResponse{
				SKU:              item.SKU,
				Title:            item.Title,
				Price:            item.Price,
				Quantity:         item.Quantity,
				ProductURL:       item.ProductURL,
				IsSupplierItem:   item.IsSupplierItem,
				ShopifyVariantID: item.ShopifyVariantID,
			}
		}

		response := OrderResponse{
			ID:                  order.ID.String(),
			PartnerOrderID:      order.PartnerOrderID,
			Status:              order.Status,
			ShopifyDraftOrderID: order.ShopifyDraftOrderID,
			CustomerName:        order.CustomerName,
			ShippingAddress:     order.ShippingAddress,
			CartTotal:           order.CartTotal,
			Items:               itemResponses,
			CreatedAt:           order.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
			UpdatedAt:           order.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
		}

		if order.CustomerPhone != "" {
			response.CustomerPhone = order.CustomerPhone
		}
		if order.PaymentStatus != "" {
			response.PaymentStatus = order.PaymentStatus
		}
		if order.PaymentMethod != nil {
			response.PaymentMethod = order.PaymentMethod
		}
		if order.RejectionReason != nil {
			response.RejectionReason = order.RejectionReason
		}
		if order.TrackingCarrier != nil {
			response.TrackingCarrier = order.TrackingCarrier
		}
		if order.TrackingNumber != nil {
			response.TrackingNumber = order.TrackingNumber
		}
		if order.TrackingURL != nil {
			response.TrackingURL = order.TrackingURL
		}

		c.JSON(http.StatusOK, response)
	}
}
