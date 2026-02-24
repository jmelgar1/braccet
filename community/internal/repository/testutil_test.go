//go:build integration

package repository

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"

	_ "github.com/lib/pq"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
)

// testDB holds the shared database connection for integration tests
type testDB struct {
	db        *sql.DB
	container testcontainers.Container
}

// setupTestDB creates a PostgreSQL container and runs migrations
func setupTestDB(t *testing.T) *sql.DB {
	t.Helper()
	ctx := context.Background()

	// Start PostgreSQL container
	container, err := postgres.Run(ctx,
		"postgres:16-alpine",
		postgres.WithDatabase("test_community"),
		postgres.WithUsername("test"),
		postgres.WithPassword("test"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(30*time.Second),
		),
	)
	if err != nil {
		t.Fatalf("failed to start postgres container: %v", err)
	}

	// Register cleanup
	t.Cleanup(func() {
		if err := container.Terminate(ctx); err != nil {
			t.Logf("failed to terminate container: %v", err)
		}
	})

	// Get connection string
	connStr, err := container.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		t.Fatalf("failed to get connection string: %v", err)
	}

	// Connect to database
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		t.Fatalf("failed to connect to database: %v", err)
	}

	t.Cleanup(func() {
		db.Close()
	})

	// Wait for database to be ready
	for i := 0; i < 30; i++ {
		if err := db.PingContext(ctx); err == nil {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}

	// Run migrations
	if err := runMigrations(t, db); err != nil {
		t.Fatalf("failed to run migrations: %v", err)
	}

	return db
}

// runMigrations executes all .up.sql migrations in order
func runMigrations(t *testing.T, db *sql.DB) error {
	t.Helper()

	// Find migrations directory (relative to test file)
	migrationsDir := findMigrationsDir(t)

	// Read all migration files
	files, err := os.ReadDir(migrationsDir)
	if err != nil {
		return fmt.Errorf("failed to read migrations directory: %w", err)
	}

	// Collect and sort .up.sql files
	var upFiles []string
	for _, f := range files {
		if strings.HasSuffix(f.Name(), ".up.sql") {
			upFiles = append(upFiles, f.Name())
		}
	}
	sort.Strings(upFiles)

	// Execute each migration
	for _, fileName := range upFiles {
		path := filepath.Join(migrationsDir, fileName)
		content, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("failed to read migration %s: %w", fileName, err)
		}

		if _, err := db.Exec(string(content)); err != nil {
			return fmt.Errorf("failed to execute migration %s: %w", fileName, err)
		}
		t.Logf("applied migration: %s", fileName)
	}

	return nil
}

// findMigrationsDir locates the migrations directory
func findMigrationsDir(t *testing.T) string {
	t.Helper()

	// Try relative paths from test location
	candidates := []string{
		"../../../migrations",
		"../../migrations",
		"../migrations",
		"migrations",
	}

	for _, candidate := range candidates {
		if info, err := os.Stat(candidate); err == nil && info.IsDir() {
			return candidate
		}
	}

	// Try absolute path
	wd, _ := os.Getwd()
	t.Logf("working directory: %s", wd)

	// Go up directories looking for migrations
	dir := wd
	for i := 0; i < 5; i++ {
		migPath := filepath.Join(dir, "migrations")
		if info, err := os.Stat(migPath); err == nil && info.IsDir() {
			return migPath
		}
		dir = filepath.Dir(dir)
	}

	t.Fatalf("could not find migrations directory")
	return ""
}

// cleanupTestData truncates all test tables (use between tests if needed)
func cleanupTestData(t *testing.T, db *sql.DB) {
	t.Helper()

	tables := []string{
		"power_ranking_match_cache",
		"member_power_rankings",
		"tournament_placements",
		"power_ranking_systems",
		"elo_history",
		"member_elo_ratings",
		"elo_systems",
		"community_members",
		"communities",
	}

	for _, table := range tables {
		_, err := db.Exec(fmt.Sprintf("TRUNCATE TABLE %s CASCADE", table))
		if err != nil {
			t.Logf("warning: failed to truncate %s: %v", table, err)
		}
	}
}

// createTestCommunity creates a community for test data
func createTestCommunity(t *testing.T, db *sql.DB) uint64 {
	t.Helper()

	var id uint64
	err := db.QueryRow(`
		INSERT INTO communities (name, slug, owner_id, description)
		VALUES ('Test Community', 'test-community', 1, 'A test community')
		RETURNING id
	`).Scan(&id)
	if err != nil {
		t.Fatalf("failed to create test community: %v", err)
	}
	return id
}

// createTestMember creates a community member for test data
func createTestMember(t *testing.T, db *sql.DB, communityID uint64, userID uint64, displayName string) uint64 {
	t.Helper()

	var id uint64
	err := db.QueryRow(`
		INSERT INTO community_members (community_id, user_id, display_name, role)
		VALUES ($1, $2, $3, 'member')
		RETURNING id
	`, communityID, userID, displayName).Scan(&id)
	if err != nil {
		t.Fatalf("failed to create test member: %v", err)
	}
	return id
}

// ptrUint64 returns a pointer to the given uint64
func ptrUint64(v uint64) *uint64 {
	return &v
}

// ptrInt returns a pointer to the given int
func ptrInt(v int) *int {
	return &v
}

// ptrFloat64 returns a pointer to the given float64
func ptrFloat64(v float64) *float64 {
	return &v
}

// ptrString returns a pointer to the given string
func ptrString(v string) *string {
	return &v
}

// ptrTime returns a pointer to the given time
func ptrTime(v time.Time) *time.Time {
	return &v
}
