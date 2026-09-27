package store

import (
	"testing"
	"time"

	"github.com/puppe1990/grato/internal/models"
	"github.com/puppe1990/grato/internal/testdata"
)

func newTestStore(t *testing.T) *SQLiteStore {
	t.Helper()
	s, err := NewSQLiteStore(":memory:", "test")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = s.Close() })
	return s
}

func TestStore_Migrations(t *testing.T) {
	s := newTestStore(t)

	for _, table := range []string{"users", "sessions", "moments", "tags", "moment_tags", "rituals"} {
		var name string
		err := s.db.QueryRow("SELECT name FROM sqlite_master WHERE type='table' AND name=?", table).Scan(&name)
		if err != nil {
			t.Errorf("table %q not found: %v", table, err)
		}
	}
}

func TestStore_FindUserByEmail(t *testing.T) {
	s, f := seededStore(t)
	user := mustUser(t, s, f)

	found, err := s.FindUserByEmail(user.Email)
	if err != nil {
		t.Fatalf("FindUserByEmail: %v", err)
	}
	if found.ID != user.ID {
		t.Errorf("ID = %d, want %d", found.ID, user.ID)
	}
}

func TestStore_FindUserByID_missingIsNotFound(t *testing.T) {
	s := newTestStore(t)

	if _, err := s.FindUserByID(4242); err != ErrNotFound {
		t.Errorf("err = %v, want ErrNotFound", err)
	}
}

func TestStore_CreateUser_rejectsDuplicateEmail(t *testing.T) {
	s, f := seededStore(t)
	user := testdata.User(f)
	user.PasswordHash = "hash"
	if _, err := s.CreateUser(user); err != nil {
		t.Fatalf("CreateUser: %v", err)
	}

	if _, err := s.CreateUser(user); err != ErrEmailTaken {
		t.Errorf("err = %v, want ErrEmailTaken", err)
	}
}

func TestStore_MomentDays_parsesLocalDates(t *testing.T) {
	s, f := seededStore(t)
	user := mustUser(t, s, f)
	day := time.Date(2026, time.March, 7, 15, 0, 0, 0, time.Local)
	mustMoment(t, s, f, user.ID, day)

	days, err := s.MomentDays(user.ID)
	if err != nil {
		t.Fatalf("MomentDays: %v", err)
	}
	if len(days) != 1 {
		t.Fatalf("len(days) = %d, want 1", len(days))
	}
	if got := days[0].Format("2006-01-02"); got != "2026-03-07" {
		t.Errorf("day = %q, want 2026-03-07", got)
	}
}

func TestStore_MomentsBetween_spansInclusiveRange(t *testing.T) {
	s, f := seededStore(t)
	user := mustUser(t, s, f)
	october := time.Date(2026, time.October, 1, 0, 0, 0, 0, time.Local)
	mustMoment(t, s, f, user.ID, october.AddDate(0, 0, 4))
	mustMoment(t, s, f, user.ID, october.AddDate(0, 0, 20))
	mustMoment(t, s, f, user.ID, october.AddDate(0, -1, 0))

	moments, err := s.MomentsBetween(user.ID, october, october.AddDate(0, 1, -1))
	if err != nil {
		t.Fatalf("MomentsBetween: %v", err)
	}
	if len(moments) != 2 {
		t.Fatalf("len(moments) = %d, want 2 (September excluded)", len(moments))
	}
}

func TestStore_MomentsBetween_ordersOldestFirst(t *testing.T) {
	s, f := seededStore(t)
	user := mustUser(t, s, f)
	base := time.Date(2026, time.October, 1, 0, 0, 0, 0, time.Local)
	for _, offset := range []int{9, 2, 17} {
		mustMoment(t, s, f, user.ID, base.AddDate(0, 0, offset))
	}

	moments, err := s.MomentsBetween(user.ID, base, base.AddDate(0, 1, -1))
	if err != nil {
		t.Fatalf("MomentsBetween: %v", err)
	}
	if len(moments) != 3 {
		t.Fatalf("len(moments) = %d, want 3", len(moments))
	}
	for i := 1; i < len(moments); i++ {
		if moments[i].HappenedAt.Before(moments[i-1].HappenedAt) {
			t.Fatalf("moments not oldest first: %v then %v", moments[i-1].HappenedAt, moments[i].HappenedAt)
		}
	}
}

func TestStore_clampsSerenityToTheScale(t *testing.T) {
	s, f := seededStore(t)
	user := mustUser(t, s, f)
	moment := testdata.MomentAt(f, user.ID, time.Now())
	moment.Serenity = 99

	id, err := s.InsertMoment(moment)
	if err != nil {
		t.Fatalf("InsertMoment: %v", err)
	}
	got, err := s.FindMoment(id)
	if err != nil {
		t.Fatalf("FindMoment: %v", err)
	}
	if got.Serenity != models.SerenityMax {
		t.Errorf("Serenity = %d, want %d", got.Serenity, models.SerenityMax)
	}
}
