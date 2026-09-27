package handlers

import (
	"io/fs"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/puppe1990/amarra-cais/pkg/cais"

	appi18n "github.com/puppe1990/grato/internal/i18n"
	"github.com/puppe1990/grato/internal/models"
)

// The layout passes Locale ("pt-BR") to the toggle while the component
// compares against "pt"/"en", so neither button ever looked selected.
func TestLayout_languageToggleMarksTheActiveLocale(t *testing.T) {
	cases := map[string]string{"pt": "pt", "en": "en"}
	for locale, expected := range cases {
		s := setupTestStore(t)
		h := NewDiaryHandler(setupTestViews(t), s, testSite(), appi18n.NewCatalog(locale), cais.Config{})
		h.now = func() time.Time { return fixedDay }
		user, err := s.CreateUser(models.User{Name: "Sofia", Email: "sofia@example.com", PasswordHash: "hash"})
		if err != nil {
			t.Fatal(err)
		}

		rr := httptest.NewRecorder()
		h.Today(rr, signedIn(user, http.MethodGet, "/hoje", nil))
		body := rr.Body.String()

		for _, button := range []string{"en", "pt"} {
			tag, ok := localeButton(body, button)
			if !ok {
				t.Fatalf("%s: language toggle missing the %s button", locale, strings.ToUpper(button))
			}
			wantPressed := button == expected
			gotPressed := strings.Contains(tag, `aria-pressed="true"`)
			if gotPressed != wantPressed {
				t.Errorf("%s: %s aria-pressed = %v, want %v", locale, strings.ToUpper(button), gotPressed, wantPressed)
			}
		}
	}
}

// localeButton returns the opening tag of the toggle button for a language.
func localeButton(body, locale string) (string, bool) {
	marker := `data-amarra-locale="` + locale + `"`
	start := strings.Index(body, marker)
	if start < 0 {
		return "", false
	}
	end := strings.Index(body[start:], ">")
	if end < 0 {
		return "", false
	}
	return body[start : start+end], true
}

// The scaffold ships a dark palette (ink/foam/copper/tide). This app replaced
// it with Serene Hearth, so any leftover class silently renders with no CSS at
// all — which is exactly how the account footer ended up unstyled.
func TestTemplates_avoidTheScaffoldPalette(t *testing.T) {
	dead := []string{"foam", "ink", "copper", "tide"}
	prefixes := []string{"text", "bg", "border", "ring", "divide", "hover:text", "hover:bg", "accent", "fill", "stroke"}

	root := projectRoot(t)
	dir := filepath.Join(root, "web", "templates")
	err := filepath.WalkDir(dir, func(path string, entry fs.DirEntry, err error) error {
		if err != nil || entry.IsDir() {
			return err
		}
		source, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(root, path)
		for _, prefix := range prefixes {
			for _, name := range dead {
				class := prefix + "-" + name
				if strings.Contains(string(source), class) {
					t.Errorf("%s uses %q, which has no rule in tailwind.config.js", rel, class)
				}
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}
