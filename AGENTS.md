# grato — AI Conventions

Primary reader is often an LLM agent. Prefer small greps, small modules, and headless tests.

## Rule #1: TDD is mandatory

Before writing production code:

1. Write the test (`*_test.go`)
2. Run focused: `go test ./... -v -run TestName`
3. Confirm it **fails** for the right reason
4. Write the **minimal** code to pass
5. Run `amarra-cais test`

## Clean code for agents

| Priority | Rule                                                                                                   |
| -------- | ------------------------------------------------------------------------------------------------------ |
| 1        | **Small units** — functions ~4–20 lines; files target 200–300 lines, hard cap ~500                     |
| 2        | **SRP** — one reason to change per file/package                                                        |
| 3        | **Greppable names** — unique domain nouns; avoid `data`, `handler`, `Manager`, `util` as primary names |
| 4        | **Comments = WHY** — security, SQLite, CSRF/cookie, Drive vs full HTML. No narrating WHAT              |
| 5        | **Inject deps** — handlers take `Store`, `*view.Renderer`, `cais.Config` via constructor               |
| 6        | **Early returns** — max ~2 nesting levels                                                              |
| 7        | **Errors with values** — `fmt.Errorf("...: %w", err)`                                                  |
| 8        | **Headless tests** — SQLite `:memory:`; no manual seed for unit tests                                  |

## Layout

| Path                        | Responsibility                 |
| --------------------------- | ------------------------------ |
| `cmd/server/`               | Entry point                    |
| `internal/app/`             | Bootstrap, `registerRoutes`    |
| `internal/handlers/`        | HTTP handlers (`view.Write`)   |
| `internal/store/`           | SQLite + migrations            |
| `internal/models/`          | Domain structs                 |
| `web/templates/layouts/`    | Amarra layout (`#amarra-main`) |
| `web/templates/pages/`      | HTML pages                     |
| `web/templates/components/` | App component overrides        |
| `web/static/`               | CSS, `amarra.js`, PWA          |

Patch markers (do not remove): `registerRoutes`, `Close() error`, `<!-- cais:nav -->`, `// cais:live-views`.

## App domain (Grato)

| Path                  | Responsibility                                                               |
| --------------------- | ---------------------------------------------------------------------------- |
| `internal/gratitude/` | Pure reflection maths: streak, serenity curve, themes, milestones            |
| `internal/handlers/`  | One file per screen: `today.go`, `register.go`, `memories.go`, `insights.go` |
| `internal/models/`    | `Moment`, `Tag`, `Ritual`, `Stats`, `Intention`                              |
| `internal/store/`     | `store.go` (users) + `gratitude.go` (moments, tags, rituals)                 |
| `internal/testdata/`  | Seeded faker builders shared by every test package                           |
| `internal/i18n/`      | pt/en copy plus locale-aware date formatting (`date.go`)                     |

Screens are Portuguese by default (`LOCALE=pt-BR`); the catalog holds every
string, so a missing key renders as the raw key — `TestCatalogs_shareTheSameKeys`
guards against that.

`moments.day` is a denormalized `YYYY-MM-DD` column: every calendar query
(today, month grid, streaks, year counts) compares strings instead of parsing
DATETIME, which keeps them index-friendly and timezone-stable.

Handlers take an injectable clock (`h.now`) so tests pin "today" to a fixed day.

## Amarra HTML

Handlers render HTML via `view.Write`:

```go
h.render(w, r, &user, "today", map[string]any{
  "Title":     h.catalog.T("nav.today"),
  "ActiveNav": "today",
}, 0)
// Validation — same page, status 422, `.Errors` on inputs
// Flash on redirect — cais cookie API only
flash.Set(w, "notice", h.catalog.T("register.saved"), cfg.CookieSecure())
http.Redirect(w, r, todayPath, http.StatusSeeOther)
```

Pages define a content block plus kit tags `<.form>` / `<.input>` / `<.button>` / `<.flash />` / `<.locale-toggle />`.
Kit attributes interpolate: `<.stat label="Potência" value="{{ .Power }} kWp" />` renders `5 kWp`; a control action in an attribute value fails at boot.
Templates load once at boot (`view.Load`): `layouts/*.html`; `pages/*.html` plus `pages/*/*.html` — `pages/blog/post.html` is `view.Page{Name: "blog/post"}`; `partials/*.html` and `components/*.html` are **flat only** (a nested partial never loads), and an unknown `<.x>` fails at boot.
Designed 404: register `r.NotFound(handler)` in `internal/app/routes.go` — it also serves path params that fail to parse (`IntParam`, `StringParam`); render with `writeView(..., http.StatusNotFound)`.
Drive morphs `#amarra-main` by default — plain links/forms and `linkTo` need no attribute; opt out with `data-amarra-skip` (#31). Do not check `HX-Request`.
Fullbleed pages (own header/footer/sidebar, e.g. marketing landing): Drive only swaps `#amarra-main`, so page-owned chrome outside it goes stale on Drive navigation — keep that chrome **inside** `#amarra-main` (wrap the whole page; the layout emits no chrome for those pages via `ActiveNav`/a second layout, #66). Inline `<script>` inside the morph runs. Escape hatch: `data-amarra-skip` (#31).
Inline page scripts re-run on **every** Drive morph into that page (#84): never top-level `let`/`const` (second visit throws `Identifier has already been declared` and the page goes dead) — wrap in an IIFE and expose handlers via `window.x = x`. Never `addEventListener` on morphed nodes (the listener stays on the orphaned node after the next morph) — use event delegation on `document`, registered once behind a `window` flag, resolving targets at click time.
Password fields: `<.password name="password" />` kit (input + eye toggle wired to `amarra-hook="password"`; `fieldPassword` stays as the Go form-builder path).
Shipped hooks: `bulk` (select-all: `amarra-hook="bulk"` + `data-amarra-bulk-all`/`-row`/`-bar`), `clipboard`, `dialog` (native `<dialog>` via `amarra-hook="dialog"` + `data-amarra-dialog-open`/`-target`/`-close`; `<.modal>` renders the target), `dropdown` (row actions: `amarra-hook="dropdown"` + `data-amarra-dropdown-button`/`-menu`), `nav` (re-sync active link after Drive morph: `amarra-hook="nav"` + `data-amarra-nav-on`/`-off` on the container), `password`, `reveal` (client show/hide, no Drive round-trip), `theme` (`html.light` + `localStorage["amarra-theme"]`, per-element via `data-amarra-theme-key` / `-class` / `-color` / `-on-label` / `-off-label`).
Theme FOUC snippet belongs in the layout `<head>` before CSS:

```html
<script>
  try {
    if (localStorage.getItem("amarra-theme") === "light")
      document.documentElement.classList.add("light");
  } catch (e) {}
</script>
```

Parse bodies with `httpx.ParseFormOrJSON`.

## Auth, CSRF, flash

- Session middleware: `LoadSession` + `Flash` + `CSRF(cfg)`
- Protect routes: `middleware.RequireAuth("/login")` / `RequireAuthFunc`
- CSRF: double-submit cookie `cais_csrf` + form field or `X-CSRF-Token`
- Flash: **only** `flash.Set` + read via `flash.MessageFromRequest`
- Dev demo user (when seeded): `demo@example.com` / `password`

## New screen

1. Go test in `internal/handlers/` (faker builds the fixtures)
2. HTML page in `web/templates/pages/`
3. Handler + route in `internal/app/routes.go` (behind `middleware.RequireAuth`)

Generators still work for side resources:

```bash
amarra-cais g handler settings     # handler + test + web/templates/pages/settings.html + route
amarra-cais g page about           # HTML page only
amarra-cais g component input      # override a kit component with its shipped markup (--list shows them)
amarra-cais g migration add_notes
amarra-cais db migrate
```

## Commands

```bash
amarra-cais install          # npm + go mod tidy (+ Tailwind build)
amarra-cais dev              # air + tailwind watch
amarra-cais test             # go test ./...
make ci                     # test + lint + format-check
amarra-cais doctor [--mobile]
amarra-cais routes
amarra-cais db migrate | status | rollback | seed
amarra-cais jobs work | status
```

`GET /jobs` — localhost queue dashboard (heartbeats, retry/discard, prune, `?kind=`). Production: SSH tunnel.

## Do not

- Parse templates per request (`view.Load` once at boot)
- Use inline CSS (Tailwind classes)
- Mock the database (use SQLite `:memory:`)
- Grow files past ~500 lines without splitting
- Ship features without a headless test
