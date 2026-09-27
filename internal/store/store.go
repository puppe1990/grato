package store

import (
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/puppe1990/amarra-cais/pkg/cais/devlog"
	"github.com/puppe1990/amarra-cais/pkg/cais/session"
	caissqlite "github.com/puppe1990/amarra-cais/pkg/cais/sqlite"
	"github.com/puppe1990/amarra-cais/pkg/cais/sqllog"
	_ "modernc.org/sqlite"

	"github.com/puppe1990/grato/internal/models"
)

var (
	ErrEmailTaken = errors.New("email already registered")
	ErrNotFound   = errors.New("not found")
)

// Store is the persistence port the handlers depend on. Everything user-owned
// is scoped by userID so one account can never read another's entries.
type Store interface {
	FindUserByEmail(email string) (models.User, error)
	FindUserByID(id int64) (models.User, error)
	CreateUser(user models.User) (int64, error)
	UpdateUserProfile(id int64, name string, intention models.Intention) error

	CreatePasswordResetToken(userID int64) (string, error)
	FindPasswordResetUserID(token string) (int64, bool)
	ResetPasswordWithToken(token, passwordHash string) error

	InsertMoment(m models.Moment) (int64, error)
	FindMoment(id int64) (models.Moment, error)
	FindMomentForUser(id, userID int64) (models.Moment, error)
	ListMoments(userID int64, limit int) ([]models.Moment, error)
	MomentsOnDay(userID int64, day time.Time) ([]models.Moment, error)
	MomentsBetween(userID int64, from, to time.Time) ([]models.Moment, error)
	MomentDays(userID int64) ([]time.Time, error)
	CountMoments(userID int64) (int64, error)
	CountMomentsAt(userID int64, ofDay models.MomentOfDay) (int64, error)
	CountMomentDaysInYear(userID int64, year int) (int64, error)

	AttachTags(momentID, userID int64, names []string) error
	TagsForUser(userID int64) ([]models.Tag, error)
	TagStats(userID int64, limit int) ([]models.TagStat, error)

	FindRitual(userID int64, kind string) (models.Ritual, error)
	SaveRitual(r models.Ritual) error

	Sessions() session.Store
	Ping() error
	DB() *sql.DB
	Close() error
}

type SQLiteStore struct {
	db *sqllog.DB
}

func NewSQLiteStore(dsn string, env string) (*SQLiteStore, error) {
	if dsn != ":memory:" {
		dir := filepath.Dir(dsn)
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return nil, fmt.Errorf("create db dir: %w", err)
		}
	}

	db, err := sql.Open("sqlite", caissqlite.DSN(dsn))
	if err != nil {
		return nil, fmt.Errorf("open db: %w", err)
	}
	if err := caissqlite.Configure(db); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("configure sqlite: %w", err)
	}

	if err := applyMigrations(db); err != nil {
		_ = db.Close()
		return nil, err
	}

	cfg := sqllog.ConfigForEnv(env)
	if cfg.Enabled {
		cfg.Writer = devlog.MirrorDefault(os.Stdout)
	}
	wrapped := sqllog.Wrap(db, cfg)
	if err := seedAuthData(wrapped.Raw(), env); err != nil {
		_ = wrapped.Close()
		return nil, err
	}
	return &SQLiteStore{db: wrapped}, nil
}

func seedAuthData(db *sql.DB, env string) error {
	if env != "development" {
		return nil
	}
	if err := session.EnsureSQLiteSchema(db); err != nil {
		return err
	}
	hash, err := session.HashPassword("password")
	if err != nil {
		return err
	}
	_, err = db.Exec(
		"INSERT OR IGNORE INTO users (email, name, intention, password_hash) VALUES (?, ?, ?, ?)",
		"demo@example.com", "Sofia", string(models.IntentionBuildHabits), hash,
	)
	return err
}

func (s *SQLiteStore) FindUserByEmail(email string) (models.User, error) {
	return scanUser(s.db.QueryRow(
		`SELECT id, COALESCE(name, ''), email, COALESCE(intention, ''), password_hash, created_at
		   FROM users WHERE email = ?`, email,
	))
}

func (s *SQLiteStore) Sessions() session.Store {
	return session.NewSQLiteStore(s.db.Raw())
}

func (s *SQLiteStore) Ping() error {
	return s.db.Raw().Ping()
}

func (s *SQLiteStore) DB() *sql.DB {
	return s.db.Raw()
}

func (s *SQLiteStore) Close() error {
	return s.db.Close()
}
