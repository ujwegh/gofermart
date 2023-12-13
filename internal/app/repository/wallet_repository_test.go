package repository

import (
	"context"
	"fmt"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/ujwegh/gophermart/internal/app/models"
	"testing"
	"time"
)

const initWalletDB = `
CREATE TABLE IF NOT EXISTS wallets
(
    id INTEGER PRIMARY KEY,
    user_uuid TEXT UNIQUE NOT NULL,
    credits NUMERIC NOT NULL DEFAULT 0,
    debits NUMERIC NOT NULL DEFAULT 0,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CHECK (credits >= 0),
    CHECK (debits >= 0)
);
`

func setupInMemoryWalletDB(t *testing.T) *sqlx.DB {
	db, err := sqlx.Open("sqlite3", "file:memdb1?mode=memory&cache=shared")
	if err != nil {
		t.Fatalf("could not create in-memory db: %v", err)
	}
	_, err = db.Exec(initWalletDB)
	if err != nil {
		t.Fatalf("could not create wallet table: %v", err)
	}
	return db
}

func TestWalletRepositoryImpl_CreateWallet(t *testing.T) {
	db := setupInMemoryWalletDB(t)
	defer db.Close()

	repo := NewWalletRepository(db)

	tests := []struct {
		name    string
		wallet  *models.Wallet
		wantErr bool
	}{
		{
			name: "Successful Wallet Creation",
			wallet: &models.Wallet{
				UserUUID:  uuid.New(),
				Credits:   0,
				Debits:    0,
				CreatedAt: time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
				UpdatedAt: time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tx, err := db.Beginx()
			require.NoError(t, err)

			err = repo.CreateWallet(context.Background(), tx, tt.wallet)
			if tt.wantErr {
				assert.Error(t, err, "CreateWallet should fail")
				assert.NoError(t, tx.Rollback(), "Rollback should succeed")
			} else {
				assert.NoError(t, err, "CreateWallet should not fail")
				assert.NoError(t, tx.Commit(), "Commit should succeed")
				// Verify the wallet record is correctly inserted into the database
				var retrievedWallet models.Wallet
				err := db.Get(&retrievedWallet, "SELECT * FROM wallets WHERE user_uuid = ?", tt.wallet.UserUUID)
				require.NoError(t, err)
				assert.Equal(t, tt.wallet.Credits, retrievedWallet.Credits, "Credits should match")
				assert.Equal(t, tt.wallet.Debits, retrievedWallet.Debits, "Debits should match")
			}
		})
	}
}

func TestWalletRepositoryImpl_Credit(t *testing.T) {
	db := setupInMemoryWalletDB(t)
	defer db.Close()

	userUUID := uuid.New()
	newUserUID := uuid.New()

	initialCredits := 100.0
	creditAmount := 50.0

	// Insert a test wallet into the database for existing user
	_, err := db.Exec(`INSERT INTO wallets (user_uuid, credits, debits) 
					VALUES (?, ?, ?)`, userUUID.String(), initialCredits, 0.0)
	if err != nil {
		panic(fmt.Sprintf("Failed to insert test wallet: %v", err))
	}

	repo := NewWalletRepository(db)

	tests := []struct {
		name              string
		userUUID          *uuid.UUID
		amount            float64
		wantErr           bool
		wantCredits       float64
		shouldCheckWallet bool
	}{
		{
			name:              "Successful Credit Transaction",
			userUUID:          &userUUID,
			amount:            creditAmount,
			wantErr:           false,
			wantCredits:       initialCredits + creditAmount,
			shouldCheckWallet: false,
		},
		{
			name:              "Wallet Not Found for User UUID",
			userUUID:          &newUserUID, // New UUID that has no wallet
			amount:            creditAmount,
			wantErr:           true,
			wantCredits:       0.0,
			shouldCheckWallet: false,
		},
		{
			name:              "Invalid Credit Amount (Negative)",
			userUUID:          &userUUID,
			amount:            -1000.0,
			wantErr:           true,
			wantCredits:       initialCredits, // No change expected
			shouldCheckWallet: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tx, err := db.Beginx()
			require.NoError(t, err)

			wallet, err := repo.Credit(context.Background(), tx, tt.userUUID, tt.amount)
			if tt.wantErr {
				assert.Error(t, err, "Credit should fail")
				assert.NoError(t, tx.Rollback(), "Rollback should succeed")
				if tt.shouldCheckWallet {
					// Verify the wallet record is unchanged
					var wallet models.Wallet
					err := db.Get(&wallet, "SELECT * FROM wallets WHERE user_uuid = ?", tt.userUUID.String())
					require.NoError(t, err)
					assert.Equal(t, initialCredits+creditAmount, wallet.Credits, "Credits should remain unchanged after rollback")
				}
			} else {
				assert.NoError(t, err, "Credit should not fail")
				assert.NoError(t, tx.Commit(), "Commit should succeed")
				assert.Equal(t, tt.wantCredits, wallet.Credits, "Credits after transaction should match expected value")
			}
		})
	}
}

func TestWalletRepositoryImpl_Debit(t *testing.T) {
	db := setupInMemoryWalletDB(t)
	defer db.Close()

	userUUID := uuid.New()
	newUserUID := uuid.New()

	initialCredits := 100.0
	initialDebits := 20.0
	debitAmount := 30.0

	// Insert a test wallet into the database for existing user
	_, err := db.Exec(`INSERT INTO wallets (user_uuid, credits, debits) 
					VALUES (?, ?, ?)`, userUUID.String(), initialCredits, initialDebits)
	if err != nil {
		panic(fmt.Sprintf("Failed to insert test wallet: %v", err))
	}
	repo := NewWalletRepository(db)

	tests := []struct {
		name              string
		userUUID          *uuid.UUID
		amount            float64
		wantErr           bool
		wantDebits        float64
		shouldCheckWallet bool
	}{
		{
			name:              "Successful Debit Transaction",
			userUUID:          &userUUID,
			amount:            debitAmount,
			wantErr:           false,
			wantDebits:        initialDebits + debitAmount,
			shouldCheckWallet: false,
		},
		{
			name:              "Wallet Not Found for User UUID",
			userUUID:          &newUserUID,
			amount:            debitAmount,
			wantErr:           true,
			wantDebits:        0.0,
			shouldCheckWallet: false,
		},
		{
			name:              "Invalid Debit Amount (Negative)",
			userUUID:          &userUUID,
			amount:            -1000.0,
			wantErr:           true,
			wantDebits:        initialDebits, // No change expected
			shouldCheckWallet: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tx, err := db.Beginx()
			require.NoError(t, err)

			wallet, err := repo.Debit(context.Background(), tx, tt.userUUID, tt.amount)
			if tt.wantErr {
				assert.Error(t, err, "Debit should fail")
				assert.NoError(t, tx.Rollback(), "Rollback should succeed")

				if tt.shouldCheckWallet {
					// Verify the wallet record is unchanged
					var wallet models.Wallet
					err := db.Get(&wallet, "SELECT * FROM wallets WHERE user_uuid = ?", tt.userUUID.String())
					require.NoError(t, err)
					assert.Equal(t, initialDebits+debitAmount, wallet.Debits, "Debits should remain unchanged after rollback")
				}
			} else {
				assert.NoError(t, err, "Debit should not fail")
				assert.NoError(t, tx.Commit(), "Commit should succeed")
				assert.Equal(t, tt.wantDebits, wallet.Debits, "Debits after transaction should match expected value")
			}
		})
	}
}

func TestWalletRepositoryImpl_GetWallet(t *testing.T) {
	db := setupInMemoryWalletDB(t)
	defer db.Close()

	// Insert a test wallet into the database
	userUUID := uuid.New()
	newUserUUID := uuid.New()
	testWallet := &models.Wallet{
		UserUUID:  userUUID,
		Credits:   100.0,
		Debits:    0.0,
		CreatedAt: time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
		UpdatedAt: time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
	}
	_, err := db.NamedExec(`INSERT INTO wallets (user_uuid, credits, debits, created_at, updated_at)
							VALUES (:user_uuid, :credits, :debits, :created_at, :updated_at)`, testWallet)
	require.NoError(t, err)
	testWallet.ID = 1
	repo := NewWalletRepository(db)

	tests := []struct {
		name     string
		userUUID *uuid.UUID
		want     *models.Wallet
		wantErr  bool
	}{
		{
			name:     "Wallet Found by User UUID",
			userUUID: &userUUID,
			want:     testWallet,
			wantErr:  false,
		},
		{
			name:     "Wallet Not Found for User UUID",
			userUUID: &newUserUUID,
			want:     nil,
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := repo.GetWallet(context.Background(), tt.userUUID)

			if tt.wantErr {
				assert.Error(t, err, "GetWallet should fail")
			} else {
				assert.NoError(t, err, "GetWallet should not fail")
				assert.Equal(t, tt.want, got, "Expected retrieved wallet to match the test wallet")
			}
		})
	}
}
