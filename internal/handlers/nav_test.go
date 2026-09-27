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
	normalisedBody := normalised(body)
	start := strings.Index(normalisedBody, marker)
	if start < 0 {
		return "", false
	}
	open := strings.LastIndex(normalisedBody[:start], "<")
	end := strings.Index(normalisedBody[start:], ">")
	if open < 0 || end < 0 {
		return "", false
	}
	return normalisedBody[open : start+end], true
}

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

// The tab bar must say which tab is current. Relying on colour alone failed:
// the write button's filled circle sat directly above its label, so an orange
// label under an orange circle read as part of the button rather than as
// "you are here".
func TestBottomNav_marksOnlyTheCurrentTab(t *testing.T) {
	cases := []struct {
		path     string
		active   string
		inactive []string
	}{
		{"/hoje", "nav-today", []string{"nav-register", "nav-memories", "nav-insights"}},
		{"/registrar", "nav-register", []string{"nav-today", "nav-memories", "nav-insights"}},
		{"/memorias", "nav-memories", []string{"nav-today", "nav-register", "nav-insights"}},
		{"/insights", "nav-insights", []string{"nav-today", "nav-register", "nav-memories"}},
	}

	for _, tc := range cases {
		h, _, user, _ := newDiaryHandler(t)
		body := renderDiaryPage(t, h, user.ID, tc.path).Body.String()

		tag, ok := tagWith(body, tc.active)
		if !ok {
			t.Fatalf("%s: %s not rendered", tc.path, tc.active)
		}
		if !strings.Contains(tag, `aria-current="page"`) {
			t.Errorf("%s: %s should be aria-current=page, got: %s", tc.path, tc.active, tag)
		}
		for _, other := range tc.inactive {
			tag, ok := tagWith(body, other)
			if !ok {
				t.Fatalf("%s: %s not rendered", tc.path, other)
			}
			if strings.Contains(tag, `aria-current`) {
				t.Errorf("%s: %s must not be current, got: %s", tc.path, other, tag)
			}
		}
	}
}

// The write button's fill is what marks its tab as current, so the rule that
// turns aria-current into a filled circle has to exist, and the resting state
// must not paint one.
func TestStylesheet_fillsTheWriteButtonOnlyWhenCurrent(t *testing.T) {
	source, err := os.ReadFile(filepath.Join(projectRoot(t), "input.css"))
	if err != nil {
		t.Fatal(err)
	}
	css := string(source)

	resting, ok := cssBlock(css, ".hearth-nav-fab {")
	if !ok {
		t.Fatal("input.css has no .hearth-nav-fab rule")
	}
	if strings.Contains(resting, "bg-") {
		t.Errorf("the resting write tab paints a badge the other tabs lack: %s", resting)
	}
	if !strings.Contains(resting, "h-5 w-5") {
		t.Errorf("the resting write tab should be icon-sized, got: %s", resting)
	}

	current, ok := cssBlock(css, `.hearth-nav-link[aria-current="page"] .hearth-nav-fab {`)
	if !ok {
		t.Fatal("input.css is missing the current-tab rule for the write button")
	}
	for _, want := range []string{"bg-terracotta", "h-12 w-12"} {
		if !strings.Contains(current, want) {
			t.Errorf("the current write tab should be a filled button (%s), got: %s", want, current)
		}
	}
}

// cssBlock returns the declarations of the first rule with the given selector.
func cssBlock(css, selector string) (string, bool) {
	start := strings.Index(css, selector)
	if start < 0 {
		return "", false
	}
	open := strings.Index(css[start:], "{")
	close := strings.Index(css[start:], "}")
	if open < 0 || close < 0 || close < open {
		return "", false
	}
	return css[start+open+1 : start+close], true
}

// The rendered write button must not carry a hardcoded fill: the fill is the
// selected state, and a permanent one is the bug this guards.
func TestBottomNav_writeButtonIsNotPermanentlyFilled(t *testing.T) {
	h, _, user, _ := newDiaryHandler(t)
	norm := normalised(renderDiaryPage(t, h, user.ID, "/hoje").Body.String())

	for _, forbidden := range []string{"hearth-nav-fab bg-terracotta ", "hearth-nav-fab bg-terracotta\""} {
		if strings.Contains(norm, forbidden) {
			t.Errorf("write button hardcodes the selected fill: %q", forbidden)
		}
	}
	if !strings.Contains(norm, `class="hearth-nav-fab"`) {
		t.Error("write button should rely on .hearth-nav-fab alone")
	}
}
