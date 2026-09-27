// Package testdata builds fake domain values for tests and dev seeds.
//
// Every value derives from one seeded gofakeit Faker, so a failing test
// reproduces with identical data on every run.
package testdata

import (
	"sort"
	"strings"
	"time"

	"github.com/brianvoe/gofakeit/v7"

	"github.com/puppe1990/grato/internal/models"
)

// Seed keeps generated fixtures identical across runs.
const Seed = uint64(20260927)

// New returns a deterministic faker.
func New() *gofakeit.Faker {
	return gofakeit.New(Seed)
}

// User builds a fake account with a display name and intention.
func User(f *gofakeit.Faker) models.User {
	name := f.FirstName()
	return models.User{
		ID:        int64(f.IntRange(1, 10_000)),
		Name:      name,
		Email:     strings.ToLower(name) + "@example.com",
		Intention: models.Intention(f.RandomString(intentionStrings())),
		CreatedAt: f.Date(),
	}
}

// Moment builds a fake moment for a user at an arbitrary time.
func Moment(f *gofakeit.Faker, userID int64) models.Moment {
	return MomentAt(f, userID, f.Date())
}

// MomentAt builds a fake moment pinned to a day (mid-morning local time).
func MomentAt(f *gofakeit.Faker, userID int64, day time.Time) models.Moment {
	ofDay := models.MomentOfDay(f.RandomString(momentOfDayStrings()))
	return models.Moment{
		ID:          int64(f.IntRange(1, 1_000_000)),
		UserID:      userID,
		MomentOfDay: ofDay,
		Mood:        models.Mood(f.RandomString(moodStrings())),
		Title:       f.Sentence(4),
		Body:        f.Sentence(12),
		Serenity:    f.IntRange(models.SerenityMin, models.SerenityMax),
		HappenedAt:  dayAt(day, ofDay),
		CreatedAt:   day,
	}
}

// MomentsOn builds one moment per supplied day, oldest first.
func MomentsOn(f *gofakeit.Faker, userID int64, days []time.Time) []models.Moment {
	out := make([]models.Moment, 0, len(days))
	for _, day := range days {
		out = append(out, MomentAt(f, userID, day))
	}
	return out
}

// MomentOf builds a fake moment pinned to a specific checkpoint of the day.
func MomentOf(f *gofakeit.Faker, userID int64, day time.Time, ofDay models.MomentOfDay) models.Moment {
	moment := MomentAt(f, userID, day)
	moment.MomentOfDay = ofDay
	moment.HappenedAt = dayAt(day, ofDay)
	return moment
}

// ConsecutiveDays returns n consecutive calendar days ending at `end`.
func ConsecutiveDays(end time.Time, n int) []time.Time {
	days := make([]time.Time, 0, n)
	start := truncateDay(end)
	for i := n - 1; i >= 0; i-- {
		days = append(days, start.AddDate(0, 0, -i))
	}
	return days
}

// ScatteredDays returns n days spread over the past `span` days, without
// repeating a calendar day, sorted oldest first.
func ScatteredDays(f *gofakeit.Faker, span, n int, end time.Time) []time.Time {
	seen := map[string]bool{}
	days := make([]time.Time, 0, n)
	base := truncateDay(end)
	for len(days) < n {
		day := base.AddDate(0, 0, -f.IntRange(0, span-1))
		key := day.Format("2006-01-02")
		if seen[key] {
			continue
		}
		seen[key] = true
		days = append(days, day)
	}
	sort.Slice(days, func(i, j int) bool { return days[i].Before(days[j]) })
	return days
}

// Tags builds fake resonance tags, optionally scoped to a user.
func Tags(f *gofakeit.Faker, userID int64, n int) []models.Tag {
	names := make([]string, 0, n)
	for i := 0; i < n; i++ {
		names = append(names, f.Word())
	}
	out := make([]models.Tag, 0, n)
	for i, name := range names {
		out = append(out, models.Tag{
			ID:     int64(i + 1),
			UserID: userID,
			Name:   name,
			Slug:   Slugify(name),
		})
	}
	return out
}

// TagStats builds fake tag usage counts with strictly decreasing counts.
func TagStats(f *gofakeit.Faker, userID int64, n int) []models.TagStat {
	tags := Tags(f, userID, n)
	out := make([]models.TagStat, 0, n)
	count := f.IntRange(n*3, n*8)
	for _, tag := range tags {
		out = append(out, models.TagStat{Tag: tag, Count: count})
		count -= f.IntRange(1, 3)
	}
	return out
}

// Slugify lowercases a label and collapses spaces into dashes.
func Slugify(name string) string {
	return models.SlugifyTag(name)
}

// TruncateDay drops the time-of-day component.
func TruncateDay(t time.Time) time.Time { return truncateDay(t) }

func truncateDay(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())
}

// DayAt builds a timestamp on `day` at the hour matching the checkpoint.
func DayAt(day time.Time, ofDay models.MomentOfDay) time.Time { return dayAt(day, ofDay) }

func dayAt(day time.Time, ofDay models.MomentOfDay) time.Time {
	hour := map[models.MomentOfDay]int{
		models.MomentMorning:   8,
		models.MomentAfternoon: 15,
		models.MomentEvening:   21,
	}[ofDay]
	return time.Date(day.Year(), day.Month(), day.Day(), hour, 15, 0, 0, day.Location())
}

func moodStrings() []string {
	out := make([]string, 0, len(models.MoodValues))
	for _, v := range models.MoodValues {
		out = append(out, string(v))
	}
	return out
}

func momentOfDayStrings() []string {
	out := make([]string, 0, len(models.MomentOfDayValues))
	for _, v := range models.MomentOfDayValues {
		out = append(out, string(v))
	}
	return out
}

func intentionStrings() []string {
	out := make([]string, 0, len(models.IntentionValues))
	for _, v := range models.IntentionValues {
		out = append(out, string(v))
	}
	return out
}
