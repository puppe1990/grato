package i18n

import (
	"fmt"
	"strings"
	"time"
)

// Date vocabulary. Only the locales the app ships are listed; anything else
// falls back to English so a missing translation never breaks a page.
var weekdayNames = map[string][]string{
	"pt": {"domingo", "segunda-feira", "terça-feira", "quarta-feira", "quinta-feira", "sexta-feira", "sábado"},
	"en": {"Sunday", "Monday", "Tuesday", "Wednesday", "Thursday", "Friday", "Saturday"},
}

var weekdayShortNames = map[string][]string{
	"pt": {"dom", "seg", "ter", "qua", "qui", "sex", "sáb"},
	"en": {"Sun", "Mon", "Tue", "Wed", "Thu", "Fri", "Sat"},
}

var monthNames = map[string][]string{
	"pt": {"janeiro", "fevereiro", "março", "abril", "maio", "junho", "julho", "agosto", "setembro", "outubro", "novembro", "dezembro"},
	"en": {"January", "February", "March", "April", "May", "June", "July", "August", "September", "October", "November", "December"},
}

var monthTitleNames = map[string][]string{
	"pt": {"Janeiro", "Fevereiro", "Março", "Abril", "Maio", "Junho", "Julho", "Agosto", "Setembro", "Outubro", "Novembro", "Dezembro"},
	"en": {"January", "February", "March", "April", "May", "June", "July", "August", "September", "October", "November", "December"},
}

// FullDate renders "quarta-feira, 24 de outubro" (pt) or "Wednesday, October 24" (en).
func FullDate(locale string, day time.Time) string {
	weekday := weekdayName(locale, day)
	month := monthName(locale, day)
	if isPortuguese(locale) {
		return fmt.Sprintf("%s, %d de %s", weekday, day.Day(), month)
	}
	return fmt.Sprintf("%s, %s %d", weekday, month, day.Day())
}

// MonthYear renders "Outubro 2024", the calendar header on Memórias.
func MonthYear(locale string, day time.Time) string {
	return fmt.Sprintf("%s %d", monthTitle(locale, day), day.Year())
}

// WeekdayShort renders the three-letter weekday used in the week strip.
func WeekdayShort(locale string, day time.Time) string {
	names, ok := weekdayShortNames[Base(locale)]
	if !ok {
		names = weekdayShortNames["en"]
	}
	return names[int(day.Weekday())]
}

// DayMonth renders "22 de outubro" for timeline entry headers.
func DayMonth(locale string, day time.Time) string {
	if isPortuguese(locale) {
		return fmt.Sprintf("%d de %s", day.Day(), monthName(locale, day))
	}
	return fmt.Sprintf("%s %d", monthName(locale, day), day.Day())
}

// ClockTime renders a moment's time as HH:MM.
func ClockTime(at time.Time) string {
	return fmt.Sprintf("%02d:%02d", at.Hour(), at.Minute())
}

// Greeting picks the Portuguese greeting for the hour of day.
func Greeting(locale string, at time.Time) string {
	key := "greeting.morning"
	switch {
	case at.Hour() >= 12 && at.Hour() < 18:
		key = "greeting.afternoon"
	case at.Hour() >= 18 || at.Hour() < 5:
		key = "greeting.evening"
	}
	return NewCatalog(locale).T(key)
}

func weekdayName(locale string, day time.Time) string {
	names, ok := weekdayNames[Base(locale)]
	if !ok {
		names = weekdayNames["en"]
	}
	return names[int(day.Weekday())]
}

func monthName(locale string, day time.Time) string {
	names, ok := monthNames[Base(locale)]
	if !ok {
		names = monthNames["en"]
	}
	return names[int(day.Month())-1]
}

func monthTitle(locale string, day time.Time) string {
	names, ok := monthTitleNames[Base(locale)]
	if !ok {
		names = monthTitleNames["en"]
	}
	return names[int(day.Month())-1]
}

func isPortuguese(locale string) bool {
	return Base(locale) == "pt"
}

// Base reduces "pt-BR" / "pt_BR" to its language subtag ("pt"), which is what
// the language toggle and the date vocabulary compare against.
func Base(locale string) string {
	lowered := strings.ToLower(strings.TrimSpace(locale))
	if i := strings.IndexAny(lowered, "-_"); i > 0 {
		return lowered[:i]
	}
	return lowered
}
