package handlers

import (
	"net/http"
	"time"

	appi18n "github.com/puppe1990/grato/internal/i18n"

	"github.com/puppe1990/amarra-cais/pkg/cais/i18n"

	"github.com/puppe1990/grato/internal/models"
)

// Memories is the timeline screen: month calendar, tag filters, the featured
// story of this month in past years and the reverse-chronological entries.
func (h *DiaryHandler) Memories(w http.ResponseWriter, r *http.Request) {
	user, ok := h.currentUser(w, r)
	if !ok {
		return
	}

	now := h.now()
	total, err := h.store.CountMoments(user.ID)
	if err != nil {
		h.fail(w, err)
		return
	}
	monthMoments, err := h.store.MomentsBetween(user.ID, firstOfMonth(now), lastOfMonth(now))
	if err != nil {
		h.fail(w, err)
		return
	}
	moments, err := h.store.ListMoments(user.ID, 60)
	if err != nil {
		h.fail(w, err)
		return
	}
	tags, err := h.store.TagsForUser(user.ID)
	if err != nil {
		h.fail(w, err)
		return
	}
	archive, err := h.store.MomentsBetween(user.ID, firstOfMonth(now).AddDate(-6, 0, 0), lastOfMonth(now))
	if err != nil {
		h.fail(w, err)
		return
	}

	filter := r.URL.Query().Get("tag")
	entries := filterMoments(moments, filter)
	locale := h.locale(r)

	h.render(w, r, &user, "memories", map[string]any{
		"Title":         h.catalogFor(r).T("nav.memories"),
		"ActiveNav":     "memories",
		"Total":         total,
		"MonthLabel":    appi18n.MonthYear(locale, now),
		"Calendar":      buildCalendar(now, monthMoments),
		"Weekdays":      calendarWeekdays(locale),
		"MarkedCount":   markedDays(monthMoments),
		"Filters":       tagFilters(h.catalogFor(r).T("memories.filter_all"), tags, filter),
		"ActiveFilter":  filter,
		"Entries":       h.entryViews(h.catalogFor(r), entries, locale),
		"HasEntries":    len(entries) > 0,
		"Featured":      featuredMoment(archive, now),
		"FeaturedLabel": featuredLabel(h.catalog, archive, now),
	}, 0)
}

// entryView is one card in the timeline.
type entryView struct {
	Moment      models.Moment
	DayLabel    string
	TimeLabel   string
	MoodLabel   string
	ReadingTime int
	Tags        []models.Tag
}

func (h *DiaryHandler) entryViews(catalog *i18n.Catalog, moments []models.Moment, locale string) []entryView {
	views := make([]entryView, 0, len(moments))
	for _, moment := range moments {
		views = append(views, entryView{
			Moment:      moment,
			DayLabel:    appi18n.DayMonth(locale, moment.HappenedAt),
			TimeLabel:   appi18n.ClockTime(moment.HappenedAt),
			MoodLabel:   catalog.T("mood." + string(moment.Mood)),
			ReadingTime: readingMinutes(moment.Body),
			Tags:        moment.Tags,
		})
	}
	return views
}

// calendarDay is one cell of the month grid.
type calendarDay struct {
	Number  int
	Key     string
	Marked  bool
	IsToday bool
	Other   bool
}

func buildCalendar(month time.Time, moments []models.Moment) []calendarDay {
	recorded := map[string]bool{}
	for _, moment := range moments {
		recorded[moment.DayKey()] = true
	}

	first := firstOfMonth(month)
	leading := int(first.Weekday())
	days := make([]calendarDay, 0, leading+daysInMonth(month)+7)
	for i := 0; i < leading; i++ {
		days = append(days, calendarDay{Other: true})
	}
	for day := 1; day <= daysInMonth(month); day++ {
		at := time.Date(month.Year(), month.Month(), day, 0, 0, 0, 0, month.Location())
		days = append(days, calendarDay{
			Number:  day,
			Key:     at.Format("2006-01-02"),
			Marked:  recorded[at.Format("2006-01-02")],
			IsToday: sameDay(at, month),
		})
	}
	return days
}

func calendarWeekdays(locale string) []string {
	// Week starts on Sunday, matching Go's time.Weekday.
	sunday := time.Date(2026, time.March, 1, 0, 0, 0, 0, time.UTC)
	names := make([]string, 0, 7)
	for i := 0; i < 7; i++ {
		names = append(names, appi18n.WeekdayShort(locale, sunday.AddDate(0, 0, i)))
	}
	return names
}

func markedDays(moments []models.Moment) int {
	seen := map[string]bool{}
	for _, moment := range moments {
		seen[moment.DayKey()] = true
	}
	return len(seen)
}

// tagFilter is one chip in the resonance filter row.
type tagFilter struct {
	Name   string
	Value  string
	Active bool
}

func tagFilters(allLabel string, tags []models.Tag, active string) []tagFilter {
	filters := []tagFilter{{Name: allLabel, Value: "", Active: active == ""}}
	for _, tag := range tags {
		filters = append(filters, tagFilter{Name: tag.Name, Value: tag.Slug, Active: tag.Slug == active})
	}
	return filters
}

func filterMoments(moments []models.Moment, slug string) []models.Moment {
	if slug == "" {
		return moments
	}
	filtered := make([]models.Moment, 0, len(moments))
	for _, moment := range moments {
		for _, tag := range moment.Tags {
			if tag.Slug == slug {
				filtered = append(filtered, moment)
				break
			}
		}
	}
	return filtered
}

// featuredMoment is the oldest entry written in this calendar month of a
// previous year — the "há 1 ano" card on Memórias.
func featuredMoment(archive []models.Moment, now time.Time) *models.Moment {
	for i := range archive {
		moment := archive[i]
		if moment.HappenedAt.Month() != now.Month() || moment.HappenedAt.Year() >= now.Year() {
			continue
		}
		return &archive[i]
	}
	return nil
}

func featuredLabel(catalog interface{ T(string, ...any) string }, archive []models.Moment, now time.Time) string {
	moment := featuredMoment(archive, now)
	if moment == nil {
		return ""
	}
	years := now.Year() - moment.HappenedAt.Year()
	if years <= 1 {
		return catalog.T("memories.time_ago_year")
	}
	return catalog.T("memories.time_ago_years", years)
}

// readingMinutes estimates how long a reflection takes to read.
func readingMinutes(body string) int {
	runes := len([]rune(body))
	minutes := (runes + 899) / 900
	if minutes < 1 {
		return 1
	}
	return minutes
}

func firstOfMonth(at time.Time) time.Time {
	return time.Date(at.Year(), at.Month(), 1, 0, 0, 0, 0, at.Location())
}

func lastOfMonth(at time.Time) time.Time {
	return firstOfMonth(at).AddDate(0, 1, -1)
}

func daysInMonth(at time.Time) int {
	return lastOfMonth(at).Day()
}

func sameDay(a, b time.Time) bool {
	return a.Year() == b.Year() && a.YearDay() == b.YearDay()
}
