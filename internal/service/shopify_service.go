package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"go.uber.org/zap"

	"github.com/jafarshop/b2bapi/internal/config"
	"github.com/jafarshop/b2bapi/internal/domain"
	"github.com/jafarshop/b2bapi/internal/repository"
	"github.com/jafarshop/b2bapi/internal/shopify"
)

type shopifyService struct {
	client  *shopify.Client
	repos   *repository.Repositories
	logger  *zap.Logger
}

// NewShopifyService creates a new Shopify service
func NewShopifyService(cfg config.ShopifyConfig, repos *repository.Repositories, logger *zap.Logger) *shopifyService {
	return &shopifyService{
		client: shopify.NewClient(cfg, logger),
		repos:  repos,
		logger: logger,
	}
}

// CreateDraftOrder creates a Shopify draft order from a supplier order
func (s *shopifyService) CreateDraftOrder(
	ctx context.Context,
	order *domain.SupplierOrder,
	items []*domain.SupplierOrderItem,
	partnerName string,
) (int64, error) {
	// Build line items
	lineItems := make([]shopify.DraftOrderLineItemInput, 0, len(items))
	
	for _, item := range items {
		if item.IsSupplierItem && item.ShopifyVariantID != nil {
			// Supplier item - use variant
			variantIDStr := fmt.Sprintf("gid://shopify/ProductVariant/%d", *item.ShopifyVariantID)
			lineItems = append(lineItems, shopify.DraftOrderLineItemInput{
				VariantID: &variantIDStr,
				Quantity:  item.Quantity,
			})
		} else {
			// Non-supplier item - use custom line item
			priceStr := fmt.Sprintf("%.2f", item.Price)
			title := item.Title
			if item.ProductURL != nil {
				title = fmt.Sprintf("%s (URL: %s)", title, *item.ProductURL)
			}
			
			customAttrs := []shopify.DraftOrderAttributeInput{
				{Key: "product_url", Value: *item.ProductURL},
			}
			if item.ProductURL == nil {
				customAttrs = []shopify.DraftOrderAttributeInput{}
			}
			
			lineItems = append(lineItems, shopify.DraftOrderLineItemInput{
				Title:  &title,
				Price:  &priceStr,
				Quantity: item.Quantity,
				CustomAttributes: customAttrs,
			})
		}
	}

	// Build shipping address
	shippingAddr := shopify.DraftOrderAddressInput{
		Address1: getStringFromMap(order.ShippingAddress, "street"),
		City:     getStringFromMap(order.ShippingAddress, "city"),
		Zip:      getStringFromMap(order.ShippingAddress, "postal_code"),
		Country:  getStringFromMap(order.ShippingAddress, "country"),
	}
	
	// Parse customer name (assume "FirstName LastName" or just "Name")
	nameParts := strings.Fields(order.CustomerName)
	if len(nameParts) > 0 {
		shippingAddr.FirstName = nameParts[0]
		if len(nameParts) > 1 {
			lastName := strings.Join(nameParts[1:], " ")
			shippingAddr.LastName = &lastName
		}
	}
	
	if state, ok := order.ShippingAddress["state"].(string); ok && state != "" {
		shippingAddr.Province = &state
	}
	
	if order.CustomerPhone != "" {
		shippingAddr.Phone = &order.CustomerPhone
	}

	// Build tags
	tags := []string{
		fmt.Sprintf("partner:%s", partnerName),
		fmt.Sprintf("partner_order:%s", order.PartnerOrderID),
		"pending_confirmation",
	}
	
	// Check if mixed cart (has both supplier and non-supplier items)
	hasSupplierItems := false
	hasNonSupplierItems := false
	for _, item := range items {
		if item.IsSupplierItem {
			hasSupplierItems = true
		} else {
			hasNonSupplierItems = true
		}
	}
	
	if hasSupplierItems && hasNonSupplierItems {
		tags = append(tags, "mixed_cart")
	}

	// Build input
	input := shopify.DraftOrderInput{
		LineItems:      lineItems,
		ShippingAddress: &shippingAddr,
		Tags:           tags,
		Note:           stringPtr(fmt.Sprintf("Partner Order ID: %s", order.PartnerOrderID)),
	}

	// Execute mutation
	variables := map[string]interface{}{
		"input": input,
	}

	resp, err := s.client.Execute(shopify.DraftOrderCreateMutation, variables)
	if err != nil {
		return 0, fmt.Errorf("failed to create draft order: %w", err)
	}

	// Parse response to get draft order ID
	// The response structure: { "data": { "draftOrderCreate": { "draftOrder": { "id": "gid://shopify/DraftOrder/..." } } } }
	var result struct {
		Data struct {
			DraftOrderCreate struct {
				DraftOrder struct {
					ID string `json:"id"`
				} `json:"draftOrder"`
				UserErrors []struct {
					Field   []string `json:"field"`
					Message string   `json:"message"`
				} `json:"userErrors"`
			} `json:"draftOrderCreate"`
		} `json:"data"`
	}

	if err := json.Unmarshal(resp.Data, &result); err != nil {
		return 0, fmt.Errorf("failed to parse draft order response: %w", err)
	}

	if len(result.Data.DraftOrderCreate.UserErrors) > 0 {
		return 0, fmt.Errorf("shopify user errors: %v", result.Data.DraftOrderCreate.UserErrors)
	}

	// Extract numeric ID from GID
	draftOrderGID := result.Data.DraftOrderCreate.DraftOrder.ID
	draftOrderID, err := extractIDFromGID(draftOrderGID)
	if err != nil {
		return 0, fmt.Errorf("failed to extract draft order ID: %w", err)
	}

	return draftOrderID, nil
}

// Helper functions
func getStringFromMap(m map[string]interface{}, key string) string {
	if val, ok := m[key].(string); ok {
		return val
	}
	return ""
}

func stringPtr(s string) *string {
	return &s
}

func extractIDFromGID(gid string) (int64, error) {
	// GID format: "gid://shopify/DraftOrder/123456"
	parts := strings.Split(gid, "/")
	if len(parts) < 4 {
		return 0, fmt.Errorf("invalid GID format: %s", gid)
	}
	
	id, err := strconv.ParseInt(parts[len(parts)-1], 10, 64)
	if err != nil {
		return 0, fmt.Errorf("failed to parse ID from GID: %w", err)
	}
	
	return id, nil
}
