package models

// Ritual is a repeating daily practice, e.g. "Reflexão antes de dormir".
type Ritual struct {
	ID       int64
	UserID   int64
	Kind     string
	Enabled  bool
	RemindAt string
}

// RitualNightReflection is the wind-down ritual shown on Insights.
const RitualNightReflection = "reflexao_noturna"

// DefaultRemindAt is the suggested time for the night reflection ritual.
const DefaultRemindAt = "21:30"

// Intention is the "Qual é o seu principal objetivo?" answer from signup.
type Intention string

const (
	IntentionLessAnxious  Intention = "menos_ansiedade"
	IntentionBuildHabits  Intention = "habitos_positivos"
	IntentionSaveMemories Intention = "registrar_memorias"
	IntentionDefault      Intention = IntentionLessAnxious
	IntentionUnset        Intention = ""
)

// IntentionValues is the ordered chip set offered at signup.
var IntentionValues = []Intention{IntentionLessAnxious, IntentionBuildHabits, IntentionSaveMemories}

// Valid reports whether the value is one of the offered intentions.
func (i Intention) Valid() bool {
	for _, v := range IntentionValues {
		if i == v {
			return true
		}
	}
	return false
}
