package store

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/puppe1990/amarra-cais/pkg/cais/sqllog"

	"github.com/puppe1990/grato/internal/models"
)

// stampLayout matches SQLite's own DATETIME text form so string ordering and
// ORDER BY stay chronological.
const stampLayout = "2006-01-02 15:04:05"

const dayLayout = "2006-01-02"

const momentColumns = `SELECT id, user_id, moment_of_day, mood, title, body, serenity, happened_at, created_at FROM moments`

func (s *SQLiteStore) CreateUser(user models.User) (int64, error) {
	result, err := s.db.Exec(
		"INSERT INTO users (email, name, intention, password_hash) VALUES (?, ?, ?, ?)",
		user.Email, user.Name, string(user.Intention), user.PasswordHash,
	)
	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE") {
			return 0, ErrEmailTaken
		}
		return 0, fmt.Errorf("create user: %w", err)
	}
	return result.LastInsertId()
}

func (s *SQLiteStore) FindUserByID(id int64) (models.User, error) {
	return scanUser(s.db.QueryRow(
		`SELECT id, COALESCE(name, ''), email, COALESCE(intention, ''), password_hash, created_at
		   FROM users WHERE id = ?`, id,
	))
}

func (s *SQLiteStore) UpdateUserProfile(id int64, name string, intention models.Intention) error {
	_, err := s.db.Exec("UPDATE users SET name = ?, intention = ? WHERE id = ?", name, string(intention), id)
	if err != nil {
		return fmt.Errorf("update profile: %w", err)
	}
	return nil
}

func (s *SQLiteStore) InsertMoment(m models.Moment) (int64, error) {
	result, err := s.db.Exec(
		`INSERT INTO moments (user_id, moment_of_day, mood, title, body, serenity, day, happened_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		m.UserID, string(m.MomentOfDay), string(m.Mood), m.Title, m.Body,
		models.ClampSerenity(m.Serenity), m.DayKey(), m.HappenedAt.Format(stampLayout),
	)
	if err != nil {
		return 0, fmt.Errorf("insert moment: %w", err)
	}
	return result.LastInsertId()
}

func (s *SQLiteStore) FindMoment(id int64) (models.Moment, error) {
	return s.findMoment(momentColumns+" WHERE id = ?", id)
}

// FindMomentForUser scopes a lookup to its owner so one account can never
// read another's entry.
func (s *SQLiteStore) FindMomentForUser(id, userID int64) (models.Moment, error) {
	return s.findMoment(momentColumns+" WHERE id = ? AND user_id = ?", id, userID)
}

func (s *SQLiteStore) findMoment(query string, args ...any) (models.Moment, error) {
	moment, err := scanMoment(s.db.QueryRow(query, args...))
	if err != nil {
		return models.Moment{}, err
	}
	loaded, err := s.withTags([]models.Moment{moment})
	if err != nil {
		return models.Moment{}, err
	}
	return loaded[0], nil
}

func (s *SQLiteStore) ListMoments(userID int64, limit int) ([]models.Moment, error) {
	if limit <= 0 {
		limit = 50
	}
	rows, err := s.db.Query(
		momentColumns+" WHERE user_id = ? ORDER BY happened_at DESC, id DESC LIMIT ?",
		userID, limit,
	)
	if err != nil {
		return nil, fmt.Errorf("list moments: %w", err)
	}
	defer func() { _ = rows.Close() }()

	moments, err := scanMoments(rows)
	if err != nil {
		return nil, err
	}
	return s.withTags(moments)
}

func (s *SQLiteStore) MomentsOnDay(userID int64, day time.Time) ([]models.Moment, error) {
	return s.momentsBetween(userID, day.Format(dayLayout), day.Format(dayLayout))
}

func (s *SQLiteStore) MomentsBetween(userID int64, from, to time.Time) ([]models.Moment, error) {
	return s.momentsBetween(userID, from.Format(dayLayout), to.Format(dayLayout))
}

func (s *SQLiteStore) momentsBetween(userID int64, from, to string) ([]models.Moment, error) {
	rows, err := s.db.Query(
		momentColumns+" WHERE user_id = ? AND day BETWEEN ? AND ? ORDER BY happened_at ASC, id ASC",
		userID, from, to,
	)
	if err != nil {
		return nil, fmt.Errorf("moments between: %w", err)
	}
	defer func() { _ = rows.Close() }()

	moments, err := scanMoments(rows)
	if err != nil {
		return nil, err
	}
	return s.withTags(moments)
}

func (s *SQLiteStore) MomentDays(userID int64) ([]time.Time, error) {
	rows, err := s.db.Query("SELECT DISTINCT day FROM moments WHERE user_id = ? ORDER BY day ASC", userID)
	if err != nil {
		return nil, fmt.Errorf("moment days: %w", err)
	}
	defer func() { _ = rows.Close() }()

	days := []time.Time{}
	for rows.Next() {
		var day string
		if err := rows.Scan(&day); err != nil {
			return nil, fmt.Errorf("scan day: %w", err)
		}
		parsed, err := time.ParseInLocation(dayLayout, day, time.Local)
		if err != nil {
			return nil, fmt.Errorf("parse day %q: %w", day, err)
		}
		days = append(days, parsed)
	}
	return days, rows.Err()
}

func (s *SQLiteStore) CountMoments(userID int64) (int64, error) {
	return s.countQuery("SELECT COUNT(*) FROM moments WHERE user_id = ?", userID)
}

func (s *SQLiteStore) CountMomentsAt(userID int64, ofDay models.MomentOfDay) (int64, error) {
	return s.countQuery(
		"SELECT COUNT(*) FROM moments WHERE user_id = ? AND moment_of_day = ?",
		userID, string(ofDay),
	)
}

func (s *SQLiteStore) CountMomentDaysInYear(userID int64, year int) (int64, error) {
	return s.countQuery(
		"SELECT COUNT(DISTINCT day) FROM moments WHERE user_id = ? AND day LIKE ?",
		userID, fmt.Sprintf("%04d-%%", year),
	)
}

func (s *SQLiteStore) TagsForUser(userID int64) ([]models.Tag, error) {
	rows, err := s.db.Query(
		"SELECT id, user_id, name, slug FROM tags WHERE user_id = ? ORDER BY name ASC", userID,
	)
	if err != nil {
		return nil, fmt.Errorf("tags for user: %w", err)
	}
	defer func() { _ = rows.Close() }()

	tags := []models.Tag{}
	for rows.Next() {
		var tag models.Tag
		if err := rows.Scan(&tag.ID, &tag.UserID, &tag.Name, &tag.Slug); err != nil {
			return nil, fmt.Errorf("scan tag: %w", err)
		}
		tags = append(tags, tag)
	}
	return tags, rows.Err()
}

// AttachTags records the resonance labels of a moment, reusing tags the user
// already created so the same label never forks into two rows.
func (s *SQLiteStore) AttachTags(momentID, userID int64, names []string) error {
	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("begin tags: %w", err)
	}
	for _, name := range names {
		slug := models.SlugifyTag(name)
		if slug == "" {
			continue
		}
		if err := attachTag(tx, momentID, userID, strings.TrimSpace(name), slug); err != nil {
			_ = tx.Rollback()
			return err
		}
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit tags: %w", err)
	}
	return nil
}

func attachTag(tx *sqllog.Tx, momentID, userID int64, name, slug string) error {
	if _, err := tx.Exec(
		"INSERT OR IGNORE INTO tags (user_id, name, slug) VALUES (?, ?, ?)", userID, name, slug,
	); err != nil {
		return fmt.Errorf("insert tag: %w", err)
	}
	var tagID int64
	if err := tx.QueryRow("SELECT id FROM tags WHERE user_id = ? AND slug = ?", userID, slug).Scan(&tagID); err != nil {
		return fmt.Errorf("find tag: %w", err)
	}
	if _, err := tx.Exec(
		"INSERT OR IGNORE INTO moment_tags (moment_id, tag_id) VALUES (?, ?)", momentID, tagID,
	); err != nil {
		return fmt.Errorf("link tag: %w", err)
	}
	return nil
}

func (s *SQLiteStore) TagStats(userID int64, limit int) ([]models.TagStat, error) {
	if limit <= 0 {
		limit = 12
	}
	rows, err := s.db.Query(
		`SELECT t.id, t.user_id, t.name, t.slug, COUNT(mt.moment_id) AS uses
		   FROM tags t
		   JOIN moment_tags mt ON mt.tag_id = t.id
		  WHERE t.user_id = ?
		  GROUP BY t.id
		  ORDER BY uses DESC, t.name ASC
		  LIMIT ?`,
		userID, limit,
	)
	if err != nil {
		return nil, fmt.Errorf("tag stats: %w", err)
	}
	defer func() { _ = rows.Close() }()

	stats := []models.TagStat{}
	for rows.Next() {
		var stat models.TagStat
		if err := rows.Scan(&stat.Tag.ID, &stat.Tag.UserID, &stat.Tag.Name, &stat.Tag.Slug, &stat.Count); err != nil {
			return nil, fmt.Errorf("scan tag stat: %w", err)
		}
		stats = append(stats, stat)
	}
	return stats, rows.Err()
}

func (s *SQLiteStore) FindRitual(userID int64, kind string) (models.Ritual, error) {
	ritual := models.Ritual{UserID: userID, Kind: kind, RemindAt: models.DefaultRemindAt}
	err := s.db.QueryRow(
		"SELECT id, enabled, remind_at FROM rituals WHERE user_id = ? AND kind = ?", userID, kind,
	).Scan(&ritual.ID, &ritual.Enabled, &ritual.RemindAt)
	if err == sql.ErrNoRows {
		return ritual, nil
	}
	if err != nil {
		return models.Ritual{}, fmt.Errorf("find ritual: %w", err)
	}
	return ritual, nil
}

func (s *SQLiteStore) SaveRitual(r models.Ritual) error {
	_, err := s.db.Exec(
		`INSERT INTO rituals (user_id, kind, enabled, remind_at) VALUES (?, ?, ?, ?)
		 ON CONFLICT (user_id, kind) DO UPDATE SET enabled = excluded.enabled, remind_at = excluded.remind_at`,
		r.UserID, r.Kind, r.Enabled, r.RemindAt,
	)
	if err != nil {
		return fmt.Errorf("save ritual: %w", err)
	}
	return nil
}

func (s *SQLiteStore) countQuery(query string, args ...any) (int64, error) {
	var count int64
	if err := s.db.QueryRow(query, args...).Scan(&count); err != nil {
		return 0, fmt.Errorf("count: %w", err)
	}
	return count, nil
}

// withTags fills every moment's tags using one query for the whole page.
func (s *SQLiteStore) withTags(moments []models.Moment) ([]models.Moment, error) {
	if len(moments) == 0 {
		return moments, nil
	}
	ids := make([]int64, 0, len(moments))
	for _, m := range moments {
		ids = append(ids, m.ID)
	}
	byMoment, err := s.tagsForMoments(ids)
	if err != nil {
		return nil, err
	}
	for i := range moments {
		moments[i].Tags = byMoment[moments[i].ID]
	}
	return moments, nil
}

func (s *SQLiteStore) tagsForMoments(ids []int64) (map[int64][]models.Tag, error) {
	placeholders := strings.TrimSuffix(strings.Repeat("?,", len(ids)), ",")
	args := make([]any, 0, len(ids))
	for _, id := range ids {
		args = append(args, id)
	}

	rows, err := s.db.Query(
		`SELECT mt.moment_id, t.id, t.user_id, t.name, t.slug
		   FROM moment_tags mt
		   JOIN tags t ON t.id = mt.tag_id
		  WHERE mt.moment_id IN (`+placeholders+`)
		  ORDER BY t.name ASC`,
		args...,
	)
	if err != nil {
		return nil, fmt.Errorf("tags for moments: %w", err)
	}
	defer func() { _ = rows.Close() }()

	byMoment := map[int64][]models.Tag{}
	for rows.Next() {
		var momentID int64
		var tag models.Tag
		if err := rows.Scan(&momentID, &tag.ID, &tag.UserID, &tag.Name, &tag.Slug); err != nil {
			return nil, fmt.Errorf("scan moment tag: %w", err)
		}
		byMoment[momentID] = append(byMoment[momentID], tag)
	}
	return byMoment, rows.Err()
}

func scanUser(row *sql.Row) (models.User, error) {
	var u models.User
	var intention string
	if err := row.Scan(&u.ID, &u.Name, &u.Email, &intention, &u.PasswordHash, &u.CreatedAt); err != nil {
		if err == sql.ErrNoRows {
			return models.User{}, ErrNotFound
		}
		return models.User{}, fmt.Errorf("scan user: %w", err)
	}
	u.Intention = models.Intention(intention)
	return u, nil
}

func scanMoment(row *sql.Row) (models.Moment, error) {
	var m models.Moment
	var ofDay string
	if err := row.Scan(&m.ID, &m.UserID, &ofDay, &m.Mood, &m.Title, &m.Body, &m.Serenity, &m.HappenedAt, &m.CreatedAt); err != nil {
		if err == sql.ErrNoRows {
			return models.Moment{}, ErrNotFound
		}
		return models.Moment{}, fmt.Errorf("scan moment: %w", err)
	}
	m.MomentOfDay = models.MomentOfDay(ofDay)
	return m, nil
}

func scanMoments(rows *sql.Rows) ([]models.Moment, error) {
	moments := []models.Moment{}
	for rows.Next() {
		var m models.Moment
		var ofDay string
		if err := rows.Scan(&m.ID, &m.UserID, &ofDay, &m.Mood, &m.Title, &m.Body, &m.Serenity, &m.HappenedAt, &m.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan moment: %w", err)
		}
		m.MomentOfDay = models.MomentOfDay(ofDay)
		moments = append(moments, m)
	}
	return moments, rows.Err()
}
