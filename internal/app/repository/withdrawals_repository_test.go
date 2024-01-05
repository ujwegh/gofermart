package repository

import (
	"context"
	"fmt"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

const initWithdrawalDB = `
CREATE TABLE IF NOT EXISTS withdrawals
(
    id INTEGER PRIMARY KEY,
    user_uuid TEXT NOT NULL,
    order_id TEXT NOT NULL,
    amount NUMERIC NOT NULL DEFAULT 0,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CHECK (amount > 0)
);
`

func setupInMemoryWithdrawalDB(t *testing.T) *sqlx.DB {
	db, err := sqlx.Open("sqlite3", "file:memdb1?mode=memory&cache=shared")
	if err != nil {
		t.Fatalf("could not create in-memory db: %v", err)
	}
	_, err = db.Exec(initWithdrawalDB)
	if err != nil {
		t.Fatalf("could not create withdrawal table: %v", err)
	}
	return db
}

func TestWithdrawalsRepositoryImpl_CreateWithdrawal(t *testing.T) {
	db := setupInMemoryWithdrawalDB(t)
	defer db.Close()

	repo := NewWithdrawalsRepository(db)

	tests := []struct {
		name       string
		withdrawal *Withdrawal
		wantErr    bool
	}{
		{
			name: "Successful Withdrawal Creation",
			withdrawal: &Withdrawal{
				UserUUID:  uuid.New(),
				OrderID:   "order123",
				Amount:    100.0,
				CreatedAt: time.Now(),
			},
			wantErr: false,
		},
		{
			name: "Invalid Withdrawal Amount (Negative)",
			withdrawal: &Withdrawal{
				UserUUID:  uuid.New(),
				OrderID:   "order124",
				Amount:    -50.0, // Negative amount, violating the check constraint
				CreatedAt: time.Now(),
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tx, err := db.Beginx()
			require.NoError(t, err)

			err = repo.CreateWithdrawal(context.Background(), tx, tt.withdrawal)
			if tt.wantErr {
				assert.Error(t, err, "CreateWithdrawal should fail")
				assert.NoError(t, tx.Rollback(), "Rollback should succeed")
			} else {
				assert.NoError(t, err, "CreateWithdrawal should not fail")
				assert.NoError(t, tx.Commit(), "Commit should succeed")

				// Verify the withdrawal record is correctly inserted into the database
				var count int
				db.Get(&count, "SELECT COUNT(*) FROM withdrawals WHERE order_id = ?", tt.withdrawal.OrderID)
				assert.Equal(t, 1, count, "Withdrawal should be inserted")
			}
		})
	}
}

func TestWithdrawalsRepositoryImpl_GetWithdrawals(t *testing.T) {
	db := setupInMemoryWithdrawalDB(t)
	defer db.Close()

	userUUID := uuid.New()
	newUserUID := uuid.New()

	repo := NewWithdrawalsRepository(db)

	// Insert test withdrawals into the database
	insertTestWithdrawal(db, userUUID, "order1", 100.0)
	insertTestWithdrawal(db, userUUID, "order2", 50.0)

	tests := []struct {
		name     string
		userUUID *uuid.UUID
		wantLen  int
		wantErr  bool
	}{
		{
			name:     "Successful Retrieval of Withdrawals for User",
			userUUID: &userUUID,
			wantLen:  2,
			wantErr:  false,
		},
		{
			name:     "No Withdrawals Found for User",
			userUUID: &newUserUID,
			wantLen:  0,
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := repo.GetWithdrawals(context.Background(), tt.userUUID)

			if tt.wantErr {
				assert.Error(t, err, "GetWithdrawals should fail")
			} else {
				assert.NoError(t, err, "GetWithdrawals should not fail")
				assert.Len(t, *got, tt.wantLen, "Unexpected number of withdrawals retrieved")
			}
		})
	}
}

func insertTestWithdrawal(db *sqlx.DB, userUUID uuid.UUID, orderID string, amount float64) {

	_, err := db.Exec(`INSERT INTO withdrawals (user_uuid, order_id, amount) VALUES (?, ?, ?)`, userUUID.String(), orderID, amount)
	if err != nil {
		panic(fmt.Sprintf("Failed to insert test withdrawal: %v", err))
	}
}
