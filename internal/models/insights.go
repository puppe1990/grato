package models

import "time"

// TagStat is how often a resonance tag appears in a user's moments.
type TagStat struct {
	Tag   Tag
	Count int
}

// SerenityPoint is one day of the "Curva de Serenidade" chart.
type SerenityPoint struct {
	Day      time.Time
	Serenity int
	Moments  int
}

// Milestone is an achievement card in "Marcos e Conquistas".
type Milestone struct {
	Key     string
	Kind    string
	Value   int
	Goal    int
	Reached bool
}

// Stats are the headline numbers on the Insights screen.
type Stats struct {
	TotalMoments   int
	CurrentStreak  int
	LongestStreak  int
	DaysThisYear   int
	EveningMoments int
	Year           int
	FirstMomentDay time.Time
	LastMomentDay  time.Time
}
