package handlers

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// A visitor who is not signed in must never be shown the diary chrome: the
// rail links route to screens that only redirect them back, and the sign-out
// form is meaningless before there is a session.
func TestAuthPages_renderThePublicShellOnly(t *testing.T) {
	h, _ := newAuthHandler(t)

	pages := []struct {
		name    string
		path    string
		handler http.HandlerFunc
	}{
		{"login", "/login", h.Login},
		{"signup", "/signup", h.SignUp},
		{"forgot_password", "/forgot-password", h.ForgotPassword},
	}

	chrome := []string{
		`data-testid="nav-today"`,
		`data-testid="nav-register"`,
		`data-testid="nav-memories"`,
		`data-testid="nav-insights"`,
		`action="/logout"`,
		`href="/registrar"`,
	}

	for _, page := range pages {
		rr := httptest.NewRecorder()
		page.handler(rr, httptest.NewRequest(http.MethodGet, page.path, nil))

		if rr.Code != http.StatusOK {
			t.Fatalf("%s: status = %d, want 200", page.name, rr.Code)
		}
		body := rr.Body.String()
		for _, fragment := range chrome {
			if strings.Contains(body, fragment) {
				t.Errorf("%s leaks the diary chrome: %s", page.name, fragment)
			}
		}
		if !strings.Contains(body, `data-amarra-layout="auth"`) {
			t.Errorf("%s should render the public shell", page.name)
		}
	}
}

// Sign-in is the door for a signed-out visitor, so a back arrow there points
// at the page you are already on. The other public pages lead back to it.
func TestAuthPages_backLinkOnlyWhereItLeadsSomewhere(t *testing.T) {
	h, _ := newAuthHandler(t)

	login := httptest.NewRecorder()
	h.Login(login, httptest.NewRequest(http.MethodGet, "/login", nil))
	if strings.Contains(login.Body.String(), `data-testid="back-link"`) {
		t.Error("login should not offer a back link to itself")
	}

	signup := httptest.NewRecorder()
	h.SignUp(signup, httptest.NewRequest(http.MethodGet, "/signup", nil))
	if !strings.Contains(signup.Body.String(), `href="/login"`) {
		t.Error("signup should lead back to sign-in")
	}
}

// The signed-in 404 stays inside the diary, because the visitor still has a
// session and the rail is a useful way back.
func TestNotFound_usesTheDiaryShellWhenSignedIn(t *testing.T) {
	h, s, user, _ := newDiaryHandler(t)
	handler := NotFound(setupTestViews(t), testSite(), h.catalog, h.cfg, s)

	rr := httptest.NewRecorder()
	handler(rr, signedIn(user.ID, http.MethodGet, "/nope", nil))

	body := rr.Body.String()
	if !strings.Contains(body, `data-amarra-layout="app"`) {
		t.Error("signed-in visitors should keep the diary shell")
	}
	if !strings.Contains(body, `data-testid="nav-today"`) {
		t.Error("signed-in visitors should keep the navigation")
	}
}

func TestNotFound_usesThePublicShellWhenSignedOut(t *testing.T) {
	h, s, _, _ := newDiaryHandler(t)
	handler := NotFound(setupTestViews(t), testSite(), h.catalog, h.cfg, s)

	rr := httptest.NewRecorder()
	handler(rr, httptest.NewRequest(http.MethodGet, "/nope", nil))

	body := rr.Body.String()
	if !strings.Contains(body, `data-amarra-layout="auth"`) {
		t.Error("signed-out visitors should get the public shell")
	}
	if strings.Contains(body, `action="/logout"`) {
		t.Error("signed-out visitors must not see a sign-out form")
	}
}
