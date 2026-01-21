package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/jafarshop/b2bapi/internal/config"
	"github.com/jafarshop/b2bapi/internal/domain"
	"github.com/jafarshop/b2bapi/internal/repository"
	"github.com/jafarshop/b2bapi/internal/api/middleware"
	"github.com/jafarshop/b2bapi/internal/service"
)

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

// CartSubmitResponse represents the response
type CartSubmitResponse struct {
	SupplierOrderID string                `json:"supplier_order_id"`
	Status          domain.OrderStatus    `json:"status"`
}

func HandleCartSubmit(cfg *config.Config, repos *repository.Repositories, logger *zap.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get partner from context
		partner, ok := middleware.GetPartnerFromContext(c)
		if !ok {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}

		// Check if this is an idempotent request
		_, _, existingOrderID, isExisting := middleware.GetIdempotencyInfo(c)
		if isExisting {
			// Return existing order
			orderID, err := uuid.Parse(existingOrderID)
			if err != nil {
				logger.Error("Invalid existing order ID from idempotency", zap.Error(err))
				c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
				return
			}

			order, err := repos.SupplierOrder.GetByID(c.Request.Context(), orderID)
			if err != nil {
				logger.Error("Failed to get existing order", zap.Error(err))
				c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
				return
			}

			c.JSON(http.StatusOK, CartSubmitResponse{
				SupplierOrderID: order.ID.String(),
				Status:          order.Status,
			})
			return
		}

		// Parse request - use service types
		var req service.CartSubmitRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusUnprocessableEntity, gin.H{
				"error":   "validation failed",
				"details": err.Error(),
			})
			return
		}

		// Check for supplier SKUs
		skuService := service.NewSKUService(repos, logger)
		hasSupplierSKU, supplierItems, err := skuService.CheckCartForSupplierSKUs(
			c.Request.Context(),
			req.Items, // []service.CartItem
		)
		if err != nil {
			logger.Error("Failed to check SKUs", zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
			return
		}

		// If no supplier SKUs, return 204
		if !hasSupplierSKU {
			c.Status(http.StatusNoContent)
			return
		}

		// Create order
		orderService := service.NewOrderService(repos, logger)
		order, err := orderService.CreateOrderFromCart(c.Request.Context(), partner.ID, req, supplierItems)
		if err != nil {
			logger.Error("Failed to create order", zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create order"})
			return
		}

		// Create Shopify draft order
		// Get order items for draft order creation
		orderItems, err := repos.SupplierOrderItem.GetByOrderID(c.Request.Context(), order.ID)
		if err != nil {
			logger.Error("Failed to get order items for draft order", zap.Error(err))
			// Don't fail the request, draft order can be created later
		} else {
			shopifyService := service.NewShopifyService(cfg.Shopify, repos, logger)
			draftOrderID, err := shopifyService.CreateDraftOrder(c.Request.Context(), order, orderItems, partner.Name)
			if err != nil {
				logger.Error("Failed to create Shopify draft order", zap.Error(err))
				// Don't fail the request, draft order can be created later
			} else {
				// Update order with draft order ID
				if err := repos.SupplierOrder.UpdateShopifyDraftOrderID(c.Request.Context(), order.ID, draftOrderID); err != nil {
					logger.Warn("Failed to update order with draft order ID", zap.Error(err))
				}
				order.ShopifyDraftOrderID = &draftOrderID

				// Complete draft order -> create a real Shopify Order (so it shows under Orders, not Drafts)
				shopifyOrderID, err := shopifyService.CompleteDraftOrder(c.Request.Context(), draftOrderID)
				if err != nil {
					logger.Error("Failed to complete Shopify draft order", zap.Error(err))
				} else {
					if err := repos.SupplierOrder.UpdateShopifyOrderID(c.Request.Context(), order.ID, shopifyOrderID); err != nil {
						logger.Warn("Failed to update order with Shopify order ID", zap.Error(err))
					}
					order.ShopifyOrderID = &shopifyOrderID
				}
			}
		}

		// Store idempotency key if provided
		idempotencyKey, requestHash, _, _ := middleware.GetIdempotencyInfo(c)
		if idempotencyKey != "" {
			idempotency := &domain.IdempotencyKey{
				Key:             idempotencyKey,
				PartnerID:       partner.ID,
				SupplierOrderID: order.ID,
				RequestHash:     requestHash,
			}
			if err := repos.IdempotencyKey.Create(c.Request.Context(), idempotency); err != nil {
				logger.Warn("Failed to store idempotency key", zap.Error(err))
				// Don't fail the request if idempotency storage fails
			}
		}

		c.JSON(http.StatusOK, CartSubmitResponse{
			SupplierOrderID: order.ID.String(),
			Status:          order.Status,
		})
	}
}
