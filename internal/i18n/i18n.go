package i18n

import (
	caisi18n "github.com/puppe1990/amarra-cais/pkg/cais/i18n"
)

var locales = map[string]map[string]string{
	"en": enMessages,
	"pt": ptMessages,
}

// NewCatalog returns a catalog for the given locale (en default, pt for pt-BR).
func NewCatalog(locale string) *caisi18n.Catalog {
	return caisi18n.NewCatalogFrom(locale, locales)
}

// DefaultCatalog returns the English catalog.
func DefaultCatalog() *caisi18n.Catalog {
	return NewCatalog(caisi18n.DefaultLocale)
}

// Catalogs returns every locale the app can render, keyed by the tag the
// middleware, the language toggle and the renderers all agree on.
func Catalogs() map[string]*caisi18n.Catalog {
	out := make(map[string]*caisi18n.Catalog, len(locales))
	for tag := range locales {
		out[tag] = NewCatalog(tag)
	}
	return out
}
