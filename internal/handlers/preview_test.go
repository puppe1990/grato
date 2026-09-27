package handlers

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/puppe1990/amarra-cais/pkg/cais"
	"github.com/puppe1990/amarra-cais/pkg/cais/meta"

	appi18n "github.com/puppe1990/grato/internal/i18n"
)

// The social card only works if every URL in it is absolute against APP_URL:
// crawlers never resolve relative paths, and they never run the app's JS.
func TestDiary_pagesCarryAbsoluteOpenGraphTags(t *testing.T) {
	h, _, user, _ := newDiaryHandler(t)

	rr := httptest.NewRecorder()
	h.Today(rr, signedIn(user.ID, http.MethodGet, "/hoje", nil))

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rr.Code)
	}
	assertPreviewTags(t, rr.Body.String(), "https://cais.example.com/hoje")
}

// The public pages are the Portuguese ones users actually share, so this
// asserts against the pt catalog rather than the English test default.
func TestAuth_pagesCarryTheirOwnPreviewDescription(t *testing.T) {
	s := setupTestStore(t)
	h := NewAuthHandler(setupTestViews(t), s, testSite(), s.Sessions(), cais.Config{}, appi18n.NewCatalog("pt"))

	rr := httptest.NewRecorder()
	h.Login(rr, httptest.NewRequest(http.MethodGet, "/login", nil))

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rr.Code)
	}
	body := rr.Body.String()
	assertPreviewTags(t, body, "https://cais.example.com/login")
	if !strings.Contains(body, "paz interior") {
		t.Errorf("login should describe itself, got: %s", body)
	}
	if !strings.Contains(body, `<meta property="og:locale" content="pt_BR" />`) {
		t.Errorf("missing og:locale pt_BR, got: %s", body)
	}
}

func TestPreviewHead_usesEnglishFallbackWithoutACatalog(t *testing.T) {
	h, _, user, _ := newDiaryHandler(t)

	rr := httptest.NewRecorder()
	h.Today(rr, signedIn(user.ID, http.MethodGet, "/hoje", nil))

	// newDiaryHandler installs the default (English) catalog.
	if !strings.Contains(rr.Body.String(), `<meta property="og:locale" content="en_US" />`) {
		t.Error("og:locale must be language_TERRITORY, not a bare language tag")
	}
}

func TestNotFound_carriesPreviewTags(t *testing.T) {
	h, s, user, _ := newDiaryHandler(t)
	handler := NotFound(setupTestViews(t), testSite(), h.catalog, h.cfg, s)

	rr := httptest.NewRecorder()
	handler(rr, signedIn(user.ID, http.MethodGet, "/nope", nil))

	if rr.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", rr.Code)
	}
	assertPreviewTags(t, rr.Body.String(), "https://cais.example.com/nope")
}

func assertPreviewTags(t *testing.T, body, wantURL string) {
	t.Helper()
	wants := []string{
		`<meta name="description" content="`,
		`<meta property="og:type" content="website" />`,
		`<meta property="og:site_name" content="Grato" />`,
		`<meta property="og:title" content="`,
		`<meta property="og:description" content="`,
		`<meta property="og:image" content="https://cais.example.com/static/og.png" />`,
		`<meta property="og:url" content="` + wantURL + `" />`,
		`<meta name="twitter:card" content="summary_large_image" />`,
		`<meta name="twitter:title" content="`,
		`<meta name="twitter:description" content="`,
		`<meta name="twitter:image" content="https://cais.example.com/static/og.png" />`,
	}
	for _, want := range wants {
		if !strings.Contains(body, want) {
			t.Errorf("missing %s", want)
		}
	}
}

func TestPreviewHead_fallsBackToTheBrandDescription(t *testing.T) {
	h, _, user, _ := newDiaryHandler(t)

	rr := httptest.NewRecorder()
	h.Memories(rr, signedIn(user.ID, http.MethodGet, "/memorias", nil))

	if !strings.Contains(rr.Body.String(), "calmo e privado") {
		t.Error("a page without its own description should fall back to the brand one")
	}
}

// The image has to exist at the path the card advertises, otherwise every
// share renders a broken thumbnail.
func TestOpenGraphImageExists(t *testing.T) {
	root := projectRoot(t)
	if _, err := os.Stat(filepath.Join(root, "web", "static", "og.png")); err != nil {
		t.Fatalf("og.png missing at the advertised path: %v", err)
	}
	if meta.DefaultImagePath != "/static/og.png" {
		t.Errorf("framework default image path changed to %q", meta.DefaultImagePath)
	}
}
