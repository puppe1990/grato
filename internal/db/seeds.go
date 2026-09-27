package db

import (
	"time"

	"github.com/brianvoe/gofakeit/v7"

	"github.com/puppe1990/grato/internal/models"
	"github.com/puppe1990/grato/internal/store"
)

// Seed is fixed so demo data is identical on every machine.
const Seed = uint64(20261024)

// RunSeeds fills the demo account with a warm, plausible journal: a lived-in
// streak, morning/afternoon/evening checkpoints and the resonance tags the
// design calls out. Safe to run repeatedly.
func RunSeeds(s store.Store) error {
	user, err := s.FindUserByEmail("demo@example.com")
	if err != nil {
		return err
	}
	existing, err := s.CountMoments(user.ID)
	if err != nil {
		return err
	}
	if existing > 0 {
		return nil
	}

	f := gofakeit.New(Seed)
	today := time.Now()
	entries := seedEntries()

	// Walk backwards from today so the streak is unbroken and the current
	// month already has a handful of recorded days.
	for offset := len(entries) - 1; offset >= 0; offset-- {
		entry := entries[offset]
		day := today.AddDate(0, 0, -(len(entries) - 1 - offset))
		id, err := s.InsertMoment(models.Moment{
			UserID:      user.ID,
			MomentOfDay: entry.ofDay,
			Mood:        entry.mood,
			Title:       entry.title,
			Body:        entry.body,
			Serenity:    entry.serenity,
			HappenedAt:  atHour(day, entry.ofDay, f.IntRange(0, 20)),
		})
		if err != nil {
			return err
		}
		if err := s.AttachTags(id, user.ID, entry.tags); err != nil {
			return err
		}
	}

	return s.SaveRitual(models.Ritual{
		UserID:   user.ID,
		Kind:     models.RitualNightReflection,
		Enabled:  true,
		RemindAt: models.DefaultRemindAt,
	})
}

type seedEntry struct {
	ofDay    models.MomentOfDay
	mood     models.Mood
	title    string
	body     string
	serenity int
	tags     []string
}

// seedEntries is written in Portuguese on purpose: it is the demo the design
// was drawn against, not filler generated from a word list.
func seedEntries() []seedEntry {
	return []seedEntry{
		{models.MomentMorning, models.MoodSerene, "O cheiro de café fresco",
			"O cheiro de café fresco pela manhã enquanto lia meu livro favorito junto à janela iluminada.", 4, []string{"pequenos detalhes"}},
		{models.MomentAfternoon, models.MoodGrateful, "Mensagem da minha mãe",
			"Mensagem carinhosa inesperada da minha mãe no meio da tarde com uma foto nossa sorrindo.", 4, []string{"família"}},
		{models.MomentMorning, models.MoodRadiant, "Xícara da manhã ao sol",
			"Sentei na cozinha sem pressa e deixei o sol da janela esquentar minhas mãos ao redor da xícara.", 5, []string{"pequenos detalhes"}},
		{models.MomentEvening, models.MoodReflective, "Caminhada de fim de tarde no parque",
			"Deixei as notificações desligadas e foquei unicamente no som das folhas secas estalando sob os passos. O céu estava dourado.", 4, []string{"natureza", "parque da cidade"}},
		{models.MomentAfternoon, models.MoodRadiant, "Café e risadas com a Luísa",
			"Amizades verdadeiras parecem ignorar a passagem dos meses. Passamos mais de duas horas falando sobre nossos medos e planos.", 5, []string{"amizades"}},
		{models.MomentMorning, models.MoodGrateful, "Finalizei aquele projeto desafiador",
			"Em outros tempos eu teria me afundado em ansiedade e noites sem dormir. Hoje, mantive minhas pausas para respiração.", 5, []string{"trabalho", "conquista"}},
		{models.MomentEvening, models.MoodSerene, "Silêncio antes de dormir",
			"Apaguei as luzes mais cedo e fiquei alguns minutos apenas respirando, sem tela nenhuma por perto.", 4, []string{"paz", "sono"}},
		{models.MomentAfternoon, models.MoodGrateful, "Almoço feito em casa",
			"Cozinhei sem pressa, com música baixa, e comi devagar percebendo cada tempero.", 3, []string{"pequenos detalhes"}},
		{models.MomentMorning, models.MoodRadiant, "Primeira luz na cortina",
			"Acordei antes do despertador e vi a luz entrando devagar pela cortina de linho cru.", 5, []string{"natureza"}},
		{models.MomentEvening, models.MoodTired, "Dia longo, coração cheio",
			"Cansei, mas foi um cansaço de quem fez o que precisava. Deitei grato pelo esforço.", 3, []string{"descanso"}},
		{models.MomentAfternoon, models.MoodSerene, "Conversa com o vizinho",
			"Parei dois minutos no portão e ouvi sobre o jardim dele. Dois minutos que mudaram a tarde.", 4, []string{"conexão"}},
		{models.MomentMorning, models.MoodGrateful, "Café na padaria da esquina",
			"O padeiro já sabe meu pedido. Ser reconhecido num lugar pequeno é um privilégio.", 4, []string{"pequenos detalhes", "conexão"}},
		{models.MomentEvening, models.MoodSerene, "Chuva batendo na janela",
			"Fiquei só ouvindo. Nenhuma urgência, nenhuma pressa.", 4, []string{"paz", "natureza"}},
		{models.MomentAfternoon, models.MoodReflective, "Terminei um livro difícil",
			"Levei três semanas, e valeu cada página. Anotei dois trechos para reler depois.", 4, []string{"leitura"}},
	}
}

func atHour(day time.Time, ofDay models.MomentOfDay, jitter int) time.Time {
	hours := map[models.MomentOfDay]int{
		models.MomentMorning:   8,
		models.MomentAfternoon: 15,
		models.MomentEvening:   21,
	}
	return time.Date(day.Year(), day.Month(), day.Day(), hours[ofDay], jitter, 0, 0, day.Location())
}
