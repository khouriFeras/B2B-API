package service

import (
	"context"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/jafarshop/b2bapi/internal/domain"
	"github.com/jafarshop/b2bapi/internal/repository"
	"github.com/jafarshop/b2bapi/pkg/errors"
)

type orderService struct {
	repos  *repository.Repositories
	logger *zap.Logger
}

// NewOrderService creates a new order service
func NewOrderService(repos *repository.Repositories, logger *zap.Logger) *orderService {
	return &orderService{
		repos:  repos,
		logger: logger,
	}
}

// CreateOrderFromCart creates a supplier order from a cart submission
func (s *orderService) CreateOrderFromCart(
	ctx context.Context,
	partnerID uuid.UUID,
	req CartSubmitRequest,
	supplierItems map[string]*domain.SKUMapping,
) (*domain.SupplierOrder, error) {
	// Create order
	order := &domain.SupplierOrder{
		PartnerID:      partnerID,
		PartnerOrderID: req.PartnerOrderID,
		Status:         domain.OrderStatusPendingConfirmation,
		CustomerName:   req.Customer.Name,
		CartTotal:      req.Totals.Total,
		PaymentStatus:  req.PaymentStatus,
		PaymentMethod:  req.PaymentMethod,
	}

	if req.Customer.Phone != nil {
		order.CustomerPhone = *req.Customer.Phone
	}

	// Convert shipping address to map
	order.ShippingAddress = map[string]interface{}{
		"street":      req.Shipping.Street,
		"city":        req.Shipping.City,
		"postal_code": req.Shipping.PostalCode,
		"country":     req.Shipping.Country,
	}
	if req.Shipping.State != nil {
		order.ShippingAddress["state"] = *req.Shipping.State
	}

	// Create order in database
	if err := s.repos.SupplierOrder.Create(ctx, order); err != nil {
		return nil, err
	}

	// Create order items
	items := make([]*domain.SupplierOrderItem, 0, len(req.Items))
	for _, cartItem := range req.Items {
		item := &domain.SupplierOrderItem{
			SupplierOrderID: order.ID,
			SKU:             cartItem.SKU,
			Title:           cartItem.Title,
			Price:           cartItem.Price,
			Quantity:        cartItem.Quantity,
			ProductURL:      cartItem.ProductURL,
		}

		// Check if this is a supplier item
		if mapping, ok := supplierItems[cartItem.SKU]; ok {
			item.IsSupplierItem = true
			item.ShopifyVariantID = &mapping.ShopifyVariantID
		}

		items = append(items, item)
	}

	// Create items in batch
	if err := s.repos.SupplierOrderItem.CreateBatch(ctx, items); err != nil {
		return nil, err
	}

	// Log order creation event
	event := &domain.OrderEvent{
		SupplierOrderID: order.ID,
		EventType:       "order_created",
		EventData: map[string]interface{}{
			"partner_order_id": req.PartnerOrderID,
			"status":           order.Status,
		},
	}
	s.repos.OrderEvent.Create(ctx, event)

	return order, nil
}

// ConfirmOrder confirms an order
func (s *orderService) ConfirmOrder(ctx context.Context, orderID uuid.UUID) error {
	order, err := s.repos.SupplierOrder.GetByID(ctx, orderID)
	if err != nil {
		return err
	}

	// Validate state transition
	if !order.Status.CanTransitionTo(domain.OrderStatusConfirmed) {
		return &errors.ErrInvalidStateTransition{
			From: order.Status,
			To:   domain.OrderStatusConfirmed,
		}
	}

	// Update status
	if err := s.repos.SupplierOrder.UpdateStatus(ctx, orderID, domain.OrderStatusConfirmed, nil); err != nil {
		return err
	}

	// Log event
	event := &domain.OrderEvent{
		SupplierOrderID: orderID,
		EventType:       "status_change",
		EventData: map[string]interface{}{
			"from": order.Status,
			"to":   domain.OrderStatusConfirmed,
		},
	}
	s.repos.OrderEvent.Create(ctx, event)

	return nil
}

// RejectOrder rejects an order
func (s *orderService) RejectOrder(ctx context.Context, orderID uuid.UUID, reason string) error {
	order, err := s.repos.SupplierOrder.GetByID(ctx, orderID)
	if err != nil {
		return err
	}

	// Validate state transition
	if !order.Status.CanTransitionTo(domain.OrderStatusRejected) {
		return &errors.ErrInvalidStateTransition{
			From: order.Status,
			To:   domain.OrderStatusRejected,
		}
	}

	// Update status
	if err := s.repos.SupplierOrder.UpdateStatus(ctx, orderID, domain.OrderStatusRejected, &reason); err != nil {
		return err
	}

	// Log event
	event := &domain.OrderEvent{
		SupplierOrderID: orderID,
		EventType:       "status_change",
		EventData: map[string]interface{}{
			"from":   order.Status,
			"to":     domain.OrderStatusRejected,
			"reason": reason,
		},
	}
	s.repos.OrderEvent.Create(ctx, event)

	return nil
}

// ShipOrder marks an order as shipped with tracking information
func (s *orderService) ShipOrder(ctx context.Context, orderID uuid.UUID, carrier, trackingNumber string, trackingURL *string) error {
	order, err := s.repos.SupplierOrder.GetByID(ctx, orderID)
	if err != nil {
		return err
	}

	// Validate state transition
	if !order.Status.CanTransitionTo(domain.OrderStatusShipped) {
		return &errors.ErrInvalidStateTransition{
			From: order.Status,
			To:   domain.OrderStatusShipped,
		}
	}

	// Update tracking
	if err := s.repos.SupplierOrder.UpdateTracking(ctx, orderID, &carrier, &trackingNumber, trackingURL); err != nil {
		return err
	}

	// Log event
	event := &domain.OrderEvent{
		SupplierOrderID: orderID,
		EventType:       "status_change",
		EventData: map[string]interface{}{
			"from":           order.Status,
			"to":             domain.OrderStatusShipped,
			"carrier":        carrier,
			"tracking_number": trackingNumber,
		},
	}
	if trackingURL != nil {
		event.EventData["tracking_url"] = *trackingURL
	}
	s.repos.OrderEvent.Create(ctx, event)

	return nil
}
