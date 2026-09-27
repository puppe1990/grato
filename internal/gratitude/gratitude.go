// Package gratitude derives the reflection metrics shown across the diary:
// streaks, the serenity curve, heart themes, milestones and the daily quote.
package gratitude

import (
	"math"
	"sort"
	"time"

	"github.com/puppe1990/grato/internal/models"
)

const dayLayout = "2006-01-02"

// Milestone keys and the goals that unlock them.
const (
	MilestoneWeekStreak = "semana_completa"
	MilestoneNightGaze  = "olhar_noturno"
	MilestoneHundred    = "cem_momentos"

	MilestoneKindStreak  = "streak"
	MilestoneKindEvening = "evening"
	MilestoneKindTotal   = "total"

	weekStreakGoal = 7
	nightGazeGoal  = 10
	hundredGoal    = 100
)

// Streak counts consecutive days with at least one moment, ending today.
// An empty today does not break the run — the day is still open — so the
// count falls back to the run ending yesterday.
func Streak(days []time.Time, today time.Time) int {
	loc := today.Location()
	seen := daySet(days, loc)

	day := truncateDay(today)
	if !seen[day.Format(dayLayout)] {
		day = day.AddDate(0, 0, -1)
	}

	streak := 0
	for seen[day.Format(dayLayout)] {
		streak++
		day = day.AddDate(0, 0, -1)
	}
	return streak
}

// LongestStreak is the longest run of consecutive days on record.
func LongestStreak(days []time.Time) int {
	ordered := sortedDays(days)
	longest, run := 0, 0
	for i, day := range ordered {
		if i > 0 && day.Equal(ordered[i-1].AddDate(0, 0, 1)) {
			run++
		} else {
			run = 1
		}
		if run > longest {
			longest = run
		}
	}
	return longest
}

// SerenitySeries averages daily serenity over the trailing window, oldest
// first. Days without moments are skipped so the curve only plots real days.
func SerenitySeries(moments []models.Moment, days int, today time.Time) []models.SerenityPoint {
	if days <= 0 {
		return nil
	}
	loc := today.Location()
	end := truncateDay(today)
	start := end.AddDate(0, 0, -(days - 1))

	byDay := map[string]*models.SerenityPoint{}
	for _, m := range moments {
		day := truncateDay(m.HappenedAt.In(loc))
		if day.Before(start) || day.After(end) {
			continue
		}
		point, ok := byDay[day.Format(dayLayout)]
		if !ok {
			point = &models.SerenityPoint{Day: day}
			byDay[day.Format(dayLayout)] = point
		}
		point.Serenity += m.Serenity
		point.Moments++
	}

	keys := make([]string, 0, len(byDay))
	for key := range byDay {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	series := make([]models.SerenityPoint, 0, len(keys))
	for _, key := range keys {
		point := byDay[key]
		point.Serenity = int(math.Round(float64(point.Serenity) / float64(point.Moments)))
		series = append(series, *point)
	}
	return series
}

// Themes ranks heart themes by usage, keeping at most limit of them.
func Themes(stats []models.TagStat, limit int) []models.TagStat {
	ranked := make([]models.TagStat, len(stats))
	copy(ranked, stats)
	sort.SliceStable(ranked, func(i, j int) bool { return ranked[i].Count > ranked[j].Count })
	if limit > 0 && len(ranked) > limit {
		ranked = ranked[:limit]
	}
	return ranked
}

// Milestones derives the achievement cards from the headline stats.
func Milestones(stats models.Stats) []models.Milestone {
	return []models.Milestone{
		{Key: MilestoneWeekStreak, Kind: MilestoneKindStreak, Value: stats.CurrentStreak, Goal: weekStreakGoal, Reached: stats.CurrentStreak >= weekStreakGoal},
		{Key: MilestoneNightGaze, Kind: MilestoneKindEvening, Value: stats.EveningMoments, Goal: nightGazeGoal, Reached: stats.EveningMoments >= nightGazeGoal},
		{Key: MilestoneHundred, Kind: MilestoneKindTotal, Value: stats.TotalMoments, Goal: hundredGoal, Reached: stats.TotalMoments >= hundredGoal},
	}
}

// ReachedCount counts how many achievements are unlocked.
func ReachedCount(milestones []models.Milestone) int {
	count := 0
	for _, m := range milestones {
		if m.Reached {
			count++
		}
	}
	return count
}

// ReflectionOfDay rotates the daily quote deterministically through the year.
func ReflectionOfDay(day time.Time, count int) int {
	if count <= 0 {
		return 0
	}
	return (day.YearDay() - 1) % count
}

func daySet(days []time.Time, loc *time.Location) map[string]bool {
	seen := make(map[string]bool, len(days))
	for _, day := range days {
		seen[day.In(loc).Format(dayLayout)] = true
	}
	return seen
}

func sortedDays(days []time.Time) []time.Time {
	seen := make(map[string]time.Time, len(days))
	for _, day := range days {
		key := day.Format(dayLayout)
		seen[key] = truncateDay(day)
	}
	ordered := make([]time.Time, 0, len(seen))
	for _, day := range seen {
		ordered = append(ordered, day)
	}
	sort.Slice(ordered, func(i, j int) bool { return ordered[i].Before(ordered[j]) })
	return ordered
}

func truncateDay(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())
}
