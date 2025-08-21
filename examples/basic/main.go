package main

import (
	"fmt"
	"log"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"github.com/thecodearcher/aegis"
	adapter "github.com/thecodearcher/aegis/adapters/gorm"
)

// Example showing basic usage of the aegis library
func main() {
	fmt.Println("Aegis Authentication Library - Basic Example")
	dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=disable",
		"localhost",
		"root",
		"root",
		"aegis",
		"5432",
	)
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatalf("Failed to open database: %v", err)
	}

	// Create configuration
	config := &aegis.Config{
		Database: aegis.DatabaseConfig{
			Adapter: adapter.New(db),
		},
	}

	aegis, err := aegis.New(config)
	if err != nil {
		log.Fatalf("Failed to create aegis: %v", err)
	}

	fmt.Printf("%+v\n", aegis)
	fmt.Println("Aegis instance created successfully!")
}
