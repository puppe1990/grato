package handlers

import (
	"net/http"

	"github.com/puppe1990/amarra-cais/pkg/amarra/view"
	"github.com/puppe1990/amarra-cais/pkg/cais/i18n"
)

// Renderers holds one renderer per catalog tag, plus the one the app boots
// with.
//
// Templates bind their `t` function when they are parsed, so a single renderer
// is frozen in whatever locale it was loaded with — no request can change the
// copy it produces. Keeping one renderer per locale is what lets the language
// toggle mean something.
type Renderers struct {
	ByLocale map[string]*view.Renderer
	Default  *view.Renderer
}

// For picks the renderer for the request's negotiated locale, falling back to
// the boot locale when the request carries no catalog.
func (rs Renderers) For(r *http.Request) *view.Renderer {
	if catalog := i18n.CatalogFromRequest(r); catalog != nil {
		if renderer, ok := rs.ByLocale[catalog.Locale()]; ok {
			return renderer
		}
	}
	return rs.Default
}
