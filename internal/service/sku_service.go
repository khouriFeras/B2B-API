package service

import (
	"context"

	"go.uber.org/zap"

	"github.com/jafarshop/b2bapi/internal/domain"
	"github.com/jafarshop/b2bapi/internal/repository"
)

type skuService struct {
	repos  *repository.Repositories
	logger *zap.Logger
}

// NewSKUService creates a new SKU service
func NewSKUService(repos *repository.Repositories, logger *zap.Logger) *skuService {
	return &skuService{
		repos:  repos,
		logger: logger,
	}
}

// CheckCartForSupplierSKUs checks if cart contains at least one supplier SKU
// Returns: hasSupplierSKU, supplierItems map (SKU -> mapping), error
func (s *skuService) CheckCartForSupplierSKUs(
	ctx context.Context,
	items []CartItem,
) (bool, map[string]*domain.SKUMapping, error) {
	supplierItems := make(map[string]*domain.SKUMapping)

	for _, item := range items {
		mapping, err := s.repos.SKUMapping.GetBySKU(ctx, item.SKU)
		if err != nil {
			// SKU not found or error - skip
			continue
		}

		if mapping.IsActive {
			supplierItems[item.SKU] = mapping
		}
	}

	return len(supplierItems) > 0, supplierItems, nil
}
