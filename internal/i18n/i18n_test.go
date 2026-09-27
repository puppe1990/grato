package i18n

import "testing"

func TestDefaultCatalog_english(t *testing.T) {
	c := DefaultCatalog()
	if got := c.T("auth.welcome"); got != "So good to have you here!" {
		t.Errorf("T(auth.welcome) = %q", got)
	}
	if got := c.T("nav.today"); got != "Today" {
		t.Errorf("T(nav.today) = %q", got)
	}
}

func TestNewCatalog_portuguese(t *testing.T) {
	c := NewCatalog("pt-BR")
	if got := c.T("auth.welcome"); got != "Que bom ter você aqui!" {
		t.Errorf("T(auth.welcome) = %q", got)
	}
	if got := c.T("nav.today"); got != "Hoje" {
		t.Errorf("T(nav.today) = %q", got)
	}
	if c.HTMLLang() != "pt-BR" {
		t.Errorf("HTMLLang() = %q, want pt-BR", c.HTMLLang())
	}
}

// Every key must exist in both catalogs: a missing translation renders the raw
// key into the page, which is how "nav.register" once leaked into the layout.
func TestCatalogs_shareTheSameKeys(t *testing.T) {
	for key := range ptMessages {
		if _, ok := enMessages[key]; !ok {
			t.Errorf("en catalog is missing %q", key)
		}
	}
	for key := range enMessages {
		if _, ok := ptMessages[key]; !ok {
			t.Errorf("pt catalog is missing %q", key)
		}
	}
}
