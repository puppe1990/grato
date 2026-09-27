package handlers

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/puppe1990/amarra-cais/pkg/cais/flash"
	"github.com/puppe1990/amarra-cais/pkg/cais/httpx"
	"github.com/puppe1990/amarra-cais/pkg/cais/validate"

	"github.com/puppe1990/grato/internal/models"
)

// registerForm holds the "Novo Registro" state, kept across validation errors.
type registerForm struct {
	MomentOfDay  models.MomentOfDay
	Mood         models.Mood
	Serenity     int
	Title        string
	Body         string
	SelectedTags []string
}

func (h *DiaryHandler) RegisterForm(w http.ResponseWriter, r *http.Request) {
	user, ok := h.currentUser(w, r)
	if !ok {
		return
	}

	ofDay := models.MomentOfDay(r.URL.Query().Get("moment_of_day"))
	if !ofDay.Valid() {
		ofDay = checkpointForHour(h.now().Hour())
	}
	mood := models.Mood(r.URL.Query().Get("mood"))
	if !mood.Valid() {
		mood = models.MoodSerene
	}

	h.renderRegister(w, r, user, registerForm{
		MomentOfDay: ofDay,
		Mood:        mood,
		Serenity:    models.SerenityDefault,
	}, validate.FieldErrors{}, http.StatusOK)
}

func (h *DiaryHandler) RegisterPost(w http.ResponseWriter, r *http.Request) {
	user, ok := h.currentUser(w, r)
	if !ok {
		return
	}
	if err := httpx.ParseFormOrJSON(r); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	form := registerForm{
		MomentOfDay:  checkpointOrDefault(r.FormValue("moment_of_day"), h.now().Hour()),
		Mood:         moodOrDefault(r.FormValue("mood")),
		Serenity:     serenityOrDefault(r.FormValue("serenity")),
		Title:        strings.TrimSpace(r.FormValue("title")),
		Body:         strings.TrimSpace(r.FormValue("body")),
		SelectedTags: submittedTags(r),
	}

	var errs validate.FieldErrors
	if form.Body == "" {
		errs.Add("body", h.catalog.T("register.body_required"))
	}
	if errs.Any() {
		h.renderRegister(w, r, user, form, errs, http.StatusUnprocessableEntity)
		return
	}

	moment := models.Moment{
		UserID:      user.ID,
		MomentOfDay: form.MomentOfDay,
		Mood:        form.Mood,
		Title:       form.Title,
		Body:        form.Body,
		Serenity:    form.Serenity,
		HappenedAt:  h.now(),
	}
	id, err := h.store.InsertMoment(moment)
	if err != nil {
		h.fail(w, err)
		return
	}
	if len(form.SelectedTags) > 0 {
		if err := h.store.AttachTags(id, user.ID, form.SelectedTags); err != nil {
			h.fail(w, err)
			return
		}
	}

	flash.Set(w, "notice", h.catalog.T("register.saved"), h.cfg.CookieSecure())
	http.Redirect(w, r, todayPath, http.StatusSeeOther)
}

func (h *DiaryHandler) renderRegister(w http.ResponseWriter, r *http.Request, user models.User, form registerForm, errs validate.FieldErrors, status int) {
	tags, err := h.store.TagsForUser(user.ID)
	if err != nil {
		h.fail(w, err)
		return
	}

	h.render(w, r, &user, "register", map[string]any{
		"Title":       h.catalog.T("nav.register"),
		"ActiveNav":   "register",
		"Prompt":      h.catalog.T("register.prompt." + string(form.MomentOfDay)),
		"Checkpoints": h.checkpointSwitcher(r, form.MomentOfDay),
		"Bodies":      bodyInspirations(h.catalog),
		"Moods":       h.moodChips(form.Mood, registerPath),
		"TagChoices":  tagChoices(tags, form.SelectedTags),
		"Serenity":    form.Serenity,
		"SerenityMax": models.SerenityMax,
		"Form":        form,
		"Errors":      errs,
	}, status)
}

// checkpointSwitcher renders the "Manhã / Tarde / Noite" picker with counts.
func (h *DiaryHandler) checkpointSwitcher(r *http.Request, active models.MomentOfDay) []moodChip {
	chips := make([]moodChip, 0, len(models.MomentOfDayValues))
	for _, ofDay := range models.MomentOfDayValues {
		chips = append(chips, moodChip{
			Value:  string(ofDay),
			Label:  h.catalog.T("checkpoint." + string(ofDay)),
			Active: ofDay == active,
			URL:    registerPath + "?moment_of_day=" + string(ofDay),
		})
	}
	return chips
}

// tagChoice is one resonance tag offered on the form.
type tagChoice struct {
	Name   string
	Slug   string
	Active bool
}

func tagChoices(tags []models.Tag, selected []string) []tagChoice {
	chosen := map[string]bool{}
	for _, name := range selected {
		chosen[models.SlugifyTag(name)] = true
	}
	choices := make([]tagChoice, 0, len(tags))
	for _, tag := range tags {
		choices = append(choices, tagChoice{Name: tag.Name, Slug: tag.Slug, Active: chosen[tag.Slug]})
	}
	return choices
}

// bodyInspiration is a starter prompt chip above the free reflection box.
type bodyInspiration struct {
	Key   string
	Label string
}

func bodyInspirations(catalog interface{ T(string, ...any) string }) []bodyInspiration {
	keys := []string{"detail", "person", "nature", "other"}
	out := make([]bodyInspiration, 0, len(keys))
	for _, key := range keys {
		out = append(out, bodyInspiration{Key: key, Label: catalog.T("register.inspiration_" + key)})
	}
	return out
}

func checkpointForHour(hour int) models.MomentOfDay {
	switch {
	case hour < 12:
		return models.MomentMorning
	case hour < 18:
		return models.MomentAfternoon
	default:
		return models.MomentEvening
	}
}

func checkpointOrDefault(raw string, hour int) models.MomentOfDay {
	ofDay := models.MomentOfDay(raw)
	if !ofDay.Valid() {
		return checkpointForHour(hour)
	}
	return ofDay
}

func moodOrDefault(raw string) models.Mood {
	mood := models.Mood(raw)
	if !mood.Valid() {
		return models.MoodSerene
	}
	return mood
}

func serenityOrDefault(raw string) int {
	value, err := strconv.Atoi(strings.TrimSpace(raw))
	if err != nil {
		return models.SerenityDefault
	}
	return models.ClampSerenity(value)
}

// submittedTags accepts repeated `tags` fields (checkboxes) and comma or
// space separated single values, plus an optional free-text `new_tag`.
func submittedTags(r *http.Request) []string {
	out := []string{}
	for _, field := range r.Form["tags"] {
		for _, part := range strings.FieldsFunc(field, func(c rune) bool { return c == ',' }) {
			if trimmed := strings.TrimSpace(part); trimmed != "" {
				out = append(out, trimmed)
			}
		}
	}
	for _, part := range strings.FieldsFunc(r.FormValue("new_tag"), func(c rune) bool { return c == ',' }) {
		if trimmed := strings.TrimSpace(part); trimmed != "" {
			out = append(out, trimmed)
		}
	}
	return out
}
