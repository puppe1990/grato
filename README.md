# Grato — Diário da Gratidão

A calm, private gratitude journal. Three checkpoints a day, a timeline of
memories, and the quiet maths of how a practice changes a person over time.

Built with [amarra-cais](https://github.com/puppe1990/amarra-cais) — Go,
Amarra Views + Drive, Tailwind and SQLite. Server-rendered HTML, no SPA.

<p align="center">
  <img src="web/static/og.png" alt="Grato — Diário da Gratidão" width="720" />
</p>

## The four screens

| Screen        | Route        | What it does                                                                 |
| ------------- | ------------ | ---------------------------------------------------------------------------- |
| **Hoje**      | `/hoje`      | Greeting, the daily reflection, today's three checkpoints and the streak     |
| **Registrar** | `/registrar` | Write one moment: prompt, free reflection, feeling slider, resonance tags    |
| **Memórias**  | `/memorias`  | Month calendar, tag filters, the "há 1 ano" card and the reverse timeline    |
| **Insights**  | `/insights`  | Month recap, serenity curve, heart themes, milestones and the nightly ritual |

Sign in at `/login`, create an account at `/signup`. A signed-out visitor
lands straight on the door — the diary has no marketing page, the first screen
is the day itself.

## Design

The visual system is **Serene Hearth**, ported from the Stitch design export:
warm paper surfaces (`#FAF7F2`, `#F3EDE4`, `#E9DFD3`), terracotta
(`#D97746`) for intentional actions, sage (`#5A7865`) for completion and
rest, and muted gold for milestones. Depth comes from tonal paper layering
and ultra-diffused ambient warmth rather than hard drop shadows — there is no
pure black or pure white anywhere in the palette.

Type pairs **Literata** (reflections, headings) with **Plus Jakarta Sans**
(interface). Both are served by Google Fonts, so the CSP allowlist in
`.env` must keep `CSP_STYLE_SRC` / `CSP_FONT_SRC` set or the browser silently
falls back to system serif/sans.

The whole theme lives in `tailwind.config.js` plus the `hearth-*` component
classes in `input.css`.

## Quick start

```bash
export PATH="$HOME/go/bin:$PATH"
npm install --include=dev     # see note below
amarra-cais install
amarra-cais db migrate
amarra-cais db seed           # demo journal: 14 days, tags, ritual
amarra-cais dev               # http://localhost:8080
```

Demo login: `demo@example.com` / `password`.

> **Note** if `npm install` reports "audited 1 package" and installs nothing,
> your npm is configured with `omit=dev` and this project's dependencies are
> all devDependencies. Install with `npm install --include=dev`.

## Architecture

```
internal/gratitude/    pure reflection maths — streak, serenity curve, themes, milestones
internal/handlers/     one file per screen, plus diary.go for shared view data
internal/store/        SQLite: store.go (users) + gratitude.go (moments, tags, rituals)
internal/models/       Moment, Tag, Ritual, Stats, Intention
internal/testdata/     seeded faker builders shared by every test package
internal/i18n/         pt/en copy + locale-aware date formatting
web/templates/         layouts/ + pages/ + restyled kit components
web/static/            Tailwind CSS, amarra.js, PWA (icons, manifest, service worker)
design/                social card sources: og.svg + the phone capture it embeds
```

Two decisions worth knowing:

- **`moments.day` is denormalized** (`YYYY-MM-DD`). Every calendar query —
  today, month grid, streaks, year counts — compares strings instead of
  parsing DATETIME, which keeps them index-friendly and timezone-stable.
- **Everything user-owned is scoped by `user_id`** in the store, and the
  handler tests assert that one account can never read or write another's
  moments or tags.

## Tests

TDD throughout, with [gofakeit](https://github.com/brianvoe/gofakeit) driving
the fixtures from a fixed seed so failures reproduce exactly.

```bash
amarra-cais test        # go test ./...
make ci                 # test + lint + format-check
```

| Package              | What it pins down                                                                          |
| -------------------- | ------------------------------------------------------------------------------------------ |
| `internal/gratitude` | Streak edges (open today, stale run, duplicates), serenity averaging, milestone thresholds |
| `internal/store`     | Round-tripping moments + tags, per-user scoping, tag reuse, ritual defaults                |
| `internal/handlers`  | Every screen renders, validation returns 422, writes stay scoped to the writer             |
| `internal/i18n`      | Date vocabulary, greeting by hour, and that pt/en catalogs share the same keys             |

The i18n key-parity test exists because a missing key renders as the raw key
into the page — which is exactly how `nav.register` once leaked into the layout.

## PWA and social cards

Icons and the social card are generated from the design's flame mark. The
vector sources live next to the raster output for the icons, and in `design/`
for the card — `design/` holds the composition plus the phone screenshot it
embeds, so nothing that only feeds a build step ends up served publicly:

```bash
# PWA icons
rsvg-convert -w 512 -h 512 web/static/icons/icon.svg -o web/static/icons/icon-512.png
rsvg-convert -w 192 -h 192 web/static/icons/icon.svg -o web/static/icons/icon-192.png
rsvg-convert -w 180 -h 180 web/static/icons/icon.svg -o web/static/icons/icon.png
rsvg-convert -w 512 -h 512 web/static/icons/icon-maskable.svg -o web/static/icons/icon-512-maskable.png

# Open Graph card (1200x630) — embeds design/og-phone.png
rsvg-convert -w 1200 -h 630 design/og.svg -o web/static/og.png
```

`icon-maskable.svg` bleeds its plate to the edges and keeps the mark inside
the inner 80% safe zone, so Android launchers can crop it to a circle without
clipping the flame.

`design/og-phone.png` is a real capture of the Hoje screen (390x844, Chrome at
device width), so the card carries the app's actual typography instead of a
redrawn facsimile. Re-capture it at `/hoje` with a full day recorded:

```bash
playwright-cli resize 390 844 && playwright-cli goto http://localhost:8080/hoje
playwright-cli screenshot --filename=design/og-phone.png
```

The card's own text uses Georgia, an editorial serif standing in for Literata
because the rasteriser does not have Literata installed — the real typeface is
carried by the embedded screenshot. Everything else — `og:title`,
`og:description`, `og:url`, `og:locale`, `og:image`, and the full Twitter
card — is generated per page by `pkg/cais/meta.PreviewHTML` in
`amarraData`, absolute against `APP_URL` because crawlers neither resolve
relative paths nor run JavaScript. Public pages carry their own description
from the `meta.*.description` catalog keys; everything else falls back to the
brand one. `internal/handlers/preview_test.go` pins both the tag set and the
fact that `web/static/og.png` exists at the advertised path.

## Deploy

```bash
amarra-cais css
amarra-cais build --os linux --arch amd64 -o bin/server-linux
```

Ship the binary beside `web/static`. Set `ENV=production`, an `ADMIN_TOKEN`,
and keep the font CSP entries from `.env.example`.

## Continuous integration

`amarra-cais g ci` scaffolded the pipeline; it runs on every push:

- **Test** — `go test ./... -race -count=1`
- **Lint** — golangci-lint (errcheck, gocritic, govet, ineffassign, staticcheck, unused)
- **JS** — `npm ci`, Prettier `--check`, `npm test`

The same checks run locally via `pre-commit install` (trailing whitespace,
YAML, goimports, `go test`, golangci-lint, `npm test`) and `make ci`.

## Environment variables

| Variable        | Default         | Description                                              |
| --------------- | --------------- | -------------------------------------------------------- |
| `PORT`          | `:8080`         | Server port (auto-shifts if busy)                        |
| `DB_PATH`       | `./data/app.db` | SQLite file path                                         |
| `ENV`           | `development`   | Environment                                              |
| `LOCALE`        | `pt-BR`         | Default catalog locale                                   |
| `CSP_STYLE_SRC` | —               | Extra `style-src` hosts (`https://fonts.googleapis.com`) |
| `CSP_FONT_SRC`  | —               | Extra `font-src` hosts (`https://fonts.gstatic.com`)     |

Health check: `GET /health` → `{"status":"ok"}`.
Jobs dashboard: `GET /jobs` (localhost only).

## Testing on phone (LAN)

1. Run `amarra-cais dev` and note the **LAN** URL printed at boot.
2. Open that URL in mobile Safari/Chrome on the same Wi‑Fi.
3. After template changes, run `amarra-cais pwa --bump` and reinstall the PWA.
4. `amarra-cais doctor --mobile` catches flash markup, font CSP, and SW cache issues.

## Known issues

- Loading any page logs a 404 for `/true`. This is an upstream amarra-cais
  bug — `stream.mjs` marks `<html data-amarra-stream="true">` as its
  "already started" flag and then matches that same attribute when scanning
  for stream roots, so it opens `new EventSource("true")`. Tracked in
  [puppe1990/amarra-cais#204](https://github.com/puppe1990/amarra-cais/issues/204).
  The vendored `web/static/js/amarra.js` is left untouched so it keeps
  matching what the framework generates.

## License

MIT
