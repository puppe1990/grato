package handlers

import (
	"net/http"
	"strings"
	"time"

	"github.com/puppe1990/amarra-cais/pkg/amarra/view"
	"github.com/puppe1990/amarra-cais/pkg/cais"
	"github.com/puppe1990/amarra-cais/pkg/cais/i18n"
	"github.com/puppe1990/amarra-cais/pkg/cais/meta"
	"github.com/puppe1990/amarra-cais/pkg/cais/session"

	"github.com/puppe1990/grato/internal/gratitude"
	"github.com/puppe1990/grato/internal/models"
	"github.com/puppe1990/grato/internal/store"
)

const (
	todayPath    = "/hoje"
	registerPath = "/registrar"
	memoriesPath = "/memorias"
	insightsPath = "/insights"
)

// DiaryHandler renders the four diary screens. Routes reaching it sit behind
// RequireAuth, but it re-checks the session anyway so a stale cookie can never
// render an empty journal as if it belonged to the visitor.
type DiaryHandler struct {
	views   *view.Renderer
	store   store.Store
	site    meta.Site
	catalog *i18n.Catalog
	cfg     cais.Config
	now     func() time.Time
}

func NewDiaryHandler(views *view.Renderer, s store.Store, site meta.Site, catalog *i18n.Catalog, cfg cais.Config) *DiaryHandler {
	return &DiaryHandler{views: views, store: s, site: site, catalog: catalog, cfg: cfg, now: time.Now}
}

// locale is the catalog locale negotiated for this request.
func (h *DiaryHandler) locale(r *http.Request) string {
	if catalog := i18n.CatalogFromRequest(r); catalog != nil {
		return catalog.Locale()
	}
	return h.catalog.Locale()
}

func (h *DiaryHandler) currentUser(w http.ResponseWriter, r *http.Request) (models.User, bool) {
	userID, ok := session.UserID(r)
	if !ok {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return models.User{}, false
	}
	user, err := h.store.FindUserByID(userID)
	if err != nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return models.User{}, false
	}
	return user, true
}

func (h *DiaryHandler) render(w http.ResponseWriter, r *http.Request, user *models.User, name string, extra map[string]any, status int) {
	data := amarraData(r, h.site, h.catalog, extra)
	if user != nil {
		h.addHeaderData(data, *user)
	}
	writeView(w, r, h.views, h.cfg, "app", name, data, status)
}

// addHeaderData feeds the persistent chrome: the greeting name, the avatar
// initial and the streak badge. They live inside #amarra-main so a Drive morph
// after writing a moment never leaves a stale streak on screen.
func (h *DiaryHandler) addHeaderData(data map[string]any, user models.User) {
	name := user.DisplayName()
	data["UserName"] = name
	data["UserInitial"] = initialOf(name)

	days, err := h.store.MomentDays(user.ID)
	if err != nil {
		return
	}
	streak := gratitude.Streak(days, h.now())
	data["StreakDays"] = streak
	data["StreakBadge"] = h.catalog.T("today.days_badge", streak)
}

// initialOf renders the avatar glyph, tolerating an empty name.
func initialOf(name string) string {
	for _, r := range strings.TrimSpace(name) {
		return strings.ToUpper(string(r))
	}
	return "•"
}

func (h *DiaryHandler) fail(w http.ResponseWriter, err error) {
	http.Error(w, err.Error(), http.StatusInternalServerError)
}

// moodChips renders "Como você está agora?" with the active chip selected.
func (h *DiaryHandler) moodChips(active models.Mood, target string) []moodChip {
	chips := make([]moodChip, 0, len(models.MoodValues))
	for _, mood := range models.MoodValues {
		chips = append(chips, moodChip{
			Value:  string(mood),
			Label:  h.catalog.T("mood." + string(mood)),
			Active: mood == active,
			URL:    target + "?mood=" + string(mood),
		})
	}
	return chips
}

// moodChip is one feeling button.
type moodChip struct {
	Value  string
	Label  string
	Active bool
	URL    string
}
