package gratitude

import (
	"testing"
	"time"

	"github.com/puppe1990/grato/internal/models"
	"github.com/puppe1990/grato/internal/testdata"
)

func daysOf(moments []models.Moment) []time.Time {
	out := make([]time.Time, 0, len(moments))
	for _, m := range moments {
		out = append(out, m.HappenedAt)
	}
	return out
}

func TestStreak_countsUpToToday(t *testing.T) {
	f := testdata.New()
	today := time.Date(2026, time.October, 24, 18, 0, 0, 0, time.UTC)
	moments := testdata.MomentsOn(f, 1, testdata.ConsecutiveDays(today, 7))

	if got := Streak(daysOf(moments), today); got != 7 {
		t.Errorf("Streak() = %d, want 7", got)
	}
}

func TestStreak_keepsYesterdaysRunWhenTodayIsEmpty(t *testing.T) {
	f := testdata.New()
	today := time.Date(2026, time.October, 24, 18, 0, 0, 0, time.UTC)
	// Four consecutive days ending yesterday, nothing written yet today.
	moments := testdata.MomentsOn(f, 1, testdata.ConsecutiveDays(today.AddDate(0, 0, -1), 4))

	if got := Streak(daysOf(moments), today); got != 4 {
		t.Errorf("Streak() = %d, want 4 (yesterday still counts)", got)
	}
}

func TestStreak_isZeroWhenTheRunIsStale(t *testing.T) {
	f := testdata.New()
	today := time.Date(2026, time.October, 24, 18, 0, 0, 0, time.UTC)
	// Last entry three days ago: the streak is broken.
	moments := testdata.MomentsOn(f, 1, testdata.ConsecutiveDays(today.AddDate(0, 0, -3), 3))

	if got := Streak(daysOf(moments), today); got != 0 {
		t.Errorf("Streak() = %d, want 0", got)
	}
}

func TestStreak_ignoresTimeOfDayAndDuplicateDays(t *testing.T) {
	f := testdata.New()
	today := time.Date(2026, time.October, 24, 23, 30, 0, 0, time.UTC)
	days := testdata.ConsecutiveDays(today, 3)
	moments := testdata.MomentsOn(f, 1, days)
	// Three more entries on the very same days must not inflate the run.
	for i := 0; i < 3; i++ {
		moments = append(moments, testdata.MomentAt(f, 1, days[i]))
	}

	if got := Streak(daysOf(moments), today); got != 3 {
		t.Errorf("Streak() = %d, want 3", got)
	}
}

func TestStreak_ignoresScatteredDaysBeforeTheRun(t *testing.T) {
	f := testdata.New()
	today := time.Date(2026, time.October, 24, 18, 0, 0, 0, time.UTC)
	scattered := testdata.ScatteredDays(f, 120, 20, today.AddDate(0, 0, -10))
	run := testdata.ConsecutiveDays(today, 5)
	moments := testdata.MomentsOn(f, 1, append(scattered, run...))

	if got := Streak(daysOf(moments), today); got != 5 {
		t.Errorf("Streak() = %d, want 5", got)
	}
}

func TestStreak_isZeroWithoutMoments(t *testing.T) {
	today := time.Date(2026, time.October, 24, 18, 0, 0, 0, time.UTC)
	if got := Streak(nil, today); got != 0 {
		t.Errorf("Streak() = %d, want 0", got)
	}
}

func TestLongestStreak_picksTheLongestRun(t *testing.T) {
	f := testdata.New()
	today := time.Date(2026, time.October, 24, 18, 0, 0, 0, time.UTC)
	short := testdata.ConsecutiveDays(today.AddDate(0, 0, -40), 2)
	long := testdata.ConsecutiveDays(today, 9)
	moments := testdata.MomentsOn(f, 1, append(short, long...))

	if got := LongestStreak(daysOf(moments)); got != 9 {
		t.Errorf("LongestStreak() = %d, want 9", got)
	}
}

func TestLongestStreak_isZeroWithoutMoments(t *testing.T) {
	if got := LongestStreak(nil); got != 0 {
		t.Errorf("LongestStreak() = %d, want 0", got)
	}
}

func TestSerenitySeries_averagesMomentsPerDay(t *testing.T) {
	f := testdata.New()
	today := time.Date(2026, time.October, 24, 12, 0, 0, 0, time.UTC)
	day := today.AddDate(0, 0, -1)

	calm := testdata.MomentAt(f, 1, day)
	calm.Serenity = models.SerenityCalm
	radiant := testdata.MomentAt(f, 1, day)
	radiant.Serenity = models.SerenityRadiant

	points := SerenitySeries([]models.Moment{calm, radiant}, 30, today)
	if len(points) != 1 {
		t.Fatalf("len(points) = %d, want 1", len(points))
	}
	if points[0].Serenity != 3 {
		t.Errorf("Serenity = %d, want 3 (average of 1 and 5)", points[0].Serenity)
	}
	if points[0].Moments != 2 {
		t.Errorf("Moments = %d, want 2", points[0].Moments)
	}
}

func TestSerenitySeries_dropsDaysOutsideTheWindowAndOrdersOldestFirst(t *testing.T) {
	f := testdata.New()
	today := time.Date(2026, time.October, 24, 12, 0, 0, 0, time.UTC)
	recent := testdata.ConsecutiveDays(today, 3)
	ancient := testdata.MomentsOn(f, 1, testdata.ConsecutiveDays(today.AddDate(0, 0, -90), 2))
	moments := append(testdata.MomentsOn(f, 1, recent), ancient...)

	points := SerenitySeries(moments, 7, today)
	if len(points) != 3 {
		t.Fatalf("len(points) = %d, want 3 (window is 7 days)", len(points))
	}
	for i := 1; i < len(points); i++ {
		if !points[i].Day.After(points[i-1].Day) {
			t.Fatalf("points not ordered oldest first: %v then %v", points[i-1].Day, points[i].Day)
		}
	}
}

func TestThemes_sortsByCountAndLimits(t *testing.T) {
	f := testdata.New()
	stats := testdata.TagStats(f, 1, 8)

	themes := Themes(stats, 4)
	if len(themes) != 4 {
		t.Fatalf("len(themes) = %d, want 4", len(themes))
	}
	for i := 1; i < len(themes); i++ {
		if themes[i].Count > themes[i-1].Count {
			t.Fatalf("themes not sorted by count desc: %d then %d", themes[i-1].Count, themes[i].Count)
		}
	}
}

func TestMilestones_marksReachedGoals(t *testing.T) {
	milestones := Milestones(models.Stats{CurrentStreak: 7, EveningMoments: 4, TotalMoments: 142})
	if len(milestones) != 3 {
		t.Fatalf("len(milestones) = %d, want 3", len(milestones))
	}

	byKey := map[string]models.Milestone{}
	for _, m := range milestones {
		byKey[m.Key] = m
	}

	if m := byKey["semana_completa"]; !m.Reached || m.Goal != 7 || m.Value != 7 {
		t.Errorf("semana_completa = %+v, want reached with goal 7", m)
	}
	if m := byKey["olhar_noturno"]; m.Reached {
		t.Errorf("olhar_noturno = %+v, want not reached (4 of 10)", m)
	}
	if m := byKey["cem_momentos"]; !m.Reached || m.Goal != 100 {
		t.Errorf("cem_momentos = %+v, want reached with goal 100", m)
	}
}

func TestMilestones_countsReached(t *testing.T) {
	milestones := Milestones(models.Stats{CurrentStreak: 7, EveningMoments: 12, TotalMoments: 10})
	if got := ReachedCount(milestones); got != 2 {
		t.Errorf("ReachedCount() = %d, want 2", got)
	}
}

func TestReflectionOfDay_rotatesAndStaysInRange(t *testing.T) {
	f := testdata.New()
	first := ReflectionOfDay(time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC), 4)
	if first != 0 {
		t.Errorf("ReflectionOfDay(new year) = %d, want 0", first)
	}

	for _, day := range testdata.ScatteredDays(f, 400, 40, time.Date(2026, time.December, 31, 0, 0, 0, 0, time.UTC)) {
		got := ReflectionOfDay(day, 4)
		if got < 0 || got >= 4 {
			t.Fatalf("ReflectionOfDay(%s) = %d, out of range", day.Format("2006-01-02"), got)
		}
		// Same calendar day must always pick the same quote.
		if again := ReflectionOfDay(day.Add(3*time.Hour), 4); again != got {
			t.Fatalf("ReflectionOfDay is not stable within the day: %d then %d", got, again)
		}
	}
}

func TestReflectionOfDay_isSafeWithoutQuotes(t *testing.T) {
	if got := ReflectionOfDay(time.Now(), 0); got != 0 {
		t.Errorf("ReflectionOfDay(count 0) = %d, want 0", got)
	}
}
