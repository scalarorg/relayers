package evm_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
	postgresDriver "gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func TestDB(t *testing.T) {
	ctx := context.Background()

	// Setup test database configuration
	dbName := "users"
	dbUser := "user"
	dbPassword := "password"

	// Create and run postgres container
	postgresContainer, err := postgres.Run(ctx,
		"postgres:16-alpine",
		postgres.WithDatabase(dbName),
		postgres.WithUsername(dbUser),
		postgres.WithPassword(dbPassword),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(5*time.Second)),
	)
	require.NoError(t, err)
	defer postgresContainer.Terminate(ctx)

	// Get container connection details
	host, err := postgresContainer.Host(ctx)
	require.NoError(t, err)
	port, err := postgresContainer.MappedPort(ctx, "5432")
	require.NoError(t, err)

	// Create connection string
	dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%d sslmode=disable TimeZone=UTC",
		host, dbUser, dbPassword, dbName, port.Int())

	// Connect to database
	db, err := gorm.Open(postgresDriver.Open(dsn), &gorm.Config{})
	require.NoError(t, err)

	// Verify connection works by running a simple query
	var result int
	err = db.Raw("SELECT 1").Scan(&result).Error
	require.NoError(t, err)
	require.Equal(t, 1, result)
}
