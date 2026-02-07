package adapters

import (
	"context"
	"database/sql"

	"github.com/Keneke-Einar/delivertrack/pkg/auth/domain"
)

// PostgresUserRepository implements the UserRepository interface using PostgreSQL
type PostgresUserRepository struct {
	db *sql.DB
}

// NewPostgresUserRepository creates a new PostgreSQL user repository
func NewPostgresUserRepository(db *sql.DB) *PostgresUserRepository {
	return &PostgresUserRepository{db: db}
}

// Create stores a new user, auto-creating a customer/courier profile if needed
func (r *PostgresUserRepository) Create(ctx context.Context, user *domain.User) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// If role is customer/courier and no linked profile, create one
	if user.Role == "customer" && user.CustomerID == nil {
		var custID int
		err := tx.QueryRowContext(ctx,
			`INSERT INTO customers (name, address, contact, email) VALUES ($1, $2, $3, $4) RETURNING id`,
			user.Username, "N/A", "N/A", user.Email,
		).Scan(&custID)
		if err != nil {
			return err
		}
		user.CustomerID = &custID
	}
	if user.Role == "courier" && user.CourierID == nil {
		var courID int
		err := tx.QueryRowContext(ctx,
			`INSERT INTO couriers (name, vehicle_type, phone) VALUES ($1, $2, $3) RETURNING id`,
			user.Username, "bicycle", "N/A",
		).Scan(&courID)
		if err != nil {
			return err
		}
		user.CourierID = &courID
	}

	query := `
		INSERT INTO users (username, email, password_hash, role, customer_id, courier_id, active)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, created_at, updated_at
	`

	var customerID, courierID sql.NullInt64
	if user.CustomerID != nil {
		customerID = sql.NullInt64{Int64: int64(*user.CustomerID), Valid: true}
	}
	if user.CourierID != nil {
		courierID = sql.NullInt64{Int64: int64(*user.CourierID), Valid: true}
	}

	err = tx.QueryRowContext(
		ctx,
		query,
		user.Username,
		user.Email,
		user.PasswordHash,
		user.Role,
		customerID,
		courierID,
		user.Active,
	).Scan(&user.ID, &user.CreatedAt, &user.UpdatedAt)
	if err != nil {
		return err
	}

	return tx.Commit()
}

// GetByID retrieves a user by ID
func (r *PostgresUserRepository) GetByID(ctx context.Context, id int) (*domain.User, error) {
	query := `
		SELECT id, username, email, password_hash, role, customer_id, courier_id, active, created_at, updated_at
		FROM users
		WHERE id = $1
	`

	var user domain.User
	var customerID, courierID sql.NullInt64

	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&user.ID,
		&user.Username,
		&user.Email,
		&user.PasswordHash,
		&user.Role,
		&customerID,
		&courierID,
		&user.Active,
		&user.CreatedAt,
		&user.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, domain.ErrUserNotFound
	}
	if err != nil {
		return nil, err
	}

	if customerID.Valid {
		cid := int(customerID.Int64)
		user.CustomerID = &cid
	}
	if courierID.Valid {
		cid := int(courierID.Int64)
		user.CourierID = &cid
	}

	return &user, nil
}

// GetByUsername retrieves a user by username
func (r *PostgresUserRepository) GetByUsername(ctx context.Context, username string) (*domain.User, error) {
	query := `
		SELECT id, username, email, password_hash, role, customer_id, courier_id, active, created_at, updated_at
		FROM users
		WHERE username = $1
	`

	var user domain.User
	var customerID, courierID sql.NullInt64

	err := r.db.QueryRowContext(ctx, query, username).Scan(
		&user.ID,
		&user.Username,
		&user.Email,
		&user.PasswordHash,
		&user.Role,
		&customerID,
		&courierID,
		&user.Active,
		&user.CreatedAt,
		&user.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, domain.ErrUserNotFound
	}
	if err != nil {
		return nil, err
	}

	if customerID.Valid {
		cid := int(customerID.Int64)
		user.CustomerID = &cid
	}
	if courierID.Valid {
		cid := int(courierID.Int64)
		user.CourierID = &cid
	}

	return &user, nil
}

// GetByEmail retrieves a user by email
func (r *PostgresUserRepository) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	query := `
		SELECT id, username, email, password_hash, role, customer_id, courier_id, active, created_at, updated_at
		FROM users
		WHERE email = $1
	`

	var user domain.User
	var customerID, courierID sql.NullInt64

	err := r.db.QueryRowContext(ctx, query, email).Scan(
		&user.ID,
		&user.Username,
		&user.Email,
		&user.PasswordHash,
		&user.Role,
		&customerID,
		&courierID,
		&user.Active,
		&user.CreatedAt,
		&user.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, domain.ErrUserNotFound
	}
	if err != nil {
		return nil, err
	}

	if customerID.Valid {
		cid := int(customerID.Int64)
		user.CustomerID = &cid
	}
	if courierID.Valid {
		cid := int(courierID.Int64)
		user.CourierID = &cid
	}

	return &user, nil
}

// Update updates a user
func (r *PostgresUserRepository) Update(ctx context.Context, user *domain.User) error {
	query := `
		UPDATE users
		SET username = $1, email = $2, password_hash = $3, role = $4, 
		    customer_id = $5, courier_id = $6, active = $7, updated_at = CURRENT_TIMESTAMP
		WHERE id = $8
		RETURNING updated_at
	`

	var customerID, courierID sql.NullInt64
	if user.CustomerID != nil {
		customerID = sql.NullInt64{Int64: int64(*user.CustomerID), Valid: true}
	}
	if user.CourierID != nil {
		courierID = sql.NullInt64{Int64: int64(*user.CourierID), Valid: true}
	}

	err := r.db.QueryRowContext(
		ctx,
		query,
		user.Username,
		user.Email,
		user.PasswordHash,
		user.Role,
		customerID,
		courierID,
		user.Active,
		user.ID,
	).Scan(&user.UpdatedAt)

	if err == sql.ErrNoRows {
		return domain.ErrUserNotFound
	}
	return err
}

// Delete deletes a user
func (r *PostgresUserRepository) Delete(ctx context.Context, id int) error {
	query := `DELETE FROM users WHERE id = $1`

	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return err
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rows == 0 {
		return domain.ErrUserNotFound
	}

	return nil
}
