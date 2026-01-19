package main

import (
	"context"
	"fmt"
	"os"

	"golang.org/x/crypto/bcrypt"

	"github.com/jafarshop/b2bapi/internal/config"
	"github.com/jafarshop/b2bapi/internal/domain"
	"github.com/jafarshop/b2bapi/internal/repository/postgres"
	"go.uber.org/zap"
)

func main() {
	if len(os.Args) < 3 {
		fmt.Println("Usage: go run cmd/create-partner/main.go <partner-name> <api-key>")
		fmt.Println("Example: go run cmd/create-partner/main.go \"Zain Shop\" \"zain-api-key-12345\"")
		os.Exit(1)
	}

	partnerName := os.Args[1]
	apiKey := os.Args[2]

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load configuration: %v\n", err)
		os.Exit(1)
	}

	// Initialize logger
	logger, _ := zap.NewDevelopment()
	defer logger.Sync()

	// Connect to database
	db, err := postgres.NewConnection(cfg.Database)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to connect to database: %v\n", err)
		os.Exit(1)
	}
	defer db.Close()

	// Hash the API key
	apiKeyHash, err := bcrypt.GenerateFromPassword([]byte(apiKey), 10)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to hash API key: %v\n", err)
		os.Exit(1)
	}

	// Create repositories
	repos := postgres.NewRepositories(db, logger)

	// Create partner
	partner := &domain.Partner{
		Name:       partnerName,
		APIKeyHash: string(apiKeyHash),
		IsActive:   true,
	}

	err = repos.Partner.Create(context.Background(), partner)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create partner: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("✅ Partner created successfully!\n\n")
	fmt.Printf("Partner ID: %s\n", partner.ID.String())
	fmt.Printf("Partner Name: %s\n", partner.Name)
	fmt.Printf("API Key: %s\n", apiKey)
	fmt.Printf("\n⚠️  IMPORTANT: Save this API key securely! You won't be able to see it again.\n")
	fmt.Printf("\nUse this API key in the Authorization header:\n")
	fmt.Printf("Authorization: Bearer %s\n", apiKey)
}
