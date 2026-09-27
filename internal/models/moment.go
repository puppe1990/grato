package models

import (
	"strings"
	"time"
)

// MomentOfDay names the three gratitude checkpoints of a day.
type MomentOfDay string

const (
	MomentMorning   MomentOfDay = "manha"
	MomentAfternoon MomentOfDay = "tarde"
	MomentEvening   MomentOfDay = "noite"
)

// MomentOfDayValues is the ordered set of checkpoints shown in the UI.
var MomentOfDayValues = []MomentOfDay{MomentMorning, MomentAfternoon, MomentEvening}

// Valid reports whether the value is one of the three checkpoints.
func (m MomentOfDay) Valid() bool {
	for _, v := range MomentOfDayValues {
		if m == v {
			return true
		}
	}
	return false
}

// Mood is the feeling chip selected in "Como você está agora?".
type Mood string

const (
	MoodRadiant    Mood = "radiante"
	MoodSerene     Mood = "sereno"
	MoodGrateful   Mood = "grato"
	MoodReflective Mood = "reflexivo"
	MoodTired      Mood = "cansado"
)

// MoodValues is the ordered set of feeling chips.
var MoodValues = []Mood{MoodRadiant, MoodSerene, MoodGrateful, MoodReflective, MoodTired}

// Valid reports whether the value is a known feeling chip.
func (m Mood) Valid() bool {
	for _, v := range MoodValues {
		if m == v {
			return true
		}
	}
	return false
}

// Serenity bounds for the "Como seu coração se sentiu?" slider.
const (
	SerenityCalm      = 1
	SerenityBalanced  = 3
	SerenityRadiant   = 5
	SerenityMin       = SerenityCalm
	SerenityMax       = SerenityRadiant
	SerenityDefault   = SerenityBalanced
	SerenityScaleSpan = SerenityMax - SerenityMin
)

// ClampSerenity keeps a slider value inside the supported scale.
func ClampSerenity(v int) int {
	if v < SerenityMin {
		return SerenityMin
	}
	if v > SerenityMax {
		return SerenityMax
	}
	return v
}

// Moment is one gratitude entry written by a user.
type Moment struct {
	ID          int64
	UserID      int64
	MomentOfDay MomentOfDay
	Mood        Mood
	Title       string
	Body        string
	Serenity    int
	HappenedAt  time.Time
	CreatedAt   time.Time
	Tags        []Tag
}

// Tag is a resonance label such as #paz or #família.
type Tag struct {
	ID     int64
	UserID int64
	Name   string
	Slug   string
}

// SlugifyTag normalizes a label into its storage key: accents are kept,
// a leading '#' and surrounding space are dropped, and runs of whitespace
// collapse into single dashes.
func SlugifyTag(name string) string {
	slug := strings.ToLower(strings.TrimSpace(name))
	slug = strings.TrimPrefix(slug, "#")
	return strings.Join(strings.Fields(slug), "-")
}

// DayKey is the calendar day a moment belongs to, in local time.
func (m Moment) DayKey() string {
	return m.HappenedAt.Local().Format("2006-01-02")
}
