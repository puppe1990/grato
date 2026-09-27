package store

import (
	"testing"
	"time"

	"github.com/brianvoe/gofakeit/v7"

	"github.com/puppe1990/grato/internal/models"
	"github.com/puppe1990/grato/internal/testdata"
)

func seededStore(t *testing.T) (Store, *gofakeit.Faker) {
	t.Helper()
	return newTestStore(t), testdata.New()
}

func mustUser(t *testing.T, s Store, f *gofakeit.Faker) models.User {
	t.Helper()
	user := testdata.User(f)
	user.PasswordHash = "hash"
	id, err := s.CreateUser(user)
	if err != nil {
		t.Fatalf("CreateUser: %v", err)
	}
	user.ID = id
	return user
}

func mustMoment(t *testing.T, s Store, f *gofakeit.Faker, userID int64, day time.Time) models.Moment {
	t.Helper()
	moment := testdata.MomentAt(f, userID, day)
	id, err := s.InsertMoment(moment)
	if err != nil {
		t.Fatalf("InsertMoment: %v", err)
	}
	moment.ID = id
	return moment
}

func mustMomentOf(t *testing.T, s Store, f *gofakeit.Faker, userID int64, day time.Time, ofDay models.MomentOfDay) models.Moment {
	t.Helper()
	moment := testdata.MomentOf(f, userID, day, ofDay)
	id, err := s.InsertMoment(moment)
	if err != nil {
		t.Fatalf("InsertMoment: %v", err)
	}
	moment.ID = id
	return moment
}

func TestCreateUser_storesProfileFields(t *testing.T) {
	s, f := seededStore(t)
	user := testdata.User(f)
	user.Name = "Sofia"
	user.Intention = models.IntentionBuildHabits
	user.PasswordHash = "hash"

	id, err := s.CreateUser(user)
	if err != nil {
		t.Fatalf("CreateUser: %v", err)
	}

	got, err := s.FindUserByID(id)
	if err != nil {
		t.Fatalf("FindUserByID: %v", err)
	}
	if got.Name != "Sofia" {
		t.Errorf("Name = %q, want Sofia", got.Name)
	}
	if got.Intention != models.IntentionBuildHabits {
		t.Errorf("Intention = %q, want %q", got.Intention, models.IntentionBuildHabits)
	}
	if got.Email != user.Email {
		t.Errorf("Email = %q, want %q", got.Email, user.Email)
	}
}

func TestUpdateUserProfile_keepsEmail(t *testing.T) {
	s, f := seededStore(t)
	user := mustUser(t, s, f)

	if err := s.UpdateUserProfile(user.ID, "Melody", models.IntentionSaveMemories); err != nil {
		t.Fatalf("UpdateUserProfile: %v", err)
	}

	got, err := s.FindUserByID(user.ID)
	if err != nil {
		t.Fatalf("FindUserByID: %v", err)
	}
	if got.Name != "Melody" || got.Intention != models.IntentionSaveMemories {
		t.Errorf("profile = %q/%q, want Melody/%q", got.Name, got.Intention, models.IntentionSaveMemories)
	}
	if got.Email != user.Email {
		t.Errorf("Email changed to %q", got.Email)
	}
}

func TestInsertMoment_roundTripsWithTags(t *testing.T) {
	s, f := seededStore(t)
	user := mustUser(t, s, f)
	moment := testdata.MomentAt(f, user.ID, time.Now())

	id, err := s.InsertMoment(moment)
	if err != nil {
		t.Fatalf("InsertMoment: %v", err)
	}
	if id == 0 {
		t.Fatal("InsertMoment returned id 0")
	}

	if err := s.AttachTags(id, user.ID, []string{"#paz", "família"}); err != nil {
		t.Fatalf("AttachTags: %v", err)
	}

	got, err := s.FindMoment(id)
	if err != nil {
		t.Fatalf("FindMoment: %v", err)
	}
	if got.Body != moment.Body {
		t.Errorf("Body = %q, want %q", got.Body, moment.Body)
	}
	if got.Mood != moment.Mood {
		t.Errorf("Mood = %q, want %q", got.Mood, moment.Mood)
	}
	if got.Serenity != moment.Serenity {
		t.Errorf("Serenity = %d, want %d", got.Serenity, moment.Serenity)
	}
	if len(got.Tags) != 2 {
		t.Fatalf("len(Tags) = %d, want 2", len(got.Tags))
	}
}

func TestAttachTags_isIdempotentAndReusesUserTags(t *testing.T) {
	s, f := seededStore(t)
	user := mustUser(t, s, f)
	first := mustMoment(t, s, f, user.ID, time.Now())
	second := mustMoment(t, s, f, user.ID, time.Now())

	if err := s.AttachTags(first.ID, user.ID, []string{"paz", "paz"}); err != nil {
		t.Fatalf("AttachTags: %v", err)
	}
	if err := s.AttachTags(second.ID, user.ID, []string{"Paz"}); err != nil {
		t.Fatalf("AttachTags: %v", err)
	}

	tags, err := s.TagsForUser(user.ID)
	if err != nil {
		t.Fatalf("TagsForUser: %v", err)
	}
	if len(tags) != 1 {
		t.Fatalf("len(tags) = %d, want 1 (reused)", len(tags))
	}
	if tags[0].Name != "paz" {
		t.Errorf("Name = %q, want paz (first spelling wins)", tags[0].Name)
	}
}

func TestAttachTags_ignoresBlanks(t *testing.T) {
	s, f := seededStore(t)
	user := mustUser(t, s, f)
	moment := mustMoment(t, s, f, user.ID, time.Now())

	if err := s.AttachTags(moment.ID, user.ID, []string{"", "   ", "#"}); err != nil {
		t.Fatalf("AttachTags: %v", err)
	}

	tags, err := s.TagsForUser(user.ID)
	if err != nil {
		t.Fatalf("TagsForUser: %v", err)
	}
	if len(tags) != 0 {
		t.Errorf("len(tags) = %d, want 0", len(tags))
	}
}

func TestListMoments_newestFirstAndLimited(t *testing.T) {
	s, f := seededStore(t)
	user := mustUser(t, s, f)
	days := testdata.ConsecutiveDays(time.Now(), 5)
	testdata.MomentsOn(f, user.ID, days)
	for _, day := range days {
		mustMoment(t, s, f, user.ID, day.Add(9*time.Hour))
	}

	moments, err := s.ListMoments(user.ID, 3)
	if err != nil {
		t.Fatalf("ListMoments: %v", err)
	}
	if len(moments) != 3 {
		t.Fatalf("len(moments) = %d, want 3", len(moments))
	}
	for i := 1; i < len(moments); i++ {
		if moments[i].HappenedAt.After(moments[i-1].HappenedAt) {
			t.Fatalf("moments not newest first: %v then %v", moments[i-1].HappenedAt, moments[i].HappenedAt)
		}
	}
}

func TestListMoments_loadsTagsInOneShot(t *testing.T) {
	s, f := seededStore(t)
	user := mustUser(t, s, f)
	moment := mustMoment(t, s, f, user.ID, time.Now())
	if err := s.AttachTags(moment.ID, user.ID, []string{"natureza", "paz"}); err != nil {
		t.Fatalf("AttachTags: %v", err)
	}

	moments, err := s.ListMoments(user.ID, 10)
	if err != nil {
		t.Fatalf("ListMoments: %v", err)
	}
	if len(moments) != 1 {
		t.Fatalf("len(moments) = %d, want 1", len(moments))
	}
	if len(moments[0].Tags) != 2 {
		t.Errorf("len(Tags) = %d, want 2", len(moments[0].Tags))
	}
}

func TestMomentsOnDay_filtersOtherDays(t *testing.T) {
	s, f := seededStore(t)
	user := mustUser(t, s, f)
	today := testdata.TruncateDay(time.Now())
	mustMoment(t, s, f, user.ID, today.Add(8*time.Hour))
	mustMoment(t, s, f, user.ID, today.AddDate(0, 0, -3).Add(8*time.Hour))

	moments, err := s.MomentsOnDay(user.ID, today)
	if err != nil {
		t.Fatalf("MomentsOnDay: %v", err)
	}
	if len(moments) != 1 {
		t.Fatalf("len(moments) = %d, want 1", len(moments))
	}
}

func TestStore_scopesDataPerUser(t *testing.T) {
	s, f := seededStore(t)
	owner := mustUser(t, s, f)
	intruder := mustUser(t, s, f)
	moment := mustMoment(t, s, f, owner.ID, time.Now())
	if err := s.AttachTags(moment.ID, owner.ID, []string{"paz"}); err != nil {
		t.Fatalf("AttachTags: %v", err)
	}

	if _, err := s.FindMomentForUser(moment.ID, intruder.ID); err == nil {
		t.Error("FindMomentForUser returned another user's moment")
	}
	moments, err := s.ListMoments(intruder.ID, 10)
	if err != nil {
		t.Fatalf("ListMoments: %v", err)
	}
	if len(moments) != 0 {
		t.Errorf("len(moments) = %d, want 0", len(moments))
	}
	tags, err := s.TagsForUser(intruder.ID)
	if err != nil {
		t.Fatalf("TagsForUser: %v", err)
	}
	if len(tags) != 0 {
		t.Errorf("len(tags) = %d, want 0", len(tags))
	}
}

func TestMomentDays_returnsDistinctSortedDays(t *testing.T) {
	s, f := seededStore(t)
	user := mustUser(t, s, f)
	days := testdata.ConsecutiveDays(time.Now(), 4)
	for _, day := range days {
		mustMomentOf(t, s, f, user.ID, day, models.MomentMorning)
		mustMomentOf(t, s, f, user.ID, day, models.MomentEvening)
	}

	got, err := s.MomentDays(user.ID)
	if err != nil {
		t.Fatalf("MomentDays: %v", err)
	}
	if len(got) != 4 {
		t.Fatalf("len(got) = %d, want 4 distinct days", len(got))
	}
	for i := 1; i < len(got); i++ {
		if got[i].Before(got[i-1]) {
			t.Fatalf("days not ascending: %v then %v", got[i-1], got[i])
		}
	}
}

func TestCounters(t *testing.T) {
	s, f := seededStore(t)
	user := mustUser(t, s, f)
	now := time.Now()
	mustMomentOf(t, s, f, user.ID, now, models.MomentMorning)
	mustMomentOf(t, s, f, user.ID, now, models.MomentEvening)
	mustMomentOf(t, s, f, user.ID, now.AddDate(0, 0, -1), models.MomentEvening)

	total, err := s.CountMoments(user.ID)
	if err != nil {
		t.Fatalf("CountMoments: %v", err)
	}
	if total != 3 {
		t.Errorf("CountMoments = %d, want 3", total)
	}

	evening, err := s.CountMomentsAt(user.ID, models.MomentEvening)
	if err != nil {
		t.Fatalf("CountMomentsAt: %v", err)
	}
	if evening != 2 {
		t.Errorf("CountMomentsAt(evening) = %d, want 2", evening)
	}

	daysThisYear, err := s.CountMomentDaysInYear(user.ID, now.Year())
	if err != nil {
		t.Fatalf("CountMomentDaysInYear: %v", err)
	}
	if daysThisYear < 1 {
		t.Errorf("CountMomentDaysInYear = %d, want >= 1", daysThisYear)
	}
}

func TestTagStats_ordersByUsage(t *testing.T) {
	s, f := seededStore(t)
	user := mustUser(t, s, f)
	for i := 0; i < 3; i++ {
		moment := mustMoment(t, s, f, user.ID, time.Now())
		if err := s.AttachTags(moment.ID, user.ID, []string{"paz"}); err != nil {
			t.Fatalf("AttachTags: %v", err)
		}
	}
	once := mustMoment(t, s, f, user.ID, time.Now())
	if err := s.AttachTags(once.ID, user.ID, []string{"família"}); err != nil {
		t.Fatalf("AttachTags: %v", err)
	}

	stats, err := s.TagStats(user.ID, 10)
	if err != nil {
		t.Fatalf("TagStats: %v", err)
	}
	if len(stats) != 2 {
		t.Fatalf("len(stats) = %d, want 2", len(stats))
	}
	if stats[0].Tag.Name != "paz" || stats[0].Count != 3 {
		t.Errorf("stats[0] = %s/%d, want paz/3", stats[0].Tag.Name, stats[0].Count)
	}
}

func TestRitual_defaultsWhenMissingThenRoundTrips(t *testing.T) {
	s, f := seededStore(t)
	user := mustUser(t, s, f)

	got, err := s.FindRitual(user.ID, models.RitualNightReflection)
	if err != nil {
		t.Fatalf("FindRitual: %v", err)
	}
	if got.Enabled {
		t.Error("a missing ritual should default to disabled")
	}
	if got.RemindAt != models.DefaultRemindAt {
		t.Errorf("RemindAt = %q, want %q", got.RemindAt, models.DefaultRemindAt)
	}

	got.Enabled = true
	got.RemindAt = "22:15"
	if err := s.SaveRitual(got); err != nil {
		t.Fatalf("SaveRitual: %v", err)
	}

	again, err := s.FindRitual(user.ID, models.RitualNightReflection)
	if err != nil {
		t.Fatalf("FindRitual: %v", err)
	}
	if !again.Enabled || again.RemindAt != "22:15" {
		t.Errorf("ritual = %+v, want enabled at 22:15", again)
	}
}
