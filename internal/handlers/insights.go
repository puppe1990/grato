package handlers

import (
	"fmt"
	"math"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/puppe1990/amarra-cais/pkg/cais/flash"
	"github.com/puppe1990/amarra-cais/pkg/cais/httpx"

	"github.com/puppe1990/amarra-cais/pkg/cais/i18n"

	"github.com/puppe1990/grato/internal/gratitude"

	"github.com/puppe1990/grato/internal/models"
)

// serenityWindow is how many days the "Curva de Serenidade" plots.
const serenityWindow = 30

// Insights summarises the journal: month recap, headline stats, the serenity
// curve, heart themes, milestones and the nightly ritual.
func (h *DiaryHandler) Insights(w http.ResponseWriter, r *http.Request) {
	user, ok := h.currentUser(w, r)
	if !ok {
		return
	}
	now := h.now()

	stats, err := h.statsFor(user.ID, now)
	if err != nil {
		h.fail(w, err)
		return
	}
	moments, err := h.store.MomentsBetween(user.ID, now.AddDate(0, 0, -(serenityWindow-1)), now)
	if err != nil {
		h.fail(w, err)
		return
	}
	tagStats, err := h.store.TagStats(user.ID, 6)
	if err != nil {
		h.fail(w, err)
		return
	}
	monthMoments, err := h.store.MomentsBetween(user.ID, firstOfMonth(now), lastOfMonth(now))
	if err != nil {
		h.fail(w, err)
		return
	}
	ritual, err := h.store.FindRitual(user.ID, models.RitualNightReflection)
	if err != nil {
		h.fail(w, err)
		return
	}

	milestones := gratitude.Milestones(stats)
	series := gratitude.SerenitySeries(moments, serenityWindow, now)

	h.render(w, r, &user, "insights", map[string]any{
		"Title":         h.catalogFor(r).T("nav.insights"),
		"ActiveNav":     "insights",
		"Stats":         stats,
		"Milestones":    h.milestoneViews(h.catalogFor(r), milestones),
		"Unlocked":      gratitude.ReachedCount(milestones),
		"Chart":         buildSerenityChart(series),
		"HasChart":      len(series) > 1,
		"Themes":        themeViews(gratitude.Themes(tagStats, 6)),
		"HasThemes":     len(tagStats) > 0,
		"MonthDays":     markedDays(monthMoments),
		"MonthLabel":    h.catalogFor(r).T("insights.month_card_label"),
		"Predominant":   h.catalogFor(r).T("insights.predominant", h.catalogFor(r).T("mood."+string(predominantMood(monthMoments)))),
		"SerenityDelta": serenityDelta(series),
		"Ritual":        ritual,
		"RitualKinds":   models.RitualNightReflection,
		"HasAnyMoments": stats.TotalMoments > 0,
		"MarkedMemory":  markedMemory(moments),
	}, 0)
}

// RitualPost toggles or reschedules the nightly reflection ritual.
func (h *DiaryHandler) RitualPost(w http.ResponseWriter, r *http.Request) {
	user, ok := h.currentUser(w, r)
	if !ok {
		return
	}
	if err := httpx.ParseFormOrJSON(r); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	remindAt := strings.TrimSpace(r.FormValue("remind_at"))
	if !validClockTime(remindAt) {
		remindAt = models.DefaultRemindAt
	}
	ritual := models.Ritual{
		UserID:   user.ID,
		Kind:     models.RitualNightReflection,
		Enabled:  httpx.FormTruthy(r.FormValue("enabled")),
		RemindAt: remindAt,
	}
	if err := h.store.SaveRitual(ritual); err != nil {
		h.fail(w, err)
		return
	}
	flash.Set(w, "notice", h.catalogFor(r).T("insights.ritual_title"), h.cfg.CookieSecure())
	http.Redirect(w, r, insightsPath, http.StatusSeeOther)
}

func (h *DiaryHandler) statsFor(userID int64, now time.Time) (models.Stats, error) {
	days, err := h.store.MomentDays(userID)
	if err != nil {
		return models.Stats{}, err
	}
	total, err := h.store.CountMoments(userID)
	if err != nil {
		return models.Stats{}, err
	}
	evening, err := h.store.CountMomentsAt(userID, models.MomentEvening)
	if err != nil {
		return models.Stats{}, err
	}
	daysThisYear, err := h.store.CountMomentDaysInYear(userID, now.Year())
	if err != nil {
		return models.Stats{}, err
	}

	stats := models.Stats{
		TotalMoments:   int(total),
		CurrentStreak:  gratitude.Streak(days, now),
		LongestStreak:  gratitude.LongestStreak(days),
		EveningMoments: int(evening),
		DaysThisYear:   int(daysThisYear),
		Year:           now.Year(),
	}
	if len(days) > 0 {
		stats.FirstMomentDay = days[0]
		stats.LastMomentDay = days[len(days)-1]
	}
	return stats, nil
}

// milestoneView decorates a milestone with its translated copy and progress.
type milestoneView struct {
	Key     string
	Title   string
	Body    string
	Value   int
	Goal    int
	Percent int
	Reached bool
}

func (h *DiaryHandler) milestoneViews(catalog *i18n.Catalog, milestones []models.Milestone) []milestoneView {
	views := make([]milestoneView, 0, len(milestones))
	for _, milestone := range milestones {
		percent := 0
		if milestone.Goal > 0 {
			percent = int(math.Round(float64(milestone.Value) / float64(milestone.Goal) * 100))
		}
		if percent > 100 {
			percent = 100
		}
		views = append(views, milestoneView{
			Key:     milestone.Key,
			Title:   catalog.T("milestone." + milestone.Key + ".title"),
			Body:    catalog.T("milestone."+milestone.Key+".body", milestone.Value),
			Value:   milestone.Value,
			Goal:    milestone.Goal,
			Percent: percent,
			Reached: milestone.Reached,
		})
	}
	return views
}

// themeView is one row of "Temas do Coração" with its bar width.
type themeView struct {
	Name    string
	Slug    string
	Count   int
	Percent int
}

func themeViews(stats []models.TagStat) []themeView {
	views := make([]themeView, 0, len(stats))
	top := 0
	for _, stat := range stats {
		if stat.Count > top {
			top = stat.Count
		}
	}
	for _, stat := range stats {
		percent := 0
		if top > 0 {
			percent = int(math.Round(float64(stat.Count) / float64(top) * 100))
		}
		views = append(views, themeView{Name: stat.Tag.Name, Slug: stat.Tag.Slug, Count: stat.Count, Percent: percent})
	}
	return views
}

// serenityChart is the polyline + area path for the 30 day curve.
type serenityChart struct {
	Path       string
	Area       string
	Points     []serenityDot
	FirstLabel string
	LastLabel  string
}

type serenityDot struct {
	X      int
	Y      int
	Day    string
	Serene int
}

// buildSerenityChart maps the series onto a fixed 0..320 x 0..120 viewBox so
// the SVG scales without recomputing coordinates in the browser.
func buildSerenityChart(series []models.SerenityPoint) serenityChart {
	const (
		width   = 320
		height  = 120
		padding = 12
	)
	chart := serenityChart{}
	if len(series) == 0 {
		return chart
	}

	span := width - padding*2
	step := 0
	if len(series) > 1 {
		step = span / (len(series) - 1)
	}

	coords := make([]string, 0, len(series))
	for i, point := range series {
		x := padding + i*step
		ratio := float64(point.Serenity-models.SerenityMin) / float64(models.SerenityScaleSpan)
		y := height - padding - int(math.Round(ratio*float64(height-padding*2)))
		coords = append(coords, fmt.Sprintf("%d,%d", x, y))
		chart.Points = append(chart.Points, serenityDot{
			X:      x,
			Y:      y,
			Day:    point.Day.Format("02/01"),
			Serene: point.Serenity,
		})
	}

	chart.Path = "M" + strings.Join(coords, " L")
	chart.Area = fmt.Sprintf("%s L%d,%d L%d,%d Z", chart.Path, padding+(len(series)-1)*step, height, padding, height)
	chart.FirstLabel = series[0].Day.Format("02/01")
	chart.LastLabel = series[len(series)-1].Day.Format("02/01")
	return chart
}

// serenityDelta compares the first and last third of the window, so a single
// bad day cannot flip the headline number.
func serenityDelta(series []models.SerenityPoint) int {
	if len(series) < 4 {
		return 0
	}
	third := len(series) / 3
	early := averageSerenity(series[:third])
	late := averageSerenity(series[len(series)-third:])
	if early == 0 {
		return 0
	}
	return int(math.Round((late - early) / early * 100))
}

func averageSerenity(points []models.SerenityPoint) float64 {
	if len(points) == 0 {
		return 0
	}
	sum := 0
	for _, point := range points {
		sum += point.Serenity
	}
	return float64(sum) / float64(len(points))
}

func predominantMood(moments []models.Moment) models.Mood {
	counts := map[models.Mood]int{}
	for _, moment := range moments {
		counts[moment.Mood]++
	}
	best := models.MoodSerene
	bestCount := -1
	ordered := append([]models.Mood{}, models.MoodValues...)
	sort.SliceStable(ordered, func(i, j int) bool { return counts[ordered[i]] > counts[ordered[j]] })
	for _, mood := range ordered {
		if counts[mood] > bestCount {
			best, bestCount = mood, counts[mood]
		}
	}
	return best
}

// markedMemory picks the most substantial recent entry to highlight.
func markedMemory(moments []models.Moment) *models.Moment {
	var best *models.Moment
	for i := range moments {
		if best == nil || len([]rune(moments[i].Body)) > len([]rune(best.Body)) {
			best = &moments[i]
		}
	}
	return best
}

// validClockTime accepts the "HH:MM" shape used by the ritual time input.
func validClockTime(value string) bool {
	if len(value) != 5 || value[2] != ':' {
		return false
	}
	for _, i := range []int{0, 1, 3, 4} {
		if value[i] < '0' || value[i] > '9' {
			return false
		}
	}
	hour, _ := strconv.Atoi(value[:2])
	minute, _ := strconv.Atoi(value[3:])
	return hour < 24 && minute < 60
}
