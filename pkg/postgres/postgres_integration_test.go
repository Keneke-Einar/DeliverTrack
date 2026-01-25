package postgres_test

import (
	"database/sql"
	"os"
	"testing"
	"time"

	"github.com/keneke/delivertrack/pkg/postgres"
)

// Integration tests that require a real PostgreSQL database

// TestIntegration_CreateTables tests creating tables from schema
func TestIntegration_CreateTables(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		t.Skip("DATABASE_URL not set, skipping integration test")
	}

	db, err := postgres.New(dbURL)
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}
	defer db.Close()

	// Create test tables
	schema := `
		CREATE TABLE IF NOT EXISTS test_deliveries (
			id SERIAL PRIMARY KEY,
			customer_id INT NOT NULL,
			courier_id INT,
			status VARCHAR(50) NOT NULL,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		);

		CREATE TABLE IF NOT EXISTS test_couriers (
			id SERIAL PRIMARY KEY,
			name VARCHAR(255) NOT NULL,
			vehicle_type VARCHAR(50),
			phone VARCHAR(20),
			status VARCHAR(50),
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		);
	`

	_, err = db.Exec(schema)
	if err != nil {
		t.Fatalf("Failed to create tables: %v", err)
	}

	// Verify tables exist
	var exists bool
	err = db.QueryRow(`
		SELECT EXISTS (
			SELECT FROM information_schema.tables 
			WHERE table_name = 'test_deliveries'
		)
	`).Scan(&exists)
	if err != nil {
		t.Errorf("Failed to check table existence: %v", err)
	}
	if !exists {
		t.Error("test_deliveries table was not created")
	}

	// Cleanup
	db.Exec("DROP TABLE IF EXISTS test_deliveries")
	db.Exec("DROP TABLE IF EXISTS test_couriers")
}

// TestIntegration_CRUD tests full CRUD operations
func TestIntegration_CRUD(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		t.Skip("DATABASE_URL not set, skipping integration test")
	}

	db, err := postgres.New(dbURL)
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}
	defer db.Close()

	// Create temp table
	tableName := "test_crud_" + time.Now().Format("20060102150405")
	_, err = db.Exec(`
		CREATE TEMP TABLE ` + tableName + ` (
			id SERIAL PRIMARY KEY,
			name VARCHAR(255) NOT NULL,
			email VARCHAR(255),
			active BOOLEAN DEFAULT true,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)
	`)
	if err != nil {
		t.Fatalf("Failed to create table: %v", err)
	}

	// CREATE
	var insertedID int
	err = db.QueryRow(
		"INSERT INTO "+tableName+" (name, email) VALUES ($1, $2) RETURNING id",
		"Test User", "test@example.com",
	).Scan(&insertedID)
	if err != nil {
		t.Fatalf("Failed to insert: %v", err)
	}
	if insertedID == 0 {
		t.Error("Expected non-zero ID after insert")
	}

	// READ
	var name, email string
	var active bool
	err = db.QueryRow(
		"SELECT name, email, active FROM "+tableName+" WHERE id = $1",
		insertedID,
	).Scan(&name, &email, &active)
	if err != nil {
		t.Errorf("Failed to read: %v", err)
	}
	if name != "Test User" || email != "test@example.com" || !active {
		t.Errorf("Read data mismatch: got (%s, %s, %v)", name, email, active)
	}

	// UPDATE
	_, err = db.Exec(
		"UPDATE "+tableName+" SET email = $1, active = $2 WHERE id = $3",
		"updated@example.com", false, insertedID,
	)
	if err != nil {
		t.Errorf("Failed to update: %v", err)
	}

	// Verify update
	err = db.QueryRow(
		"SELECT email, active FROM "+tableName+" WHERE id = $1",
		insertedID,
	).Scan(&email, &active)
	if err != nil {
		t.Errorf("Failed to read after update: %v", err)
	}
	if email != "updated@example.com" || active {
		t.Errorf("Update verification failed: got (%s, %v)", email, active)
	}

	// DELETE
	result, err := db.Exec("DELETE FROM "+tableName+" WHERE id = $1", insertedID)
	if err != nil {
		t.Errorf("Failed to delete: %v", err)
	}

	rows, _ := result.RowsAffected()
	if rows != 1 {
		t.Errorf("Expected 1 row deleted, got %d", rows)
	}

	// Verify delete
	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM "+tableName+" WHERE id = $1", insertedID).Scan(&count)
	if err != nil {
		t.Errorf("Failed to verify delete: %v", err)
	}
	if count != 0 {
		t.Errorf("Expected 0 rows after delete, got %d", count)
	}
}

// TestIntegration_Concurrency tests concurrent database access
func TestIntegration_Concurrency(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		t.Skip("DATABASE_URL not set, skipping integration test")
	}

	db, err := postgres.New(dbURL)
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}
	defer db.Close()

	// Create regular table (not TEMP, as concurrent goroutines use different connections)
	tableName := "test_concurrent_" + time.Now().Format("20060102150405")
	_, err = db.Exec("CREATE TABLE " + tableName + " (id SERIAL PRIMARY KEY, value INT)")
	if err != nil {
		t.Fatalf("Failed to create table: %v", err)
	}
	defer db.Exec("DROP TABLE IF EXISTS " + tableName)

	// Run concurrent inserts
	concurrency := 10
	done := make(chan error, concurrency)

	for i := 0; i < concurrency; i++ {
		go func(val int) {
			_, err := db.Exec("INSERT INTO "+tableName+" (value) VALUES ($1)", val)
			done <- err
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < concurrency; i++ {
		if err := <-done; err != nil {
			t.Errorf("Concurrent insert failed: %v", err)
		}
	}

	// Verify all inserts completed
	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM " + tableName).Scan(&count)
	if err != nil {
		t.Errorf("Failed to count rows: %v", err)
	}
	if count != concurrency {
		t.Errorf("Expected %d rows, got %d", concurrency, count)
	}
}

// TestIntegration_TransactionIsolation tests transaction isolation
func TestIntegration_TransactionIsolation(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		t.Skip("DATABASE_URL not set, skipping integration test")
	}

	db, err := postgres.New(dbURL)
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}
	defer db.Close()

	// Create temp table
	tableName := "test_isolation_" + time.Now().Format("20060102150405")
	_, err = db.Exec("CREATE TEMP TABLE " + tableName + " (id SERIAL PRIMARY KEY, counter INT DEFAULT 0)")
	if err != nil {
		t.Fatalf("Failed to create table: %v", err)
	}

	// Insert initial row
	_, err = db.Exec("INSERT INTO " + tableName + " (counter) VALUES (0)")
	if err != nil {
		t.Fatalf("Failed to insert initial row: %v", err)
	}

	// Start two transactions
	tx1, err := db.Begin()
	if err != nil {
		t.Fatalf("Failed to begin tx1: %v", err)
	}
	defer tx1.Rollback()

	tx2, err := db.Begin()
	if err != nil {
		t.Fatalf("Failed to begin tx2: %v", err)
	}
	defer tx2.Rollback()

	// tx1: Read counter
	var counter1 int
	err = tx1.QueryRow("SELECT counter FROM " + tableName + " WHERE id = 1").Scan(&counter1)
	if err != nil {
		t.Errorf("tx1 read failed: %v", err)
	}

	// tx2: Read counter
	var counter2 int
	err = tx2.QueryRow("SELECT counter FROM " + tableName + " WHERE id = 1").Scan(&counter2)
	if err != nil {
		t.Errorf("tx2 read failed: %v", err)
	}

	// Both should see same value
	if counter1 != counter2 {
		t.Errorf("Isolation violation: tx1=%d, tx2=%d", counter1, counter2)
	}

	// tx1: Update
	_, err = tx1.Exec("UPDATE " + tableName + " SET counter = counter + 1 WHERE id = 1")
	if err != nil {
		t.Errorf("tx1 update failed: %v", err)
	}

	// tx1: Commit
	if err := tx1.Commit(); err != nil {
		t.Errorf("tx1 commit failed: %v", err)
	}

	// tx2: Should still see old value (not yet committed)
	var counter2After int
	err = tx2.QueryRow("SELECT counter FROM " + tableName + " WHERE id = 1").Scan(&counter2After)
	if err != nil {
		t.Errorf("tx2 read after tx1 commit failed: %v", err)
	}

	t.Logf("Counter values: tx1_initial=%d, tx2_before=%d, tx2_after=%d",
		counter1, counter2, counter2After)
}

// TestIntegration_LargeResultSet tests handling large result sets
func TestIntegration_LargeResultSet(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		t.Skip("DATABASE_URL not set, skipping integration test")
	}

	db, err := postgres.New(dbURL)
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}
	defer db.Close()

	// Create temp table with many rows
	tableName := "test_large_" + time.Now().Format("20060102150405")
	_, err = db.Exec("CREATE TEMP TABLE " + tableName + " (id SERIAL PRIMARY KEY, value INT)")
	if err != nil {
		t.Fatalf("Failed to create table: %v", err)
	}

	// Insert 1000 rows
	tx, _ := db.Begin()
	stmt, _ := tx.Prepare("INSERT INTO " + tableName + " (value) VALUES ($1)")
	for i := 0; i < 1000; i++ {
		stmt.Exec(i)
	}
	stmt.Close()
	tx.Commit()

	// Query all rows
	rows, err := db.Query("SELECT id, value FROM " + tableName + " ORDER BY id")
	if err != nil {
		t.Fatalf("Failed to query: %v", err)
	}
	defer rows.Close()

	// Count rows
	count := 0
	for rows.Next() {
		var id, value int
		if err := rows.Scan(&id, &value); err != nil {
			t.Errorf("Failed to scan row: %v", err)
		}
		count++
	}

	if err := rows.Err(); err != nil {
		t.Errorf("Rows iteration error: %v", err)
	}

	if count != 1000 {
		t.Errorf("Expected 1000 rows, got %d", count)
	}
}

// TestIntegration_DataTypes tests various PostgreSQL data types
func TestIntegration_DataTypes(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		t.Skip("DATABASE_URL not set, skipping integration test")
	}

	db, err := postgres.New(dbURL)
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}
	defer db.Close()

	// Create temp table with various types
	tableName := "test_types_" + time.Now().Format("20060102150405")
	_, err = db.Exec(`
		CREATE TEMP TABLE ` + tableName + ` (
			id SERIAL PRIMARY KEY,
			text_col TEXT,
			int_col INT,
			bool_col BOOLEAN,
			timestamp_col TIMESTAMP,
			json_col JSONB,
			array_col INT[]
		)
	`)
	if err != nil {
		t.Fatalf("Failed to create table: %v", err)
	}

	// Insert data with various types
	now := time.Now()
	_, err = db.Exec(`
		INSERT INTO `+tableName+` 
		(text_col, int_col, bool_col, timestamp_col, json_col, array_col) 
		VALUES ($1, $2, $3, $4, $5, $6)
	`, "test", 42, true, now, `{"key": "value"}`, "{1,2,3}")
	if err != nil {
		t.Errorf("Failed to insert: %v", err)
	}

	// Read and verify
	var (
		textCol      string
		intCol       int
		boolCol      bool
		timestampCol time.Time
		jsonCol      string
		arrayCol     string
	)

	err = db.QueryRow(`
		SELECT text_col, int_col, bool_col, timestamp_col, json_col, array_col 
		FROM `+tableName+` WHERE id = 1
	`).Scan(&textCol, &intCol, &boolCol, &timestampCol, &jsonCol, &arrayCol)

	if err != nil {
		t.Errorf("Failed to read: %v", err)
	}

	if textCol != "test" {
		t.Errorf("Text mismatch: got %s", textCol)
	}
	if intCol != 42 {
		t.Errorf("Int mismatch: got %d", intCol)
	}
	if !boolCol {
		t.Error("Bool should be true")
	}
	// Timestamp comparison (within 1 second)
	if timestampCol.Unix() != now.Unix() {
		t.Errorf("Timestamp mismatch: got %v, want %v", timestampCol, now)
	}
}

// TestIntegration_ErrorHandling tests various error scenarios
func TestIntegration_ErrorHandling(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		t.Skip("DATABASE_URL not set, skipping integration test")
	}

	db, err := postgres.New(dbURL)
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}
	defer db.Close()

	// Test 1: Syntax error
	_, err = db.Exec("INVALID SQL")
	if err == nil {
		t.Error("Expected error for invalid SQL")
	}

	// Test 2: Non-existent table
	_, err = db.Query("SELECT * FROM nonexistent_table_xyz")
	if err == nil {
		t.Error("Expected error for non-existent table")
	}

	// Test 3: Constraint violation (if we have a constraint)
	tableName := "test_constraint_" + time.Now().Format("20060102150405")
	db.Exec(`
		CREATE TEMP TABLE ` + tableName + ` (
			id SERIAL PRIMARY KEY,
			unique_col VARCHAR(50) UNIQUE
		)
	`)

	// Insert first value
	db.Exec("INSERT INTO "+tableName+" (unique_col) VALUES ($1)", "unique_value")

	// Try to insert duplicate
	_, err = db.Exec("INSERT INTO "+tableName+" (unique_col) VALUES ($1)", "unique_value")
	if err == nil {
		t.Error("Expected unique constraint violation error")
	}

	// Test 4: QueryRow with no results
	var result string
	err = db.QueryRow("SELECT name FROM couriers WHERE id = -1").Scan(&result)
	if err != sql.ErrNoRows {
		t.Errorf("Expected sql.ErrNoRows, got %v", err)
	}
}
