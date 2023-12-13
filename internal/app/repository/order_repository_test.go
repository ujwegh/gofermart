package repository

import (
	"context"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/ujwegh/gophermart/internal/app/models"
	"testing"
	"time"
)

const initOrderDB = `
CREATE TABLE IF NOT EXISTS orders
(
    id VARCHAR PRIMARY KEY,
    user_uuid VARCHAR NOT NULL,
    status TEXT NOT NULL DEFAULT 'NEW',
    accrual NUMERIC,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CHECK (accrual > 0)
);
`

func setupInMemoryOrderDB(t *testing.T) *sqlx.DB {
	db, err := sqlx.Open("sqlite3", "file:memdb1?mode=memory&cache=shared")
	if err != nil {
		t.Fatalf("could not create in-memory db: %v", err)
	}
	_, err = db.Exec(initOrderDB)
	if err != nil {
		t.Fatalf("could not create order table: %v", err)
	}
	return db
}

func TestOrderRepositoryImpl_CountUnprocessedOrders(t *testing.T) {
	db := setupInMemoryOrderDB(t)
	defer db.Close()

	// Insert test orders into the database
	statuses := []string{"NEW", "PROCESSING", "FINISHED"}
	for _, status := range statuses {
		_, err := db.Exec(`INSERT INTO orders (id, user_uuid, status, created_at, updated_at) 
			VALUES (?, ?, ?, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)`, uuid.New().String(), uuid.New().String(), status)
		require.NoError(t, err)
	}

	repo := NewOrderRepository(db)

	tests := []struct {
		name      string
		wantCount int
		wantErr   bool
	}{
		{
			name:      "Count Unprocessed Orders",
			wantCount: 2, // Expecting 2 unprocessed orders ("NEW" and "PROCESSING")
			wantErr:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := repo.CountUnprocessedOrders()

			if tt.wantErr {
				assert.Error(t, err, "CountUnprocessedOrders should fail")
			} else {
				assert.NoError(t, err, "CountUnprocessedOrders should not fail")
				assert.Equal(t, tt.wantCount, got, "Unexpected count of unprocessed orders")
			}
		})
	}
}

func TestOrderRepositoryImpl_CreateOrder(t *testing.T) {
	db := setupInMemoryOrderDB(t)
	defer db.Close()

	tests := []struct {
		name    string
		order   *models.Order
		wantErr bool
	}{
		{
			name: "Successful Order Creation",
			order: &models.Order{
				ID:        "order-uuid",
				UserUUID:  uuid.New(),
				Status:    models.NEW,
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := NewOrderRepository(db)

			err := repo.CreateOrder(context.Background(), tt.order)
			if tt.wantErr {
				assert.Error(t, err, "CreateOrder should fail")
			} else {
				assert.NoError(t, err, "CreateOrder should not fail")
			}

			if !tt.wantErr {
				var count int
				err := db.Get(&count, "SELECT COUNT(*) FROM orders WHERE id = ?", tt.order.ID)
				require.NoError(t, err)
				assert.Equal(t, 1, count, "Order should be inserted")
			}
		})
	}
}

func TestOrderRepositoryImpl_GetOrderByID(t *testing.T) {
	db := setupInMemoryOrderDB(t)
	defer db.Close()

	var acc = 10.0
	// Insert a test order into the database for retrieval
	testOrder := &models.Order{
		ID:        "test-order-uuid",
		UserUUID:  uuid.New(),
		Status:    "NEW",
		Accrual:   &acc,
		CreatedAt: time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
		UpdatedAt: time.Date(2021, 1, 2, 0, 0, 0, 0, time.UTC),
	}
	_, err := db.NamedExec(`INSERT INTO orders (id, user_uuid, status, accrual, created_at, updated_at) 
							VALUES (:id, :user_uuid, :status, :accrual, :created_at, :updated_at)`, testOrder)
	require.NoError(t, err)

	repo := NewOrderRepository(db)

	tests := []struct {
		name    string
		orderID string
		want    *models.Order
		wantErr bool
	}{
		{
			name:    "Successful Order Retrieval by ID",
			orderID: "test-order-uuid",
			want:    testOrder,
			wantErr: false,
		},
		{
			name:    "Order Not Found by ID",
			orderID: "non-existent-uuid",
			want:    nil,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := repo.GetOrderByID(context.Background(), tt.orderID)

			if tt.wantErr {
				assert.Error(t, err, "GetOrderByID should fail for non-existent ID")
				assert.Nil(t, got, "Expected no order to be returned")
			} else {
				assert.NoError(t, err, "GetOrderByID should not fail for existing ID")
				assert.Equal(t, tt.want, got, "Expected retrieved order to match the test order")
			}
		})
	}
}

func TestOrderRepositoryImpl_GetOrdersByUserUID(t *testing.T) {
	db := setupInMemoryOrderDB(t)
	defer db.Close()

	userUUID := uuid.New()
	newUserUUID := uuid.New()
	var acc = 10.0
	// Insert test orders for the user into the database
	testOrders := []models.Order{
		{
			ID:        "order1",
			UserUUID:  userUUID,
			Status:    "NEW",
			Accrual:   &acc,
			CreatedAt: time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
			UpdatedAt: time.Date(2021, 1, 2, 0, 0, 0, 0, time.UTC),
		},
	}
	for _, order := range testOrders {
		_, err := db.NamedExec(`INSERT INTO orders (id, user_uuid, status, accrual, created_at, updated_at) 
								VALUES (:id, :user_uuid, :status, :accrual, :created_at, :updated_at)`, order)
		require.NoError(t, err)
	}

	repo := NewOrderRepository(db)

	tests := []struct {
		name     string
		userUUID *uuid.UUID
		want     *[]models.Order
		wantErr  bool
	}{
		{
			name:     "Successful Retrieval of Orders for User",
			userUUID: &userUUID,
			want:     &testOrders,
			wantErr:  false,
		},
		{
			name:     "No Orders Found for User",
			userUUID: &newUserUUID, // A new UUID that has no orders
			want:     &[]models.Order{},
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := repo.GetOrdersByUserUID(context.Background(), tt.userUUID)

			if tt.wantErr {
				assert.Error(t, err, "GetOrdersByUserUID should fail")
			} else {
				assert.NoError(t, err, "GetOrdersByUserUID should not fail")
				assert.Equal(t, tt.want, got, "Expected retrieved orders to match")
			}
		})
	}
}

func TestOrderRepositoryImpl_GetUnprocessedOrders(t *testing.T) {
	db := setupInMemoryOrderDB(t)
	defer db.Close()

	// Insert test orders into the database
	for _, status := range []string{"NEW", "PROCESSING", "FINISHED"} {
		_, err := db.Exec(`INSERT INTO orders (id, user_uuid, status, created_at, updated_at) 
			VALUES (?, ?, ?, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)`, uuid.New().String(), uuid.New().String(), status)
		require.NoError(t, err)
	}

	repo := NewOrderRepository(db)

	tests := []struct {
		name    string
		limit   int
		offset  int
		wantLen int // Expected number of orders returned
		wantErr bool
	}{
		{
			name:    "Retrieve First Batch of Unprocessed Orders",
			limit:   2,
			offset:  0,
			wantLen: 2, // Expecting 2 unprocessed orders
			wantErr: false,
		},
		{
			name:    "Retrieve Second Batch of Unprocessed Orders",
			limit:   1,
			offset:  1,
			wantLen: 1, // Expecting 1 more unprocessed order
			wantErr: false,
		},
		{
			name:    "No Unprocessed Orders Found",
			limit:   1,
			offset:  10, // Offset beyond the range of available orders
			wantLen: 0,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := repo.GetUnprocessedOrders(tt.limit, tt.offset)

			if tt.wantErr {
				assert.Error(t, err, "GetUnprocessedOrders should fail")
			} else {
				assert.NoError(t, err, "GetUnprocessedOrders should not fail")
				assert.Len(t, *got, tt.wantLen, "Unexpected number of orders retrieved")
			}
		})
	}
}

func TestOrderRepositoryImpl_UpdateOrder(t *testing.T) {
	db := setupInMemoryOrderDB(t)
	defer db.Close()

	var acc = 10.0
	var newAcc = 20.0
	var newDate = time.Date(2021, 1, 3, 0, 0, 0, 0, time.UTC)
	// Insert a test order into the database for update
	testOrder := &models.Order{
		ID:        "order-uuid",
		UserUUID:  uuid.New(),
		Status:    "NEW",
		Accrual:   &acc,
		CreatedAt: time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
		UpdatedAt: time.Date(2021, 1, 2, 0, 0, 0, 0, time.UTC),
	}
	_, err := db.NamedExec(`INSERT INTO orders (id, user_uuid, status, accrual, created_at, updated_at) VALUES (:id, :user_uuid, :status, :accrual, :created_at, :updated_at)`, testOrder)
	require.NoError(t, err)

	repo := NewOrderRepository(db)

	tests := []struct {
		name    string
		order   *models.Order
		wantErr bool
	}{
		{
			name: "Successful Order Update",
			order: &models.Order{
				ID:        "order-uuid",
				Status:    "UPDATED",
				Accrual:   &newAcc,
				UpdatedAt: newDate,
			},
			wantErr: false,
		},
		// Additional test cases for failure scenarios...
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tx, err := db.Beginx()
			require.NoError(t, err)

			err = repo.UpdateOrder(context.Background(), tx, tt.order)
			if tt.wantErr {
				assert.Error(t, err, "UpdateOrder should fail")
				assert.NoError(t, tx.Rollback(), "Rollback should succeed")
			} else {
				assert.NoError(t, err, "UpdateOrder should not fail")
				assert.NoError(t, tx.Commit(), "Commit should succeed")

				// Verify the order was updated correctly
				var updatedOrder models.Order
				err := db.Get(&updatedOrder, "SELECT * FROM orders WHERE id = ?", tt.order.ID)
				require.NoError(t, err)
				assert.Equal(t, tt.order.Status, updatedOrder.Status, "Order status should be updated")
				assert.Equal(t, tt.order.Accrual, updatedOrder.Accrual, "Order accrual should be updated")
			}
		})
	}
}
