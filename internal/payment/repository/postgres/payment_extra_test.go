package postgres

import (
	"context"
	"errors"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"

	"github.com/go-park-mail-ru/2026_1_KISS/internal/domain"
)

func TestPaymentRepo_GetByYooKassaID_Found(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	repo := NewPaymentRepository(db)
	mock.ExpectQuery(`SELECT .+ FROM payments WHERE yookassa_payment_id`).
		WithArgs("yk-001").
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "user_id", "plan_id", "yookassa_payment_id", "status", "amount_kopeks",
			"currency", "confirmation_token", "idempotence_key", "description",
			"created_at", "updated_at", "paid_at",
		}).AddRow("uuid-1", int64(42), int64(1), "yk-001", "pending", int64(99900),
			"RUB", "ct-x", "key-1", "desc", anyTime(), anyTime(), nil))

	p, err := repo.GetByYooKassaID(context.Background(), "yk-001")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if p.YooKassaPaymentID != "yk-001" {
		t.Errorf("unexpected yk id: %s", p.YooKassaPaymentID)
	}
}

func TestPaymentRepo_GetByYooKassaID_NotFound(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	repo := NewPaymentRepository(db)
	mock.ExpectQuery(`SELECT .+ FROM payments WHERE yookassa_payment_id`).
		WithArgs("missing").
		WillReturnError(sqlNoRows())

	_, err = repo.GetByYooKassaID(context.Background(), "missing")
	if !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestPaymentRepo_UpdateStatus(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	repo := NewPaymentRepository(db)
	mock.ExpectExec(`UPDATE payments SET status`).
		WithArgs("canceled", sqlmock.AnyArg(), "uuid-1").
		WillReturnResult(sqlmock.NewResult(0, 1))

	if err := repo.UpdateStatus(context.Background(), "uuid-1", "canceled", ""); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
