package handlers

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/brianvoe/gofakeit/v7"
	"github.com/puppe1990/amarra-cais/pkg/cais"
	"github.com/puppe1990/amarra-cais/pkg/cais/session"

	appi18n "github.com/puppe1990/grato/internal/i18n"

	"github.com/puppe1990/grato/internal/models"
	"github.com/puppe1990/grato/internal/store"
	"github.com/puppe1990/grato/internal/testdata"
)

// fixedDay is the day every diary test pretends it is.
var fixedDay = time.Date(2026, time.October, 24, 9, 30, 0, 0, time.Local)

func newDiaryHandler(t *testing.T) (*DiaryHandler, store.Store, models.User, *gofakeit.Faker) {
	t.Helper()
	s := setupTestStore(t)
	f := testdata.New()
	h := NewDiaryHandler(setupTestViews(t), s, testSite(), appi18n.DefaultCatalog(), cais.Config{})
	h.now = func() time.Time { return fixedDay }

	user := models.User{Name: "Sofia", Email: "sofia@example.com", PasswordHash: "hash"}
	id, err := s.CreateUser(user)
	if err != nil {
		t.Fatalf("CreateUser: %v", err)
	}
	user.ID = id
	return h, s, user, f
}

func signedIn(userID int64, method, target string, body url.Values) *http.Request {
	var req *http.Request
	if body == nil {
		req = httptest.NewRequest(method, target, nil)
	} else {
		req = httptest.NewRequest(method, target, strings.NewReader(body.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	}
	if userID > 0 {
		req = session.WithUserID(req, userID)
	}
	return req
}

func seedMoment(t *testing.T, s store.Store, f *gofakeit.Faker, userID int64, day time.Time, ofDay models.MomentOfDay) models.Moment {
	t.Helper()
	moment := testdata.MomentAt(f, userID, day)
	moment.MomentOfDay = ofDay
	moment.Serenity = models.SerenityBalanced
	id, err := s.InsertMoment(moment)
	if err != nil {
		t.Fatalf("InsertMoment: %v", err)
	}
	moment.ID = id
	return moment
}

func TestDiary_requiresAuthentication(t *testing.T) {
	h, _, _, _ := newDiaryHandler(t)

	cases := map[string]http.HandlerFunc{
		"today":    h.Today,
		"register": h.RegisterForm,
		"memories": h.Memories,
		"insights": h.Insights,
	}
	for name, handler := range cases {
		rr := httptest.NewRecorder()
		handler(rr, signedIn(0, http.MethodGet, "/", nil))
		if rr.Code != http.StatusSeeOther {
			t.Errorf("%s: status = %d, want 303 redirect to login", name, rr.Code)
		}
		if got := rr.Header().Get("Location"); got != "/login" {
			t.Errorf("%s: Location = %q, want /login", name, got)
		}
	}
}

func TestToday_greetsTheUserAndListsThreeCheckpoints(t *testing.T) {
	h, s, user, f := newDiaryHandler(t)
	seedMoment(t, s, f, user.ID, fixedDay, models.MomentMorning)

	rr := httptest.NewRecorder()
	h.Today(rr, signedIn(user.ID, http.MethodGet, "/hoje", nil))

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200, body: %s", rr.Code, rr.Body.String())
	}
	body := rr.Body.String()
	if !strings.Contains(body, `data-testid="today-greeting"`) {
		t.Error("missing greeting block")
	}
	if !strings.Contains(body, "Sofia") {
		t.Error("greeting should include the user's name")
	}
	if got := strings.Count(body, `data-testid="today-checkpoint"`); got != 3 {
		t.Errorf("checkpoints = %d, want 3", got)
	}
	if !strings.Contains(body, `data-testid="today-streak"`) {
		t.Error("missing streak badge")
	}
	if !strings.Contains(body, `data-testid="today-reflection"`) {
		t.Error("missing daily reflection")
	}
}

func TestToday_countsFilledCheckpoints(t *testing.T) {
	h, s, user, f := newDiaryHandler(t)
	seedMoment(t, s, f, user.ID, fixedDay, models.MomentMorning)
	seedMoment(t, s, f, user.ID, fixedDay, models.MomentAfternoon)
	// Yesterday's evening entry must not count towards today.
	seedMoment(t, s, f, user.ID, fixedDay.AddDate(0, 0, -1), models.MomentEvening)

	rr := httptest.NewRecorder()
	h.Today(rr, signedIn(user.ID, http.MethodGet, "/hoje", nil))

	if !strings.Contains(rr.Body.String(), `data-filled="true"`) {
		t.Error("expected filled checkpoints")
	}
	if got := strings.Count(rr.Body.String(), `data-filled="true"`); got != 2 {
		t.Errorf("filled = %d, want 2", got)
	}
}

func TestRegisterForm_rendersTagChoicesFromUserTags(t *testing.T) {
	h, s, user, f := newDiaryHandler(t)
	moment := seedMoment(t, s, f, user.ID, fixedDay, models.MomentMorning)
	if err := s.AttachTags(moment.ID, user.ID, []string{"paz", "natureza"}); err != nil {
		t.Fatalf("AttachTags: %v", err)
	}

	rr := httptest.NewRecorder()
	h.RegisterForm(rr, signedIn(user.ID, http.MethodGet, "/registrar", nil))

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rr.Code)
	}
	if !strings.Contains(rr.Body.String(), `value="paz"`) {
		t.Error("missing existing tag choice")
	}
}

func TestRegisterPost_storesMomentAndRedirects(t *testing.T) {
	h, s, user, _ := newDiaryHandler(t)

	form := url.Values{
		"moment_of_day": {"tarde"},
		"mood":          {"grato"},
		"serenity":      {"5"},
		"title":         {"Caminhada no parque"},
		"body":          {"Caminhei devagar e notei o cheiro da terra molhada."},
		"tags":          {"natureza", "paz"},
	}
	rr := httptest.NewRecorder()
	h.RegisterPost(rr, signedIn(user.ID, http.MethodPost, "/registrar", form))

	if rr.Code != http.StatusSeeOther {
		t.Fatalf("status = %d, want 303, body: %s", rr.Code, rr.Body.String())
	}
	if got := rr.Header().Get("Location"); got != todayPath {
		t.Errorf("Location = %q, want %q", got, todayPath)
	}

	moments, err := s.MomentsOnDay(user.ID, fixedDay)
	if err != nil {
		t.Fatalf("MomentsOnDay: %v", err)
	}
	if len(moments) != 1 {
		t.Fatalf("len(moments) = %d, want 1", len(moments))
	}
	if moments[0].Title != "Caminhada no parque" {
		t.Errorf("Title = %q", moments[0].Title)
	}
	if moments[0].Serenity != models.SerenityMax {
		t.Errorf("Serenity = %d, want %d", moments[0].Serenity, models.SerenityMax)
	}
	if len(moments[0].Tags) != 2 {
		t.Errorf("len(Tags) = %d, want 2", len(moments[0].Tags))
	}
}

func TestRegisterPost_rejectsEmptyBody(t *testing.T) {
	h, s, user, _ := newDiaryHandler(t)

	form := url.Values{"moment_of_day": {"noite"}, "body": {"   "}}
	rr := httptest.NewRecorder()
	h.RegisterPost(rr, signedIn(user.ID, http.MethodPost, "/registrar", form))

	if rr.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d, want 422", rr.Code)
	}
	if !strings.Contains(rr.Body.String(), "at least one line") {
		t.Errorf("missing validation message, got: %s", rr.Body.String())
	}
	count, err := s.CountMoments(user.ID)
	if err != nil {
		t.Fatalf("CountMoments: %v", err)
	}
	if count != 0 {
		t.Errorf("CountMoments = %d, want 0", count)
	}
}

func TestRegisterPost_fallsBackToSafeDefaultsForUnknownValues(t *testing.T) {
	h, s, user, _ := newDiaryHandler(t)

	form := url.Values{
		"moment_of_day": {"madrugada"},
		"mood":          {"euforico"},
		"serenity":      {"nao-e-numero"},
		"body":          {"Um detalhe pequeno e bom."},
	}
	rr := httptest.NewRecorder()
	h.RegisterPost(rr, signedIn(user.ID, http.MethodPost, "/registrar", form))
	if rr.Code != http.StatusSeeOther {
		t.Fatalf("status = %d, want 303", rr.Code)
	}

	moments, err := s.MomentsOnDay(user.ID, fixedDay)
	if err != nil {
		t.Fatalf("MomentsOnDay: %v", err)
	}
	if len(moments) != 1 {
		t.Fatalf("len(moments) = %d, want 1", len(moments))
	}
	if !moments[0].MomentOfDay.Valid() {
		t.Errorf("MomentOfDay = %q, want a valid checkpoint", moments[0].MomentOfDay)
	}
	if !moments[0].Mood.Valid() {
		t.Errorf("Mood = %q, want a valid mood", moments[0].Mood)
	}
	if moments[0].Serenity != models.SerenityDefault {
		t.Errorf("Serenity = %d, want %d", moments[0].Serenity, models.SerenityDefault)
	}
}

func TestRegisterPost_neverWritesToAnotherAccount(t *testing.T) {
	h, s, user, f := newDiaryHandler(t)
	other := models.User{Name: "Gui", Email: "gui@example.com", PasswordHash: "hash"}
	otherID, err := s.CreateUser(other)
	if err != nil {
		t.Fatalf("CreateUser: %v", err)
	}
	moment := seedMoment(t, s, f, otherID, fixedDay, models.MomentMorning)
	if err := s.AttachTags(moment.ID, otherID, []string{"paz"}); err != nil {
		t.Fatalf("AttachTags: %v", err)
	}

	form := url.Values{"body": {"Meu momento."}, "tags": {"paz"}}
	rr := httptest.NewRecorder()
	h.RegisterPost(rr, signedIn(user.ID, http.MethodPost, "/registrar", form))
	if rr.Code != http.StatusSeeOther {
		t.Fatalf("status = %d, want 303", rr.Code)
	}

	tags, err := s.TagsForUser(user.ID)
	if err != nil {
		t.Fatalf("TagsForUser: %v", err)
	}
	if len(tags) != 1 {
		t.Fatalf("len(tags) = %d, want the writer's own tag", len(tags))
	}
	if tags[0].UserID != user.ID {
		t.Errorf("tag owner = %d, want %d", tags[0].UserID, user.ID)
	}
}

func TestMemories_listsTimelineAndMonthGrid(t *testing.T) {
	h, s, user, f := newDiaryHandler(t)
	seedMoment(t, s, f, user.ID, fixedDay, models.MomentMorning)
	seedMoment(t, s, f, user.ID, fixedDay.AddDate(0, 0, -2), models.MomentEvening)

	rr := httptest.NewRecorder()
	h.Memories(rr, signedIn(user.ID, http.MethodGet, "/memorias", nil))

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rr.Code)
	}
	body := rr.Body.String()
	if got := strings.Count(body, `data-testid="memories-entry"`); got != 2 {
		t.Errorf("entries = %d, want 2", got)
	}
	if !strings.Contains(body, `data-testid="memories-count"`) {
		t.Error("missing memories count")
	}
	if !strings.Contains(body, `data-testid="memories-day-marked"`) {
		t.Error("month grid should mark recorded days")
	}
}

func TestMemories_filtersByTag(t *testing.T) {
	h, s, user, f := newDiaryHandler(t)
	kept := seedMoment(t, s, f, user.ID, fixedDay, models.MomentMorning)
	silent := seedMoment(t, s, f, user.ID, fixedDay.AddDate(0, 0, -1), models.MomentMorning)
	if err := s.AttachTags(kept.ID, user.ID, []string{"paz"}); err != nil {
		t.Fatalf("AttachTags: %v", err)
	}
	_ = silent

	rr := httptest.NewRecorder()
	h.Memories(rr, signedIn(user.ID, http.MethodGet, "/memorias?tag=paz", nil))

	if got := strings.Count(rr.Body.String(), `data-testid="memories-entry"`); got != 1 {
		t.Errorf("entries = %d, want 1 after tag filter", got)
	}
}

func TestMemories_rendersEmptyState(t *testing.T) {
	h, _, user, _ := newDiaryHandler(t)

	rr := httptest.NewRecorder()
	h.Memories(rr, signedIn(user.ID, http.MethodGet, "/memorias", nil))

	if !strings.Contains(rr.Body.String(), `data-testid="memories-empty"`) {
		t.Errorf("missing empty state, got: %s", rr.Body.String())
	}
}

func TestInsights_summarisesTheJournal(t *testing.T) {
	h, s, user, f := newDiaryHandler(t)
	days := testdata.ConsecutiveDays(fixedDay, 7)
	for _, day := range days {
		moment := seedMoment(t, s, f, user.ID, day, models.MomentEvening)
		if err := s.AttachTags(moment.ID, user.ID, []string{"paz"}); err != nil {
			t.Fatalf("AttachTags: %v", err)
		}
	}

	rr := httptest.NewRecorder()
	h.Insights(rr, signedIn(user.ID, http.MethodGet, "/insights", nil))

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rr.Code)
	}
	body := rr.Body.String()
	for _, testid := range []string{"insights-stats", "insights-streak", "insights-theme", "insights-milestone"} {
		if !strings.Contains(body, `data-testid="`+testid+`"`) {
			t.Errorf("missing %s", testid)
		}
	}
	if !strings.Contains(body, ">7<") {
		t.Errorf("expected the 7 day streak in the stats, body: %s", body)
	}
	if !strings.Contains(body, "paz") {
		t.Error("expected the heart theme in the themes list")
	}
}

func TestInsights_rendersSerenityCurve(t *testing.T) {
	h, s, user, f := newDiaryHandler(t)
	for _, day := range testdata.ConsecutiveDays(fixedDay, 5) {
		seedMoment(t, s, f, user.ID, day, models.MomentMorning)
	}

	rr := httptest.NewRecorder()
	h.Insights(rr, signedIn(user.ID, http.MethodGet, "/insights", nil))

	if !strings.Contains(rr.Body.String(), `data-testid="insights-serenity"`) {
		t.Error("missing serenity curve")
	}
}

func TestInsights_handlesAnEmptyJournal(t *testing.T) {
	h, _, user, _ := newDiaryHandler(t)

	rr := httptest.NewRecorder()
	h.Insights(rr, signedIn(user.ID, http.MethodGet, "/insights", nil))

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rr.Code)
	}
	body := rr.Body.String()
	if !strings.Contains(body, `data-testid="insights-stats"`) {
		t.Error("empty journal should still render the stats band")
	}
	if !strings.Contains(body, `data-testid="insights-empty"`) {
		t.Error("expected the empty hints for curve and themes")
	}
}

func TestToday_ignoresOtherAccountsMoments(t *testing.T) {
	h, s, user, f := newDiaryHandler(t)
	otherID, err := s.CreateUser(models.User{Name: "Gui", Email: "gui@example.com", PasswordHash: "hash"})
	if err != nil {
		t.Fatalf("CreateUser: %v", err)
	}
	seedMoment(t, s, f, otherID, fixedDay, models.MomentMorning)

	rr := httptest.NewRecorder()
	h.Today(rr, signedIn(user.ID, http.MethodGet, "/hoje", nil))

	if got := strings.Count(rr.Body.String(), `data-filled="true"`); got != 0 {
		t.Errorf("filled = %d, want 0 — another account's entry leaked in", got)
	}
}
