package domain

// OrderStatus represents the status of a supplier order
type OrderStatus string

const (
	OrderStatusPendingConfirmation OrderStatus = "PENDING_CONFIRMATION"
	OrderStatusConfirmed           OrderStatus = "CONFIRMED"
	OrderStatusRejected            OrderStatus = "REJECTED"
	OrderStatusShipped             OrderStatus = "SHIPPED"
	OrderStatusDelivered           OrderStatus = "DELIVERED"
	OrderStatusCancelled           OrderStatus = "CANCELLED"
)

// IsValid checks if the order status is valid
func (s OrderStatus) IsValid() bool {
	switch s {
	case OrderStatusPendingConfirmation,
		OrderStatusConfirmed,
		OrderStatusRejected,
		OrderStatusShipped,
		OrderStatusDelivered,
		OrderStatusCancelled:
		return true
	default:
		return false
	}
}

// CanTransitionTo checks if a status transition is valid
func (s OrderStatus) CanTransitionTo(newStatus OrderStatus) bool {
	switch s {
	case OrderStatusPendingConfirmation:
		return newStatus == OrderStatusConfirmed ||
			newStatus == OrderStatusRejected ||
			newStatus == OrderStatusCancelled
	case OrderStatusConfirmed:
		return newStatus == OrderStatusShipped ||
			newStatus == OrderStatusCancelled
	case OrderStatusShipped:
		return newStatus == OrderStatusDelivered
	case OrderStatusRejected, OrderStatusDelivered, OrderStatusCancelled:
		return false // Terminal states
	default:
		return false
	}
}
