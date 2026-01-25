package postgres_test

import (
	"database/sql"
	"os"
	"testing"
	"time"

	"github.com/keneke/delivertrack/pkg/postgres"
)

// TestNew_Success tests successful connection to PostgreSQL
func TestNew_Success(t *testing.T) {
	// Skip if DATABASE_URL is not set
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		t.Skip("DATABASE_URL not set, skipping integration test")
	}

	db, err := postgres.New(dbURL)
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	// Verify connection is alive
	if err := db.Ping(); err != nil {
		t.Errorf("Failed to ping database: %v", err)
	}

	// Verify connection pool settings
	stats := db.Stats()
	if stats.MaxOpenConnections != 25 {
		t.Errorf("Expected MaxOpenConnections=25, got %d", stats.MaxOpenConnections)
	}
}

// TestNew_InvalidURL tests connection with invalid URL
func TestNew_InvalidURL(t *testing.T) {
	invalidURLs := []string{
		"invalid://url",
		"postgres://invalid:5432",
		"",
	}

	for _, url := range invalidURLs {
		_, err := postgres.New(url)
		if err == nil {
			t.Errorf("Expected error for invalid URL %q, got nil", url)
		}
	}
}

// TestNew_ConnectionFailure tests connection to non-existent database
func TestNew_ConnectionFailure(t *testing.T) {
	// Try to connect to non-existent host
	invalidURL := "postgres://user:pass@nonexistent:5432/db?sslmode=disable&connect_timeout=1"

	_, err := postgres.New(invalidURL)
	if err == nil {
		t.Error("Expected error when connecting to non-existent host, got nil")
	}
}

// TestClose tests database connection closing
func TestClose(t *testing.T) {
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		t.Skip("DATABASE_URL not set, skipping integration test")
	}

	db, err := postgres.New(dbURL)
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}

	// Close the connection
	if err := db.Close(); err != nil {
		t.Errorf("Failed to close database: %v", err)
	}

	// Verify connection is closed
	if err := db.Ping(); err == nil {
		t.Error("Expected error after closing connection, got nil")
	}
}

// TestQuery tests basic query execution
func TestQuery(t *testing.T) {
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		t.Skip("DATABASE_URL not set, skipping integration test")
	}

	db, err := postgres.New(dbURL)
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	// Test simple query
	var result int
	err = db.QueryRow("SELECT 1").Scan(&result)
	if err != nil {
		t.Errorf("Failed to execute query: %v", err)
	}
	if result != 1 {
		t.Errorf("Expected result=1, got %d", result)
	}
}

// TestExec tests basic exec operations
func TestExec(t *testing.T) {
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		t.Skip("DATABASE_URL not set, skipping integration test")
	}

	db, err := postgres.New(dbURL)
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	// Create a temporary table
	tableName := "test_table_" + time.Now().Format("20060102150405")
	createSQL := "CREATE TEMP TABLE " + tableName + " (id SERIAL PRIMARY KEY, name TEXT)"

	_, err = db.Exec(createSQL)
	if err != nil {
		t.Errorf("Failed to create temp table: %v", err)
	}

	// Insert data
	insertSQL := "INSERT INTO " + tableName + " (name) VALUES ($1)"
	result, err := db.Exec(insertSQL, "test")
	if err != nil {
		t.Errorf("Failed to insert data: %v", err)
	}

	// Verify rows affected
	rows, err := result.RowsAffected()
	if err != nil {
		t.Errorf("Failed to get rows affected: %v", err)
	}
	if rows != 1 {
		t.Errorf("Expected 1 row affected, got %d", rows)
	}
}

// TestTransaction tests transaction handling
func TestTransaction(t *testing.T) {
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		t.Skip("DATABASE_URL not set, skipping integration test")
	}

	db, err := postgres.New(dbURL)
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	// Start transaction
	tx, err := db.Begin()
	if err != nil {
		t.Fatalf("Failed to begin transaction: %v", err)
	}

	// Create temp table in transaction
	tableName := "test_tx_table_" + time.Now().Format("20060102150405")
	createSQL := "CREATE TEMP TABLE " + tableName + " (id SERIAL PRIMARY KEY, value INT)"

	_, err = tx.Exec(createSQL)
	if err != nil {
		tx.Rollback()
		t.Fatalf("Failed to create table in transaction: %v", err)
	}

	// Insert data
	_, err = tx.Exec("INSERT INTO "+tableName+" (value) VALUES ($1)", 42)
	if err != nil {
		tx.Rollback()
		t.Fatalf("Failed to insert in transaction: %v", err)
	}

	// Commit transaction
	if err := tx.Commit(); err != nil {
		t.Errorf("Failed to commit transaction: %v", err)
	}

	// Verify data exists after commit
	var value int
	err = db.QueryRow("SELECT value FROM " + tableName + " WHERE id = 1").Scan(&value)
	if err != nil {
		t.Errorf("Failed to query after commit: %v", err)
	}
	if value != 42 {
		t.Errorf("Expected value=42, got %d", value)
	}
}

// TestRollback tests transaction rollback
func TestRollback(t *testing.T) {
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		t.Skip("DATABASE_URL not set, skipping integration test")
	}

	db, err := postgres.New(dbURL)
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	// Create temp table outside transaction
	tableName := "test_rollback_" + time.Now().Format("20060102150405")
	createSQL := "CREATE TEMP TABLE " + tableName + " (id SERIAL PRIMARY KEY, value INT)"
	_, err = db.Exec(createSQL)
	if err != nil {
		t.Fatalf("Failed to create table: %v", err)
	}

	// Start transaction
	tx, err := db.Begin()
	if err != nil {
		t.Fatalf("Failed to begin transaction: %v", err)
	}

	// Insert data
	_, err = tx.Exec("INSERT INTO "+tableName+" (value) VALUES ($1)", 100)
	if err != nil {
		tx.Rollback()
		t.Fatalf("Failed to insert: %v", err)
	}

	// Rollback transaction
	if err := tx.Rollback(); err != nil {
		t.Errorf("Failed to rollback transaction: %v", err)
	}

	// Verify data doesn't exist after rollback
	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM " + tableName).Scan(&count)
	if err != nil {
		t.Errorf("Failed to query after rollback: %v", err)
	}
	if count != 0 {
		t.Errorf("Expected count=0 after rollback, got %d", count)
	}
}

// TestConnectionPool tests connection pooling
func TestConnectionPool(t *testing.T) {
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		t.Skip("DATABASE_URL not set, skipping integration test")
	}

	db, err := postgres.New(dbURL)
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	// Get initial stats
	stats := db.Stats()
	initialOpen := stats.OpenConnections

	// Make multiple concurrent queries
	done := make(chan bool, 10)
	for i := 0; i < 10; i++ {
		go func() {
			var result int
			db.QueryRow("SELECT 1").Scan(&result)
			done <- true
		}()
	}

	// Wait for all queries
	for i := 0; i < 10; i++ {
		<-done
	}

	// Check stats after queries
	stats = db.Stats()
	t.Logf("Initial open: %d, After queries: %d, Max: %d",
		initialOpen, stats.OpenConnections, stats.MaxOpenConnections)

	if stats.MaxOpenConnections != 25 {
		t.Errorf("Expected MaxOpenConnections=25, got %d", stats.MaxOpenConnections)
	}
}

// TestPreparedStatement tests prepared statements
func TestPreparedStatement(t *testing.T) {
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		t.Skip("DATABASE_URL not set, skipping integration test")
	}

	db, err := postgres.New(dbURL)
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	// Prepare statement
	stmt, err := db.Prepare("SELECT $1::int + $2::int AS sum")
	if err != nil {
		t.Fatalf("Failed to prepare statement: %v", err)
	}
	defer stmt.Close()

	// Execute prepared statement
	var sum int
	err = stmt.QueryRow(10, 20).Scan(&sum)
	if err != nil {
		t.Errorf("Failed to execute prepared statement: %v", err)
	}
	if sum != 30 {
		t.Errorf("Expected sum=30, got %d", sum)
	}
}

// TestNullValues tests handling of NULL values
func TestNullValues(t *testing.T) {
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		t.Skip("DATABASE_URL not set, skipping integration test")
	}

	db, err := postgres.New(dbURL)
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	// Create temp table
	tableName := "test_null_" + time.Now().Format("20060102150405")
	_, err = db.Exec("CREATE TEMP TABLE " + tableName + " (id SERIAL PRIMARY KEY, value TEXT)")
	if err != nil {
		t.Fatalf("Failed to create table: %v", err)
	}

	// Insert NULL value
	_, err = db.Exec("INSERT INTO " + tableName + " (value) VALUES (NULL)")
	if err != nil {
		t.Errorf("Failed to insert NULL: %v", err)
	}

	// Query NULL value
	var value sql.NullString
	err = db.QueryRow("SELECT value FROM " + tableName + " WHERE id = 1").Scan(&value)
	if err != nil {
		t.Errorf("Failed to query NULL: %v", err)
	}
	if value.Valid {
		t.Errorf("Expected NULL value, got valid value: %s", value.String)
	}
}

// BenchmarkQuery benchmarks simple queries
func BenchmarkQuery(b *testing.B) {
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		b.Skip("DATABASE_URL not set, skipping benchmark")
	}

	db, err := postgres.New(dbURL)
	if err != nil {
		b.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var result int
		db.QueryRow("SELECT 1").Scan(&result)
	}
}

// BenchmarkInsert benchmarks insert operations
func BenchmarkInsert(b *testing.B) {
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		b.Skip("DATABASE_URL not set, skipping benchmark")
	}

	db, err := postgres.New(dbURL)
	if err != nil {
		b.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	// Create temp table
	tableName := "bench_insert_" + time.Now().Format("20060102150405")
	_, err = db.Exec("CREATE TEMP TABLE " + tableName + " (id SERIAL PRIMARY KEY, value INT)")
	if err != nil {
		b.Fatalf("Failed to create table: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		db.Exec("INSERT INTO "+tableName+" (value) VALUES ($1)", i)
	}
}
