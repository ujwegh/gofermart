package repository

import (
	"context"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/ujwegh/gophermart/internal/app/models"
	"testing"
	"time"
)

const initUserDB = `
CREATE TABLE IF NOT EXISTS users
(
    uuid          TEXT PRIMARY KEY DEFAULT (hex(randomblob(16))),
    login         TEXT UNIQUE NOT NULL,
    password_hash TEXT NOT NULL,
    created_at    TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);
`

func setupInMemoryUserDB(t *testing.T) *sqlx.DB {
	db, err := sqlx.Open("sqlite3", "file:memdb1?mode=memory&cache=shared")
	if err != nil {
		t.Fatalf("could not create in-memory db: %v", err)
	}
	_, err = db.Exec(initUserDB)
	if err != nil {
		t.Fatalf("could not create user table: %v", err)
	}
	return db
}

func TestUserRepositoryImpl_Create(t *testing.T) {
	db := setupInMemoryUserDB(t)
	defer db.Close()

	repo := NewUserRepository(db)

	tests := []struct {
		name    string
		user    *models.User
		wantErr bool
	}{
		{
			name: "Successful User Creation",
			user: &models.User{
				UUID:         uuid.New(),
				Login:        "newuser",
				PasswordHash: "hash",
				CreatedAt:    time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
			},
			wantErr: false,
		},
		{
			name: "User Creation with Duplicate Login",
			user: &models.User{
				UUID:         uuid.New(),
				Login:        "newuser", // Same login as above
				PasswordHash: "hash",
				CreatedAt:    time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
			},
			wantErr: true, // Expect an error due to unique constraint violation
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tx, err := db.Beginx()
			require.NoError(t, err)

			err = repo.Create(context.Background(), tx, tt.user)
			if tt.wantErr {
				assert.Error(t, err, "Create should fail")
				assert.NoError(t, tx.Rollback(), "Rollback should succeed")
			} else {
				assert.NoError(t, err, "Create should not fail")
				assert.NoError(t, tx.Commit(), "Commit should succeed")
			}
		})
	}
}

func TestUserRepositoryImpl_FindByLogin(t *testing.T) {
	db := setupInMemoryUserDB(t)
	defer db.Close()

	// Insert a test user into the database
	testUser := &models.User{
		UUID:         uuid.New(),
		Login:        "testuser",
		PasswordHash: "hash",
		CreatedAt:    time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
	}
	_, err := db.NamedExec(`INSERT INTO users (uuid, login, password_hash, created_at)
							VALUES (:uuid, :login, :password_hash, :created_at)`, testUser)
	require.NoError(t, err)

	repo := NewUserRepository(db)

	tests := []struct {
		name    string
		login   string
		want    *models.User
		wantErr bool
	}{
		{
			name:    "User Found by Login",
			login:   "testuser",
			want:    testUser,
			wantErr: false,
		},
		{
			name:    "User Not Found by Login",
			login:   "nonexistent",
			want:    nil,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := repo.FindByLogin(context.Background(), tt.login)

			if tt.wantErr {
				assert.Error(t, err, "FindByLogin should fail")
			} else {
				assert.NoError(t, err, "FindByLogin should not fail")
				assert.Equal(t, tt.want, got, "Expected retrieved user to match the test user")
			}
		})
	}
}
