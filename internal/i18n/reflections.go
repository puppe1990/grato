package i18n

// Reflection is the daily quote card on Hoje.
type Reflection struct {
	Text   string
	Author string
}

var reflections = map[string][]Reflection{
	"pt": {
		{Text: "A gratidão transforma o que temos em suficiente.", Author: "Melody Beattie"},
		{Text: "O silêncio também cura.", Author: ""},
		{Text: "Enquanto há respiração, há uma nova chance de recomeçar.", Author: ""},
		{Text: "Aquilo que você nota, cresce.", Author: ""},
		{Text: "Não é a pressa que aproxima; é a presença.", Author: ""},
		{Text: "A paz mora nos detalhes que a gente escolhe ver.", Author: ""},
		{Text: "A chama não pede pressa. Pede cuidado.", Author: ""},
		{Text: "Um pequeno detalhe guardado hoje vira abrigo amanhã.", Author: ""},
	},
	"en": {
		{Text: "Gratitude turns what we have into enough.", Author: "Melody Beattie"},
		{Text: "Silence also heals.", Author: ""},
		{Text: "While there is breath, there is another chance to begin.", Author: ""},
		{Text: "What you notice, grows.", Author: ""},
		{Text: "It is not haste that brings us closer; it is presence.", Author: ""},
		{Text: "Peace lives in the details we choose to see.", Author: ""},
		{Text: "The flame does not ask for speed. It asks for care.", Author: ""},
		{Text: "A small detail kept today becomes shelter tomorrow.", Author: ""},
	},
}

// ReflectionList returns the quotes available for a locale.
func ReflectionList(locale string) []Reflection {
	if list, ok := reflections[Base(locale)]; ok {
		return list
	}
	return reflections["en"]
}
