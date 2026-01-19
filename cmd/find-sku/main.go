package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/jafarshop/b2bapi/internal/config"
	"github.com/jafarshop/b2bapi/internal/shopify"
	"go.uber.org/zap"
)

// ProductsQuery fetches products with variants
const ProductsQuery = `
query getProducts($first: Int!, $after: String) {
  products(first: $first, after: $after) {
    pageInfo {
      hasNextPage
      endCursor
    }
    edges {
      node {
        id
        title
        variants(first: 250) {
          edges {
            node {
              id
              sku
              title
              price
            }
          }
        }
      }
    }
  }
}
`

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: go run cmd/find-sku/main.go <sku>")
		fmt.Println("Example: go run cmd/find-sku/main.go \"SCM 8502\"")
		os.Exit(1)
	}

	targetSKU := os.Args[1]

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load configuration: %v\n", err)
		os.Exit(1)
	}

	// Initialize logger
	logger, _ := zap.NewDevelopment()
	defer logger.Sync()

	// Create Shopify client
	client := shopify.NewClient(cfg.Shopify, logger)

	fmt.Printf("🔍 Searching for SKU: %s\n\n", targetSKU)

	// Search through products
	hasNextPage := true
	after := ""
	found := false

	for hasNextPage {
		variables := map[string]interface{}{
			"first": 50,
		}
		if after != "" {
			variables["after"] = after
		}

		resp, err := client.Execute(ProductsQuery, variables)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to query Shopify: %v\n", err)
			os.Exit(1)
		}

		// Parse response
		var result struct {
			Data struct {
				Products struct {
					PageInfo struct {
						HasNextPage bool   `json:"hasNextPage"`
						EndCursor   string `json:"endCursor"`
					} `json:"pageInfo"`
					Edges []struct {
						Node struct {
							ID      string `json:"id"`
							Title   string `json:"title"`
							Variants struct {
								Edges []struct {
									Node struct {
										ID    string `json:"id"`
										SKU   string `json:"sku"`
										Title string `json:"title"`
										Price string `json:"price"`
									} `json:"node"`
								} `json:"edges"`
							} `json:"variants"`
						} `json:"node"`
					} `json:"edges"`
				} `json:"products"`
			} `json:"data"`
		}

		if err := json.Unmarshal(resp.Data, &result); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to parse response: %v\n", err)
			os.Exit(1)
		}

		// Search through products and variants
		for _, productEdge := range result.Data.Products.Edges {
			product := productEdge.Node
			
			for _, variantEdge := range product.Variants.Edges {
				variant := variantEdge.Node
				
				if variant.SKU == targetSKU {
					// Extract numeric IDs from GIDs
					productID := extractIDFromGID(product.ID)
					variantID := extractIDFromGID(variant.ID)
					
					fmt.Printf("✅ Found SKU!\n\n")
					fmt.Printf("SKU: %s\n", variant.SKU)
					fmt.Printf("Product Title: %s\n", product.Title)
					fmt.Printf("Variant Title: %s\n", variant.Title)
					fmt.Printf("Price: %s\n", variant.Price)
					fmt.Printf("\nIDs:\n")
					fmt.Printf("  Product ID: %d\n", productID)
					fmt.Printf("  Variant ID: %d\n", variantID)
					fmt.Printf("\nTo add this to the database, run:\n")
					fmt.Printf("go run cmd/add-sku/main.go \"%s\" %d %d\n", 
						targetSKU, productID, variantID)
					
					found = true
					hasNextPage = false
					break
				}
			}
			
			if found {
				break
			}
		}

		if !found {
			hasNextPage = result.Data.Products.PageInfo.HasNextPage
			after = result.Data.Products.PageInfo.EndCursor
			
			if hasNextPage {
				fmt.Printf("⏳ Searching... (checked %d products so far)\n", len(result.Data.Products.Edges))
			}
		}
	}

	if !found {
		fmt.Printf("❌ SKU '%s' not found in your Shopify store.\n", targetSKU)
		fmt.Printf("\nMake sure:\n")
		fmt.Printf("  1. The SKU is correct (case-sensitive)\n")
		fmt.Printf("  2. The product is published in Shopify\n")
		fmt.Printf("  3. The variant has a SKU assigned\n")
		os.Exit(1)
	}
}

func extractIDFromGID(gid string) int64 {
	// GID format: "gid://shopify/Product/123456" or "gid://shopify/ProductVariant/123456"
	parts := []rune(gid)
	start := -1
	end := len(parts)
	
	// Find the last number sequence
	for i := len(parts) - 1; i >= 0; i-- {
		if parts[i] >= '0' && parts[i] <= '9' {
			if end == len(parts) {
				end = i + 1
			}
			start = i
		} else if start != -1 {
			break
		}
	}
	
	if start == -1 {
		return 0
	}
	
	var id int64
	for i := start; i < end; i++ {
		id = id*10 + int64(parts[i]-'0')
	}
	
	return id
}
