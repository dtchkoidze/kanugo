package main

import (
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/joho/godotenv"
)

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	dbURL := os.Getenv("DB_URL")
	if dbURL == "" {
		log.Fatal("DB_URL environment variable is not set")
	}

	conn, err := pgx.Connect(context.Background(), dbURL)
	if err != nil {
		log.Fatalf("Unable to connect to database: %v\n", err)
	}
	defer conn.Close(context.Background())

	_, err = conn.Exec(context.Background(), `
		CREATE TABLE IF NOT EXISTS migrations (
			id SERIAL PRIMARY KEY,
			name TEXT NOT NULL UNIQUE,
			applied_at TIMESTAMP NOT NULL DEFAULT NOW()
		)
	`)
	if err != nil {
		log.Fatalf("Failed to create migrations table: %v\n", err)
	}

	migrationFiles, err := filepath.Glob("migrations/*.sql")
	if err != nil {
		log.Fatalf("Failed to read migration files: %v\n", err)
	}

	for _, file := range migrationFiles {
		filename := filepath.Base(file)

		var exists bool
		err := conn.QueryRow(context.Background(),
			"SELECT EXISTS(SELECT 1 FROM migrations WHERE name = $1)",
			filename).Scan(&exists)

		if err != nil {
			log.Fatalf("Failed to check if migration exists: %v\n", err)
		}

		if exists {
			fmt.Printf("Migration %s already applied, skipping\n", filename)
			continue
		}

		content, err := ioutil.ReadFile(file)
		if err != nil {
			log.Fatalf("Failed to read migration file %s: %v\n", file, err)
		}

		parts := strings.Split(string(content), "-- +migrate Down")
		if len(parts) != 2 {
			log.Fatalf("Invalid migration format in %s\n", file)
		}

		upMigration := strings.Split(parts[0], "-- +migrate Up")[1]

		fmt.Printf("Applying migration: %s\n", filename)
		tx, err := conn.Begin(context.Background())
		if err != nil {
			log.Fatalf("Failed to start transaction: %v\n", err)
		}

		_, err = tx.Exec(context.Background(), upMigration)
		if err != nil {
			tx.Rollback(context.Background())
			log.Fatalf("Failed to apply migration %s: %v\n", filename, err)
		}

		_, err = tx.Exec(context.Background(),
			"INSERT INTO migrations (name) VALUES ($1)",
			filename)
		if err != nil {
			tx.Rollback(context.Background())
			log.Fatalf("Failed to record migration %s: %v\n", filename, err)
		}

		err = tx.Commit(context.Background())
		if err != nil {
			log.Fatalf("Failed to commit transaction: %v\n", err)
		}

		fmt.Printf("Successfully applied migration: %s\n", filename)
	}

	fmt.Println("All migrations applied successfully!")
}
