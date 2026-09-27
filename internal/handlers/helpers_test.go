package handlers

import (
	"errors"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/puppe1990/amarra-cais/pkg/amarra/view"
	"github.com/puppe1990/amarra-cais/pkg/cais/meta"

	appi18n "github.com/puppe1990/grato/internal/i18n"
	"github.com/puppe1990/grato/internal/store"
)

func testSite() meta.Site {
	return meta.Site{AppName: "grato", AppURL: "https://cais.example.com"}
}

func projectRoot(t *testing.T) string {
	t.Helper()
	root, err := repositoryRoot()
	if err != nil {
		t.Fatal(err)
	}
	return root
}

func repositoryRoot() (string, error) {
	wd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for {
		if _, err := os.Stat(filepath.Join(wd, "go.mod")); err == nil {
			return wd, nil
		}
		parent := filepath.Dir(wd)
		if parent == wd {
			return "", errors.New("go.mod not found above the test working directory")
		}
		wd = parent
	}
}

// Renderers are read-only once loaded, so the test binary builds them once
// rather than re-parsing every template for every test.
var (
	testRenderersOnce sync.Once
	testRenderers     Renderers
	testRenderersErr  error
)

func setupTestViews(t *testing.T) Renderers {
	t.Helper()
	testRenderersOnce.Do(func() {
		testRenderers, testRenderersErr = loadRenderers()
	})
	if testRenderersErr != nil {
		t.Fatal(testRenderersErr)
	}
	return testRenderers
}

func loadRenderers() (Renderers, error) {
	root, err := repositoryRoot()
	if err != nil {
		return Renderers{}, err
	}
	fsys := os.DirFS(filepath.Join(root, "web", "templates"))

	byLocale := make(map[string]*view.Renderer)
	for tag, catalog := range appi18n.Catalogs() {
		renderer, err := view.Load(fsys, catalog)
		if err != nil {
			return Renderers{}, err
		}
		byLocale[tag] = renderer
	}
	return Renderers{ByLocale: byLocale, Default: byLocale[appi18n.DefaultCatalog().Locale()]}, nil
}

func setupTestStore(t *testing.T) store.Store {
	t.Helper()
	s, err := store.NewSQLiteStore(":memory:", "test")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = s.Close() })
	return s
}
