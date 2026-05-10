package postgres

import (
	"context"
	"errors"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"

	"github.com/go-park-mail-ru/2026_1_KISS/internal/domain"
)

func TestPlanRepo_GetByName_Found(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	repo := NewPlanRepository(db)
	mock.ExpectQuery(`SELECT .+ FROM subscription_plans WHERE name`).
		WithArgs("pro").
		WillReturnRows(sqlmock.NewRows([]string{"id", "name", "price", "execution_quota", "duration_days", "created_at"}).
			AddRow(1, "pro", 99900, 100000, 30, anyTime()))

	plan, err := repo.GetByName(context.Background(), "pro")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if plan.Name != "pro" || plan.PriceKopeks != 99900 {
		t.Fatalf("unexpected plan: %+v", plan)
	}
}

func TestPlanRepo_GetByName_NotFound(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	repo := NewPlanRepository(db)
	mock.ExpectQuery(`SELECT .+ FROM subscription_plans WHERE name`).
		WithArgs("missing").
		WillReturnError(sqlNoRows())

	_, err = repo.GetByName(context.Background(), "missing")
	if !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestPlanRepo_List(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	repo := NewPlanRepository(db)
	mock.ExpectQuery(`SELECT .+ FROM subscription_plans ORDER BY price`).
		WillReturnRows(sqlmock.NewRows([]string{"id", "name", "price", "execution_quota", "duration_days", "created_at"}).
			AddRow(1, "pro", 99900, 100000, 30, anyTime()).
			AddRow(2, "max", 199900, 999999, 30, anyTime()))

	plans, err := repo.List(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(plans) != 2 {
		t.Fatalf("expected 2 plans, got %d", len(plans))
	}
	if plans[0].Name != "pro" || plans[1].Name != "max" {
		t.Errorf("unexpected order: %+v", plans)
	}
}
