package postgres

import (
	"context"
	"errors"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"

	"github.com/go-park-mail-ru/2026_1_KISS/internal/domain"
	paymentdomain "github.com/go-park-mail-ru/2026_1_KISS/internal/payment"
)

func TestPaymentRepo_Create(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	repo := NewPaymentRepository(db)

	mock.ExpectQuery(`INSERT INTO payments`).
		WithArgs(int64(42), int64(1), sqlmock.AnyArg(), "pending", int64(99900), "RUB", sqlmock.AnyArg(), "key-1", "Pro").
		WillReturnRows(sqlmock.NewRows([]string{"id", "created_at", "updated_at"}).
			AddRow("uuid-1", anyTime(), anyTime()))

	p := &paymentdomain.Payment{
		UserID:         42,
		PlanID:         1,
		Status:         "pending",
		AmountKopeks:   99900,
		Currency:       "RUB",
		IdempotenceKey: "key-1",
		Description:    "Pro",
	}
	if err := repo.Create(context.Background(), p); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if p.ID != "uuid-1" {
		t.Errorf("expected uuid-1, got %s", p.ID)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet: %v", err)
	}
}

func TestPaymentRepo_GetByID_NotFound(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	repo := NewPaymentRepository(db)
	mock.ExpectQuery(`SELECT .+ FROM payments WHERE id`).
		WithArgs("missing").
		WillReturnError(sqlNoRows())

	_, err = repo.GetByID(context.Background(), "missing")
	if !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestPaymentRepo_MarkPaid(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	repo := NewPaymentRepository(db)
	mock.ExpectExec(`UPDATE payments SET status = 'succeeded'`).
		WithArgs("uuid-1").
		WillReturnResult(sqlmock.NewResult(0, 1))

	if err := repo.MarkPaid(context.Background(), "uuid-1"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
