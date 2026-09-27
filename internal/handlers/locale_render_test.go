package handlers

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/puppe1990/amarra-cais/pkg/cais"
	caisi18n "github.com/puppe1990/amarra-cais/pkg/cais/i18n"

	appi18n "github.com/puppe1990/grato/internal/i18n"
)

// asLocale drives a handler through LocaleMiddleware with no cookie or query
// string, so the fallback decides the catalog. That is the only place a
// request catalog comes from, which is how a language switch reaches a
// handler in production too.
func asLocale(t *testing.T, handler http.HandlerFunc, req *http.Request, locale string) *httptest.ResponseRecorder {
	t.Helper()
	rr := httptest.NewRecorder()
	withLocale := caisi18n.LocaleMiddleware(appi18n.Catalogs(), locale)
	withLocale(handler).ServeHTTP(rr, req)
	return rr
}

// Templates bind their `t` function when they are parsed, so a single
// renderer is frozen in one locale however the request negotiates. The
// language toggle is only honest if the copy follows the request catalog.
func TestPages_translateWithTheRequestedLocale(t *testing.T) {
	cases := []struct{ locale, wantToday string }{
		{"en", ">Today</a>"},
		{"pt", ">Hoje</a>"},
	}

	for _, tc := range cases {
		h, _, user, _ := newDiaryHandler(t)
		req := signedIn(user.ID, http.MethodGet, "/hoje", nil)
		body := asLocale(t, h.Today, req, tc.locale).Body.String()

		if !strings.Contains(body, tc.wantToday) {
			t.Errorf("locale %s: navigation should say %q", tc.locale, tc.wantToday)
		}
	}
}

// The sidebar is the one piece of copy that never moved with the toggle: its
// tagline sat in the template as a literal while the catalog key existed
// unused beside it.
func TestSidebar_taglineComesFromTheCatalog(t *testing.T) {
	cases := map[string]string{
		"en": "Gratitude Journal",
		"pt": "Diário da Gratidão",
	}

	for locale, want := range cases {
		h, _, user, _ := newDiaryHandler(t)
		req := signedIn(user.ID, http.MethodGet, "/hoje", nil)
		body := asLocale(t, h.Today, req, locale).Body.String()

		if !strings.Contains(body, want) {
			t.Errorf("locale %s: sidebar tagline should read %q", locale, want)
		}
	}
}

// Handler-built copy has to follow the request too. The page title in the
// header comes from h.catalog, not from a template, so it stayed in the boot
// locale while the navigation around it switched — the toggle looked half
// applied.
func TestPageTitle_translatesWithTheRequestedLocale(t *testing.T) {
	cases := map[string]string{
		"en": "<title>Today · Grato</title>",
		"pt": "<title>Hoje · Grato</title>",
	}

	for locale, want := range cases {
		h, _, user, _ := newDiaryHandler(t)
		req := signedIn(user.ID, http.MethodGet, "/hoje", nil)
		body := asLocale(t, h.Today, req, locale).Body.String()

		if !strings.Contains(body, want) {
			t.Errorf("locale %s: expected %q in the head", locale, want)
		}
	}
}

// The public pages ride the same mechanism.
func TestAuthPages_translateWithTheRequestedLocale(t *testing.T) {
	s := setupTestStore(t)
	h := NewAuthHandler(setupTestViews(t), s, testSite(), s.Sessions(), cais.Config{}, appi18n.NewCatalog("pt"))

	req := httptest.NewRequest(http.MethodGet, "/login", nil)
	body := asLocale(t, h.Login, req, "en").Body.String()

	if !strings.Contains(body, "Good to have you back") {
		t.Error("login copy should follow the request locale")
	}
}
