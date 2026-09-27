package handlers

import (
	"net/http"

	appi18n "github.com/puppe1990/grato/internal/i18n"

	"github.com/puppe1990/amarra-cais/pkg/cais/i18n"

	"github.com/puppe1990/grato/internal/gratitude"

	"github.com/puppe1990/grato/internal/models"
)

// Today is the Hoje screen: greeting, daily reflection, the three gratitude
// checkpoints of the day and the streak.
func (h *DiaryHandler) Today(w http.ResponseWriter, r *http.Request) {
	user, ok := h.currentUser(w, r)
	if !ok {
		return
	}

	today := h.now()
	moments, err := h.store.MomentsOnDay(user.ID, today)
	if err != nil {
		h.fail(w, err)
		return
	}

	reflections := appi18n.ReflectionList(h.locale(r))
	checkpoints, filled := h.checkpointsFor(h.catalogFor(r), moments)

	h.render(w, r, &user, "today", map[string]any{
		"Title":       h.catalogFor(r).T("nav.today"),
		"ActiveNav":   "today",
		"Greeting":    appi18n.Greeting(h.locale(r), today),
		"FullDate":    appi18n.FullDate(h.locale(r), today),
		"Reflection":  reflections[gratitude.ReflectionOfDay(today, len(reflections))],
		"Moods":       h.moodChips(h.catalogFor(r), todayMood(moments), registerPath),
		"Checkpoints": checkpoints,
		"Filled":      filled,
		"Total":       len(models.MomentOfDayValues),
		"HasMoments":  len(moments) > 0,
	}, 0)
}

// checkpointView is one of the day's three gratitude checkpoints.
type checkpointView struct {
	OfDay  models.MomentOfDay
	Label  string
	Prompt string
	URL    string
	Filled bool
	Time   string
	Moment *models.Moment
	Tags   []models.Tag
}

func (h *DiaryHandler) checkpointsFor(catalog *i18n.Catalog, moments []models.Moment) ([]checkpointView, int) {
	byOfDay := map[models.MomentOfDay]*models.Moment{}
	for i := range moments {
		if _, seen := byOfDay[moments[i].MomentOfDay]; !seen {
			byOfDay[moments[i].MomentOfDay] = &moments[i]
		}
	}

	checkpoints := make([]checkpointView, 0, len(models.MomentOfDayValues))
	filled := 0
	for _, ofDay := range models.MomentOfDayValues {
		view := checkpointView{
			OfDay:  ofDay,
			Label:  catalog.T("checkpoint." + string(ofDay)),
			Prompt: catalog.T("today.moments_hint"),
			URL:    registerPath + "?moment_of_day=" + string(ofDay),
		}
		if moment, ok := byOfDay[ofDay]; ok {
			view.Filled = true
			view.Moment = moment
			view.Tags = moment.Tags
			view.Time = appi18n.ClockTime(moment.HappenedAt)
			filled++
		}
		checkpoints = append(checkpoints, view)
	}
	return checkpoints, filled
}

// todayMood is the feeling to highlight: the most recent one written today.
func todayMood(moments []models.Moment) models.Mood {
	if len(moments) == 0 {
		return ""
	}
	return moments[len(moments)-1].Mood
}
