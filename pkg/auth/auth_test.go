package auth_test

import (
	"testing"

	"github.com/keneke/delivertrack/pkg/auth/domain"
)

func TestPasswordValidation(t *testing.T) {
	password := "mysecretpassword"

	// Create a user with hashed password
	user, err := domain.NewUser("testuser", "test@example.com", password, domain.RoleCustomer, nil, nil)
	if err != nil {
		t.Fatalf("NewUser failed: %v", err)
	}

	if user.PasswordHash == "" {
		t.Error("Password hash should not be empty")
	}

	// Verify the password
	if err := user.VerifyPassword(password); err != nil {
		t.Error("Password verification should succeed")
	}

	// Verify wrong password fails
	if err := user.VerifyPassword("wrongpassword"); err == nil {
		t.Error("Wrong password should not verify")
	}
}

func TestIsValidRole(t *testing.T) {
	tests := []struct {
		role  string
		valid bool
	}{
		{domain.RoleCustomer, true},
		{domain.RoleCourier, true},
		{domain.RoleAdmin, true},
		{"invalid", false},
	}

	for _, tt := range tests {
		_, err := domain.NewUser("test", "test@example.com", "password", tt.role, nil, nil)
		if (err == nil) != tt.valid {
			t.Errorf("NewUser with role %q validation = %v, want %v", tt.role, err == nil, tt.valid)
		}
	}
}

func TestUserValidation(t *testing.T) {
	tests := []struct {
		name     string
		username string
		email    string
		password string
		role     string
		wantErr  bool
	}{
		{"valid customer", "user1", "user1@example.com", "password123", domain.RoleCustomer, false},
		{"valid courier", "courier1", "courier1@example.com", "password123", domain.RoleCourier, false},
		{"valid admin", "admin1", "admin1@example.com", "password123", domain.RoleAdmin, false},
		{"empty username", "", "test@example.com", "password123", domain.RoleCustomer, true},
		{"empty email", "user1", "", "password123", domain.RoleCustomer, true},
		{"empty password", "user1", "test@example.com", "", domain.RoleCustomer, true},
		{"invalid role", "user1", "test@example.com", "password123", "invalid", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := domain.NewUser(tt.username, tt.email, tt.password, tt.role, nil, nil)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewUser() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
