package handlers

import (
	"errors"
	"net/http"
	"strings"

	"github.com/puppe1990/amarra-cais/pkg/amarra/view"
	"github.com/puppe1990/amarra-cais/pkg/cais"
	"github.com/puppe1990/amarra-cais/pkg/cais/flash"
	"github.com/puppe1990/amarra-cais/pkg/cais/httpx"
	"github.com/puppe1990/amarra-cais/pkg/cais/i18n"
	"github.com/puppe1990/amarra-cais/pkg/cais/meta"
	"github.com/puppe1990/amarra-cais/pkg/cais/passwordreset"
	"github.com/puppe1990/amarra-cais/pkg/cais/session"
	"github.com/puppe1990/amarra-cais/pkg/cais/validate"

	"github.com/puppe1990/grato/internal/models"
	"github.com/puppe1990/grato/internal/store"
)

type AuthHandler struct {
	views       *view.Renderer
	store       store.Store
	site        meta.Site
	sessions    session.Store
	cfg         cais.Config
	catalog     *i18n.Catalog
	resetNotify passwordreset.Notifier
}

func NewAuthHandler(views *view.Renderer, s store.Store, site meta.Site, sessions session.Store, cfg cais.Config, catalog *i18n.Catalog) *AuthHandler {
	return &AuthHandler{views: views, store: s, site: site, sessions: sessions, cfg: cfg, catalog: catalog}
}

func (h *AuthHandler) renderAuth(w http.ResponseWriter, r *http.Request, name string, extra map[string]any, status int) {
	if extra == nil {
		extra = map[string]any{}
	}
	if _, ok := extra["Title"]; !ok {
		extra["Title"] = h.catalog.T("auth.login_title")
	}
	// Public pages are the only ones a crawler ever sees, so each carries its
	// own social description.
	if _, ok := extra["Description"]; !ok {
		if description := pageDescription(h.catalog, name); description != "" {
			extra["Description"] = description
		}
	}
	// Sign-in is the door, so it has nowhere to go back to; the other public
	// pages lead back to it.
	if _, ok := extra["BackHref"]; !ok && name != "login" {
		extra["BackHref"] = "/login"
	}
	// The public shell, not the diary chrome: these pages render before there
	// is a session, so the rail would only offer links back to the door.
	writeView(w, r, h.views, h.cfg, "auth", name, amarraData(r, h.site, h.catalog, extra), status)
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	if _, ok := session.UserID(r); ok {
		http.Redirect(w, r, todayPath, http.StatusSeeOther)
		return
	}
	email := ""
	if h.cfg.Env != "production" {
		email = "demo@example.com"
	}
	h.renderAuth(w, r, "login", map[string]any{
		"Title": h.catalog.T("auth.login_title"),
		"Email": email,
	}, 0)
}

func (h *AuthHandler) LoginPost(w http.ResponseWriter, r *http.Request) {
	if err := httpx.ParseFormOrJSON(r); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	email := strings.TrimSpace(r.FormValue("email"))
	password := r.FormValue("password")
	user, err := h.store.FindUserByEmail(email)
	if err != nil || !session.VerifyPassword(user.PasswordHash, password) {
		h.renderAuth(w, r, "login", map[string]any{
			"Title":  h.catalog.T("auth.login_title"),
			"Email":  email,
			"Errors": validate.FieldErrors{"email": h.catalog.T("auth.invalid_credentials")},
		}, http.StatusUnprocessableEntity)
		return
	}

	if err := session.SignIn(w, h.sessions, r, user.ID, session.CookieOptionsFromConfig(h.cfg)); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	flash.Set(w, "notice", h.catalog.T("auth.welcome"), h.cfg.CookieSecure())
	http.Redirect(w, r, todayPath, http.StatusSeeOther)
}

func (h *AuthHandler) LogoutPost(w http.ResponseWriter, r *http.Request) {
	session.SignOut(w, h.sessions, r, session.CookieOptionsFromConfig(h.cfg))
	http.Redirect(w, r, "/login", http.StatusSeeOther)
}

func (h *AuthHandler) SignUp(w http.ResponseWriter, r *http.Request) {
	if _, ok := session.UserID(r); ok {
		http.Redirect(w, r, todayPath, http.StatusSeeOther)
		return
	}
	h.renderAuth(w, r, "signup", map[string]any{
		"Title":      h.catalog.T("auth.signup_title"),
		"Intention":  string(models.IntentionDefault),
		"Intentions": intentionOptions(h.catalog),
	}, 0)
}

func (h *AuthHandler) SignUpPost(w http.ResponseWriter, r *http.Request) {
	if err := httpx.ParseFormOrJSON(r); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	name := strings.TrimSpace(r.FormValue("name"))
	email := strings.TrimSpace(r.FormValue("email"))
	password := r.FormValue("password")
	confirm := r.FormValue("password_confirmation")
	intention := models.Intention(r.FormValue("intention"))

	var errs validate.FieldErrors
	if name == "" {
		errs.Add("name", h.catalog.T("auth.name_required"))
	}
	if err := validate.Email(email); err != nil {
		errs.Add("email", h.catalog.T("contact.email_invalid"))
	}
	if err := validate.MinLength(password, 8); err != nil {
		errs.Add("password", h.catalog.T("auth.password_too_short"))
	}
	if password != confirm {
		errs.Add("password_confirmation", h.catalog.T("auth.password_mismatch"))
	}
	if !intention.Valid() {
		intention = models.IntentionDefault
	}
	if errs.Any() {
		h.renderSignUpErrors(w, r, name, email, string(intention), errs)
		return
	}

	hash, err := session.HashPassword(password)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	userID, err := h.store.CreateUser(models.User{
		Name:         name,
		Email:        email,
		Intention:    intention,
		PasswordHash: hash,
	})
	if err != nil {
		if errors.Is(err, store.ErrEmailTaken) {
			h.renderSignUpErrors(w, r, name, email, string(intention), validate.FieldErrors{"email": h.catalog.T("auth.email_taken")})
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if err := session.SignIn(w, h.sessions, r, userID, session.CookieOptionsFromConfig(h.cfg)); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	flash.Set(w, "notice", h.catalog.T("auth.welcome"), h.cfg.CookieSecure())
	http.Redirect(w, r, todayPath, http.StatusSeeOther)
}

func (h *AuthHandler) renderSignUpErrors(w http.ResponseWriter, r *http.Request, name, email, intention string, errs validate.FieldErrors) {
	h.renderAuth(w, r, "signup", map[string]any{
		"Title":      h.catalog.T("auth.signup_title"),
		"Name":       name,
		"Email":      email,
		"Intention":  intention,
		"Intentions": intentionOptions(h.catalog),
		"Errors":     errs,
	}, http.StatusUnprocessableEntity)
}

// intentionOption is one "Qual é o seu principal objetivo?" chip.
type intentionOption struct {
	Value string
	Label string
}

func intentionOptions(catalog *i18n.Catalog) []intentionOption {
	options := make([]intentionOption, 0, len(models.IntentionValues))
	for _, value := range models.IntentionValues {
		options = append(options, intentionOption{
			Value: string(value),
			Label: catalog.T("intention." + string(value)),
		})
	}
	return options
}

func (h *AuthHandler) ForgotPassword(w http.ResponseWriter, r *http.Request) {
	if _, ok := session.UserID(r); ok {
		http.Redirect(w, r, todayPath, http.StatusSeeOther)
		return
	}
	h.renderAuth(w, r, "forgot_password", map[string]any{"Title": h.catalog.T("auth.forgot_password_title")}, 0)
}

func (h *AuthHandler) ForgotPasswordPost(w http.ResponseWriter, r *http.Request) {
	if err := httpx.ParseFormOrJSON(r); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	email := strings.TrimSpace(r.FormValue("email"))
	var errs validate.FieldErrors
	if err := validate.Email(email); err != nil {
		errs.Add("email", h.catalog.T("contact.email_invalid"))
	}
	if errs.Any() {
		h.renderAuth(w, r, "forgot_password", map[string]any{
			"Title":  h.catalog.T("auth.forgot_password_title"),
			"Email":  email,
			"Errors": errs,
		}, http.StatusUnprocessableEntity)
		return
	}

	if user, err := h.store.FindUserByEmail(email); err == nil {
		token, err := h.store.CreatePasswordResetToken(user.ID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		_ = h.resetNotifier().NotifyReset(user.Email, token)
	}

	flash.Set(w, "notice", h.catalog.T("auth.reset_email_sent"), h.cfg.CookieSecure())
	http.Redirect(w, r, "/login", http.StatusSeeOther)
}

func (h *AuthHandler) ResetPassword(w http.ResponseWriter, r *http.Request) {
	if _, ok := session.UserID(r); ok {
		http.Redirect(w, r, todayPath, http.StatusSeeOther)
		return
	}

	token := strings.TrimSpace(r.URL.Query().Get("token"))
	extra := map[string]any{"Title": h.catalog.T("auth.reset_password_title"), "Token": token}
	if token == "" {
		extra["Errors"] = validate.FieldErrors{"token": h.catalog.T("auth.reset_invalid_token")}
		h.renderAuth(w, r, "reset_password", extra, http.StatusUnprocessableEntity)
		return
	}
	if _, ok := h.store.FindPasswordResetUserID(token); !ok {
		extra["Errors"] = validate.FieldErrors{"token": h.catalog.T("auth.reset_invalid_token")}
		h.renderAuth(w, r, "reset_password", extra, http.StatusUnprocessableEntity)
		return
	}
	h.renderAuth(w, r, "reset_password", extra, 0)
}

func (h *AuthHandler) ResetPasswordPost(w http.ResponseWriter, r *http.Request) {
	if err := httpx.ParseFormOrJSON(r); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	token := strings.TrimSpace(r.FormValue("token"))
	password := r.FormValue("password")
	confirm := r.FormValue("password_confirmation")

	var errs validate.FieldErrors
	if token == "" {
		errs.Add("token", h.catalog.T("auth.reset_invalid_token"))
	} else if _, ok := h.store.FindPasswordResetUserID(token); !ok {
		h.renderAuth(w, r, "reset_password", map[string]any{
			"Title":  h.catalog.T("auth.reset_password_title"),
			"Token":  token,
			"Errors": validate.FieldErrors{"token": h.catalog.T("auth.reset_invalid_token")},
		}, http.StatusUnprocessableEntity)
		return
	}
	if err := validate.MinLength(password, 8); err != nil {
		errs.Add("password", h.catalog.T("auth.password_too_short"))
	}
	if password != confirm {
		errs.Add("password_confirmation", h.catalog.T("auth.password_mismatch"))
	}
	if errs.Any() {
		h.renderAuth(w, r, "reset_password", map[string]any{
			"Title":  h.catalog.T("auth.reset_password_title"),
			"Token":  token,
			"Errors": errs,
		}, http.StatusUnprocessableEntity)
		return
	}

	hash, err := session.HashPassword(password)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if err := h.store.ResetPasswordWithToken(token, hash); err != nil {
		h.renderAuth(w, r, "reset_password", map[string]any{
			"Title":  h.catalog.T("auth.reset_password_title"),
			"Token":  token,
			"Errors": validate.FieldErrors{"token": h.catalog.T("auth.reset_invalid_token")},
		}, http.StatusUnprocessableEntity)
		return
	}

	flash.Set(w, "notice", h.catalog.T("auth.reset_success"), h.cfg.CookieSecure())
	http.Redirect(w, r, "/login", http.StatusSeeOther)
}

func (h *AuthHandler) resetNotifier() passwordreset.Notifier {
	if h.resetNotify != nil {
		return h.resetNotify
	}
	return passwordreset.NotifierFromConfig(h.cfg, h.site.AppName)
}
