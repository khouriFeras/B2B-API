package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/jafarshop/b2bapi/internal/domain"
	"github.com/jafarshop/b2bapi/internal/repository"
	"github.com/jafarshop/b2bapi/internal/api/middleware"
	"github.com/jafarshop/b2bapi/internal/service"
	"github.com/jafarshop/b2bapi/pkg/errors"
)

// ConfirmOrderRequest represents confirm order request
type ConfirmOrderRequest struct {
	// Empty for now, can add fields later
}

// RejectOrderRequest represents reject order request
type RejectOrderRequest struct {
	Reason string `json:"reason" binding:"required"`
}

// ShipOrderRequest represents ship order request
type ShipOrderRequest struct {
	Carrier        string `json:"carrier" binding:"required"`
	TrackingNumber string `json:"tracking_number" binding:"required"`
	TrackingURL    *string `json:"tracking_url,omitempty"`
}

// HandleConfirmOrder handles POST /v1/admin/orders/:id/confirm
func HandleConfirmOrder(repos *repository.Repositories, logger *zap.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get partner from context (for now, admin uses same auth)
		_, ok := middleware.GetPartnerFromContext(c)
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

		// Confirm order
		orderService := service.NewOrderService(repos, logger)
		if err := orderService.ConfirmOrder(c.Request.Context(), orderID); err != nil {
			if _, ok := err.(*errors.ErrInvalidStateTransition); ok {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			logger.Error("Failed to confirm order", zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to confirm order"})
			return
		}

		// Get updated order
		order, _ = repos.SupplierOrder.GetByID(c.Request.Context(), orderID)

		c.JSON(http.StatusOK, gin.H{
			"id":     order.ID.String(),
			"status": order.Status,
		})
	}
}

// HandleRejectOrder handles POST /v1/admin/orders/:id/reject
func HandleRejectOrder(repos *repository.Repositories, logger *zap.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get partner from context
		_, ok := middleware.GetPartnerFromContext(c)
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

		// Parse request
		var req RejectOrderRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusUnprocessableEntity, gin.H{
				"error":   "validation failed",
				"details": err.Error(),
			})
			return
		}

		// Reject order
		orderService := service.NewOrderService(repos, logger)
		if err := orderService.RejectOrder(c.Request.Context(), orderID, req.Reason); err != nil {
			if _, ok := err.(*errors.ErrInvalidStateTransition); ok {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			logger.Error("Failed to reject order", zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to reject order"})
			return
		}

		// Get updated order
		order, _ := repos.SupplierOrder.GetByID(c.Request.Context(), orderID)

		c.JSON(http.StatusOK, gin.H{
			"id":     order.ID.String(),
			"status": order.Status,
		})
	}
}

// HandleShipOrder handles POST /v1/admin/orders/:id/ship
func HandleShipOrder(repos *repository.Repositories, logger *zap.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get partner from context
		_, ok := middleware.GetPartnerFromContext(c)
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

		// Parse request
		var req ShipOrderRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusUnprocessableEntity, gin.H{
				"error":   "validation failed",
				"details": err.Error(),
			})
			return
		}

		// Ship order
		orderService := service.NewOrderService(repos, logger)
		if err := orderService.ShipOrder(c.Request.Context(), orderID, req.Carrier, req.TrackingNumber, req.TrackingURL); err != nil {
			if _, ok := err.(*errors.ErrInvalidStateTransition); ok {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			logger.Error("Failed to ship order", zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to ship order"})
			return
		}

		// Get updated order
		order, _ := repos.SupplierOrder.GetByID(c.Request.Context(), orderID)

		c.JSON(http.StatusOK, gin.H{
			"id":              order.ID.String(),
			"status":          order.Status,
			"tracking_carrier": order.TrackingCarrier,
			"tracking_number": order.TrackingNumber,
			"tracking_url":    order.TrackingURL,
		})
	}
}

// HandleListOrders handles GET /v1/admin/orders
func HandleListOrders(repos *repository.Repositories, logger *zap.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get partner from context
		partner, ok := middleware.GetPartnerFromContext(c)
		if !ok {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}

		// Parse query parameters
		statusStr := c.Query("status")
		limitStr := c.DefaultQuery("limit", "50")
		offsetStr := c.DefaultQuery("offset", "0")

		limit, err := strconv.Atoi(limitStr)
		if err != nil || limit < 1 || limit > 100 {
			limit = 50
		}

		offset, err := strconv.Atoi(offsetStr)
		if err != nil || offset < 0 {
			offset = 0
		}

		var orders []*domain.SupplierOrder
		if statusStr != "" {
			status := domain.OrderStatus(statusStr)
			if !status.IsValid() {
				c.JSON(http.StatusBadRequest, gin.H{"error": "invalid status"})
				return
			}
			orders, err = repos.SupplierOrder.ListByStatus(c.Request.Context(), status, limit, offset)
		} else {
			orders, err = repos.SupplierOrder.ListByPartnerID(c.Request.Context(), partner.ID, limit, offset)
		}

		if err != nil {
			logger.Error("Failed to list orders", zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
			return
		}

		// Build response
		orderResponses := make([]gin.H, len(orders))
		for i, order := range orders {
			orderResponses[i] = gin.H{
				"id":                  order.ID.String(),
				"partner_order_id":   order.PartnerOrderID,
				"status":             order.Status,
				"shopify_draft_order_id": order.ShopifyDraftOrderID,
				"customer_name":      order.CustomerName,
				"cart_total":         order.CartTotal,
				"created_at":         order.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
				"updated_at":         order.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
			}
		}

		c.JSON(http.StatusOK, gin.H{
			"orders": orderResponses,
			"limit":  limit,
			"offset": offset,
		})
	}
}
