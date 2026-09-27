package i18n

import (
	"testing"
	"time"
)

func TestFullDate_portuguese(t *testing.T) {
	// 24 October 2026 is a Saturday.
	day := time.Date(2026, time.October, 24, 9, 0, 0, 0, time.UTC)
	if got := FullDate("pt", day); got != "sábado, 24 de outubro" {
		t.Errorf("FullDate(pt) = %q", got)
	}
}

func TestFullDate_english(t *testing.T) {
	day := time.Date(2026, time.October, 24, 9, 0, 0, 0, time.UTC)
	if got := FullDate("en", day); got != "Saturday, October 24" {
		t.Errorf("FullDate(en) = %q", got)
	}
}

func TestMonthYear_portuguese(t *testing.T) {
	day := time.Date(2024, time.October, 3, 0, 0, 0, 0, time.UTC)
	if got := MonthYear("pt", day); got != "Outubro 2024" {
		t.Errorf("MonthYear(pt) = %q", got)
	}
}

func TestWeekdayShort_portuguese(t *testing.T) {
	cases := map[int]string{
		20: "dom", // 20 Sep 2026 is a Sunday
		21: "seg",
		22: "ter",
		23: "qua",
		24: "qui",
		25: "sex",
		26: "sáb",
	}
	for day, want := range cases {
		got := WeekdayShort("pt", time.Date(2026, time.September, day, 12, 0, 0, 0, time.UTC))
		if got != want {
			t.Errorf("WeekdayShort(pt, Sep %d) = %q, want %q", day, got, want)
		}
	}
}

func TestDayMonth_portuguese(t *testing.T) {
	day := time.Date(2026, time.October, 22, 12, 0, 0, 0, time.UTC)
	if got := DayMonth("pt", day); got != "22 de outubro" {
		t.Errorf("DayMonth(pt) = %q", got)
	}
}

func TestClockTime_formatsHourMinute(t *testing.T) {
	at := time.Date(2026, time.October, 22, 8, 15, 0, 0, time.UTC)
	if got := ClockTime(at); got != "08:15" {
		t.Errorf("ClockTime() = %q, want 08:15", got)
	}
}

func TestNames_fallBackToEnglishForUnknownLocale(t *testing.T) {
	day := time.Date(2026, time.October, 24, 9, 0, 0, 0, time.UTC)
	if got := WeekdayShort("de", day); got != "Sat" {
		t.Errorf("WeekdayShort(de) = %q, want Sat", got)
	}
}

func TestGreeting_tracksTheHourOfDay(t *testing.T) {
	day := time.Date(2026, time.October, 24, 9, 0, 0, 0, time.UTC)
	cases := map[int]string{
		6:  "Bom dia",
		12: "Boa tarde",
		17: "Boa tarde",
		18: "Boa noite",
		23: "Boa noite",
		2:  "Boa noite",
	}
	for hour, want := range cases {
		got := Greeting("pt", time.Date(day.Year(), day.Month(), day.Day(), hour, 0, 0, 0, time.UTC))
		if got != want {
			t.Errorf("Greeting(pt, %dh) = %q, want %q", hour, got, want)
		}
	}
}
