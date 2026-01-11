// Package repository tests
package repository

import (
	"testing"
	"time"

	"gosveltekit/internal/models"

	"github.com/stretchr/testify/assert"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to connect database: %v", err)
	}

	err = db.AutoMigrate(&models.User{})
	if err != nil {
		t.Fatalf("failed to migrate database: %v", err)
	}

	return db
}

func createTestUser(t *testing.T, db *gorm.DB) *models.User {
	user := &models.User{
		Username:     "testuser",
		Email:        "test@example.com",
		DisplayName:  "Test User",
		PasswordHash: "hashedpassword",
		FirstName:    "Test",
		LastName:     "User",
		Active:       true,
		Role:         "user",
	}

	if err := db.Create(user).Error; err != nil {
		t.Fatalf("failed to create test user: %v", err)
	}

	return user
}

func TestNewUserRepository(t *testing.T) {
	db := setupTestDB(t)
	repo := NewUserRepository(db)
	assert.NotNil(t, repo)
	assert.Equal(t, db, repo.db)
}

func TestUserRepository_FindByID(t *testing.T) {
	db := setupTestDB(t)
	repo := NewUserRepository(db)
	testUser := createTestUser(t, db)

	tests := []struct {
		name    string
		id      uint
		want    *models.User
		wantErr bool
	}{
		{
			name:    "existing user",
			id:      testUser.ID,
			want:    testUser,
			wantErr: false,
		},
		{
			name:    "non-existent user",
			id:      999,
			want:    nil,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := repo.FindByID(tt.id)
			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, got)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.want.ID, got.ID)
				assert.Equal(t, tt.want.Email, got.Email)
				assert.Equal(t, tt.want.Username, got.Username)
			}
		})
	}
}

func TestUserRepository_FindByEmail(t *testing.T) {
	db := setupTestDB(t)
	repo := NewUserRepository(db)
	testUser := createTestUser(t, db)

	tests := []struct {
		name    string
		email   string
		want    *models.User
		wantErr bool
	}{
		{
			name:    "existing email",
			email:   testUser.Email,
			want:    testUser,
			wantErr: false,
		},
		{
			name:    "non-existent email",
			email:   "nonexistent@example.com",
			want:    nil,
			wantErr: true,
		},
		{
			name:    "empty email",
			email:   "",
			want:    nil,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := repo.FindByEmail(tt.email)
			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, got)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.want.Email, got.Email)
			}
		})
	}
}

func TestUserRepository_FindByUsername(t *testing.T) {
	db := setupTestDB(t)
	repo := NewUserRepository(db)
	testUser := createTestUser(t, db)

	tests := []struct {
		name     string
		username string
		want     *models.User
		wantErr  bool
	}{
		{
			name:     "existing username",
			username: testUser.Username,
			want:     testUser,
			wantErr:  false,
		},
		{
			name:     "non-existent username",
			username: "nonexistent",
			want:     nil,
			wantErr:  true,
		},
		{
			name:     "empty username",
			username: "",
			want:     nil,
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := repo.FindByUsername(tt.username)
			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, got)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.want.Username, got.Username)
			}
		})
	}
}

func TestUserRepository_FindByResetToken(t *testing.T) {
	db := setupTestDB(t)
	repo := NewUserRepository(db)
	testUser := createTestUser(t, db)

	// Set reset token
	resetToken := "valid-reset-token"
	testUser.ResetToken = resetToken
	testUser.ResetTokenExpiry = time.Now().Add(24 * time.Hour)
	db.Save(testUser)

	tests := []struct {
		name    string
		token   string
		want    *models.User
		wantErr bool
	}{
		{
			name:    "valid token",
			token:   resetToken,
			want:    testUser,
			wantErr: false,
		},
		{
			name:    "invalid token",
			token:   "invalid-token",
			want:    nil,
			wantErr: true,
		},
		{
			name:    "empty token",
			token:   "",
			want:    nil,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := repo.FindByResetToken(tt.token)
			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, got)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.want.ResetToken, got.ResetToken)
				assert.Equal(t, tt.want.ID, got.ID)
			}
		})
	}
}

func TestUserRepository_FindUsersWithResetTokens(t *testing.T) {
	db := setupTestDB(t)
	repo := NewUserRepository(db)

	// Create users with and without reset tokens
	user1 := createTestUser(t, db)
	user1.ResetToken = "token1"
	db.Save(user1)

	user2 := &models.User{
		Username:     "testuser2",
		Email:        "test2@example.com",
		DisplayName:  "Test User 2",
		PasswordHash: "hashedpassword",
		ResetToken:   "token2",
	}
	db.Create(user2)

	user3 := &models.User{
		Username:     "testuser3",
		Email:        "test3@example.com",
		DisplayName:  "Test User 3",
		PasswordHash: "hashedpassword",
		// No reset token
	}
	db.Create(user3)

	users, err := repo.FindUsersWithResetTokens()
	assert.NoError(t, err)
	assert.Len(t, users, 2)

	// Verify that only users with reset tokens are returned
	hasToken := make(map[string]bool)
	for _, u := range users {
		hasToken[u.ResetToken] = true
	}
	assert.True(t, hasToken["token1"])
	assert.True(t, hasToken["token2"])
}

func TestUserRepository_Create(t *testing.T) {
	db := setupTestDB(t)
	repo := NewUserRepository(db)

	newUser := &models.User{
		Username:     "newuser",
		Email:        "new@example.com",
		DisplayName:  "New User",
		PasswordHash: "hashedpassword",
		FirstName:    "New",
		LastName:     "User",
		Active:       true,
		Role:         "user",
	}

	err := repo.Create(newUser)
	assert.NoError(t, err)
	assert.NotZero(t, newUser.ID)

	// Try to create a user with duplicate email
	duplicateUser := &models.User{
		Username:     "another",
		Email:        "new@example.com", // Same email
		DisplayName:  "Another User",
		PasswordHash: "hashedpassword",
	}
	err = repo.Create(duplicateUser)
	assert.Error(t, err) // Should fail due to unique constraint
}

func TestUserRepository_Update(t *testing.T) {
	db := setupTestDB(t)
	repo := NewUserRepository(db)
	user := createTestUser(t, db)

	// Update user fields
	user.DisplayName = "Updated Name"
	user.Email = "updated@example.com"

	err := repo.Update(user)
	assert.NoError(t, err)

	// Verify the update
	updated, err := repo.FindByID(user.ID)
	assert.NoError(t, err)
	assert.Equal(t, "Updated Name", updated.DisplayName)
	assert.Equal(t, "updated@example.com", updated.Email)

	// Try to update with invalid data
	user.Email = "" // Email should not be empty due to not null constraint
	err = repo.Update(user)
	assert.Error(t, err)
}
