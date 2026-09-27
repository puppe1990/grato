package handlers

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// collapseSpaces makes multi-line template output searchable as one line.
var collapseSpaces = regexp.MustCompile(`\s+`)

func normalised(body string) string {
	return collapseSpaces.ReplaceAllString(body, " ")
}

// tagWith returns the opening tag carrying the given data-testid.
func tagWith(body, testid string) (string, bool) {
	marker := `data-testid="` + testid + `"`
	flat := normalised(body)
	start := strings.Index(flat, marker)
	if start < 0 {
		return "", false
	}
	open := strings.LastIndex(flat[:start], "<")
	end := strings.Index(flat[start:], ">")
	if open < 0 || end < 0 {
		return "", false
	}
	return flat[open : start+end], true
}

func classAttr(tag string) string {
	start := strings.Index(tag, `class="`)
	if start < 0 {
		return ""
	}
	rest := tag[start+len(`class="`):]
	end := strings.Index(rest, `"`)
	if end < 0 {
		return ""
	}
	return rest[:end]
}

var bottomNavTabs = []string{"nav-today", "nav-register", "nav-memories", "nav-insights"}

// renderDiaryPage drives the screen handler that owns a tab path.
func renderDiaryPage(t *testing.T, h *DiaryHandler, userID int64, path string) *httptest.ResponseRecorder {
	t.Helper()
	screens := map[string]http.HandlerFunc{
		"/hoje":      h.Today,
		"/registrar": h.RegisterForm,
		"/memorias":  h.Memories,
		"/insights":  h.Insights,
	}
	handler, ok := screens[path]
	if !ok {
		t.Fatalf("no screen handler for %s", path)
	}
	rr := httptest.NewRecorder()
	handler(rr, signedIn(userID, http.MethodGet, path, nil))
	return rr
}

// The tab bar must say which tab is current, and it must say it the same way
// for all four. The write tab carried a circle of its own for a while, which
// made one tab look like a button while the rest looked like tabs.
func TestBottomNav_tabsAreIndistinguishableApartFromTheCurrentOne(t *testing.T) {
	cases := []struct {
		path   string
		active string
	}{
		{"/hoje", "nav-today"},
		{"/registrar", "nav-register"},
		{"/memorias", "nav-memories"},
		{"/insights", "nav-insights"},
	}

	for _, tc := range cases {
		h, _, user, _ := newDiaryHandler(t)
		body := renderDiaryPage(t, h, user.ID, tc.path).Body.String()

		classes := map[string]string{}
		for _, tab := range bottomNavTabs {
			tag, ok := tagWith(body, tab)
			if !ok {
				t.Fatalf("%s: %s not rendered", tc.path, tab)
			}
			classes[tab] = classAttr(tag)

			isCurrent := strings.Contains(tag, `aria-current="page"`)
			if isCurrent != (tab == tc.active) {
				t.Errorf("%s: %s aria-current = %v, want %v", tc.path, tab, isCurrent, tab == tc.active)
			}
		}

		// Same class list on every tab: the current state rides on
		// aria-current, so nothing has to be applied per tab at render time.
		for _, tab := range bottomNavTabs {
			if classes[tab] != classes[bottomNavTabs[0]] {
				t.Errorf("%s: %s has class %q but %s has %q — tabs must match",
					tc.path, tab, classes[tab], bottomNavTabs[0], classes[bottomNavTabs[0]])
			}
		}
	}
}

// No tab may carry markup its siblings lack — that is what made the write tab
// stand out, and a class-list comparison alone would not catch it.
func TestBottomNav_noTabCarriesExtraMarkup(t *testing.T) {
	h, _, user, _ := newDiaryHandler(t)
	body := renderDiaryPage(t, h, user.ID, "/hoje").Body.String()

	flat := normalised(body)
	open := strings.Index(flat, `<nav class="fixed`)
	if open < 0 {
		t.Fatal("bottom navigation not found")
	}
	end := strings.Index(flat[open:], "</nav>")
	if strings.Contains(flat[open:open+end], "hearth-nav-fab") {
		t.Error("a tab carries a badge the others do not")
	}

	for _, tab := range bottomNavTabs {
		tag, ok := tagWith(body, tab)
		if !ok {
			t.Fatalf("%s not rendered", tab)
		}
		if strings.Contains(tag, "<span") {
			t.Errorf("%s wraps its icon in extra markup: %s", tab, tag)
		}
	}
}

// The stylesheet must not reintroduce a per-tab decoration.
func TestStylesheet_hasNoWriteTabDecoration(t *testing.T) {
	source, err := os.ReadFile(filepath.Join(projectRoot(t), "input.css"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(source), "hearth-nav-fab") {
		t.Error("input.css still styles a write-tab badge")
	}
}
