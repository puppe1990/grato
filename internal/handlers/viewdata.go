package handlers

import (
	"html/template"
	"net/http"

	"github.com/puppe1990/amarra-cais/pkg/amarra/view"
	"github.com/puppe1990/amarra-cais/pkg/cais"
	"github.com/puppe1990/amarra-cais/pkg/cais/i18n"
	"github.com/puppe1990/amarra-cais/pkg/cais/meta"

	appi18n "github.com/puppe1990/grato/internal/i18n"
)

// Social card defaults. The image lives at the path the framework expects, so
// a future `amarra-cais pwa` pass keeps recognising it as a replaced brand
// asset instead of resetting it to the scaffold placeholder.
const (
	appName            = "Grato"
	defaultDescription = "Um diário de gratidão calmo e privado: três momentos por dia, memórias, ecos e serenidade."
	ogImagePath        = "/static/og.png"
)

// amarraData assembles the layout payload. fallback is the boot catalog the
// handler was built with: LocaleMiddleware normally supplies the negotiated
// one on the request, but a render without it must still reflect the app's
// configured locale rather than the framework default.
func amarraData(r *http.Request, site meta.Site, fallback *i18n.Catalog, extra map[string]any) map[string]any {
	s := meta.ForRequest(site, r)
	locale := fallback.Locale()
	if cat := i18n.CatalogFromRequest(r); cat != nil {
		locale = cat.Locale()
	}

	data := map[string]any{
		"Site":       s,
		"CSRFToken":  s.CSRFToken,
		"Flash":      s.Flash,
		"Locale":     locale,
		"LocaleBase": appi18n.Base(locale),
		"Head":       previewHead(r, s, locale, extra),
	}
	for k, v := range extra {
		data[k] = v
	}
	return data
}

// previewHead renders the Open Graph / Twitter block for the page. Every URL
// is absolute against APP_URL because crawlers neither resolve relative paths
// nor run the app's JavaScript.
func previewHead(r *http.Request, site meta.Site, locale string, extra map[string]any) template.HTML {
	preview := meta.Preview{
		Title:       previewField(extra, "Title"),
		Description: previewField(extra, "Description"),
		SiteName:    appName,
		SiteURL:     site.AppURL,
		Path:        r.URL.Path,
		Image:       ogImagePath,
		Locale:      previewLocale(locale),
		Type:        "website",
	}
	if preview.Title == "" {
		preview.Title = appName
	}
	if preview.Description == "" {
		preview.Description = defaultDescription
	}
	return template.HTML(meta.PreviewHTML(preview))
}

func previewField(extra map[string]any, key string) string {
	value, _ := extra[key].(string)
	return value
}

// previewLocale maps a locale to the language_TERRITORY form crawlers expect,
// so "pt" is advertised as "pt_BR" rather than a bare "pt".
func previewLocale(locale string) string {
	return i18n.NewCatalog(locale).OGLocale()
}

// pageDescription reads a page's social copy from the catalog. An absent key
// renders as the key itself, which would leak into the card, so anything
// unknown falls back to the brand description instead.
func pageDescription(catalog *i18n.Catalog, page string) string {
	key := "meta." + page + ".description"
	if message := catalog.T(key); message != key {
		return message
	}
	return ""
}

// writeView names the layout at the call site: an app with a second layout (a
// marketing "landing") must not render public pages inside the app chrome (#66).
// The renderer is chosen per request so the copy follows the negotiated locale.
func writeView(w http.ResponseWriter, r *http.Request, views Renderers, cfg cais.Config, layout, name string, data any, status int) {
	view.Write(w, r, views.For(r), view.Page{Layout: layout, Name: name, Data: data, Status: status}, cfg)
}
