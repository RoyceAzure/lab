package sqlc

import (
	"context"
	"log"
	"os"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
)

var testQueries *Queries
var testDBPool *pgxpool.Pool // Make the pool available if needed for more complex tests

func TestMain(m *testing.M) {
	var err error
	testDBPool, err = pgxpool.New(context.Background(), "postgres://royce:password@localhost:5432/authcenter")
	if err != nil {
		log.Fatalf("Failed to connect to test database: %v", err)
	}
	defer testDBPool.Close()

	log.Println("Test database migrations applied successfully.")

	// Create a Queries instance for tests
	testQueries = New(testDBPool)

	// Run tests
	code := m.Run()

	// Optional: Teardown (e.g., drop test database or specific tables)
	// For simplicity, we are just closing the pool.
	// If you ran `migrator.Down()` here, it would revert all migrations.

	os.Exit(code)
}
