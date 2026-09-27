package handlers

import (
	"net/http"

	"github.com/puppe1990/amarra-cais/pkg/amarra/view"
	"github.com/puppe1990/amarra-cais/pkg/cais"
	"github.com/puppe1990/amarra-cais/pkg/cais/i18n"
	"github.com/puppe1990/amarra-cais/pkg/cais/meta"
	"github.com/puppe1990/amarra-cais/pkg/cais/session"

	"github.com/puppe1990/grato/internal/store"
)

// Entry sends visitors to their journal, or to the door when signed out.
// The diary has no marketing landing page: the first screen is the day itself.
func (h *DiaryHandler) Entry(w http.ResponseWriter, r *http.Request) {
	if _, ok := session.UserID(r); ok {
		http.Redirect(w, r, todayPath, http.StatusSeeOther)
		return
	}
	http.Redirect(w, r, "/login", http.StatusSeeOther)
}

// NotFound renders the designed 404 inside the private layout when the
// visitor is signed in, and a public shell otherwise.
func NotFound(views *view.Renderer, site meta.Site, catalog *i18n.Catalog, cfg cais.Config, s store.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		layout, nav := "auth", ""
		if userID, ok := session.UserID(r); ok {
			if _, err := s.FindUserByID(userID); err == nil {
				layout, nav = "app", "today"
			}
		}
		writeView(w, r, views, cfg, layout, "not_found", amarraData(r, site, map[string]any{
			"Title":     catalog.T("error.not_found_title"),
			"ActiveNav": nav,
		}), http.StatusNotFound)
	}
}
