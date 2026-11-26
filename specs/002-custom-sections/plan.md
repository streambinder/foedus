# Implementation Plan: Custom Homepage Sections

**Branch**: `feat/custom-sections` | **Date**: 2026-11-26 | **Spec**: [spec.md](spec.md)

**Input**: Feature specification from `specs/002-custom-sections/spec.md`

## Summary

Add admin-defined custom sections to the homepage. A custom section
holds a desktop/mobile background image pair (or no background), a
header, a subtitle, and a body text per language, and a position.
Positions count all homepage sections, with the landing section at
position 1. Custom sections merge into the rendered order with the
built-in sections, and the up/down arrows navigate through them.
A custom section never replaces a built-in section. The change adds
one table, one dashboard accordion, one template component, and one
order computation that replaces the six hardcoded arrow helpers.
No live database migration is needed: the new table is canonical
schema, created by `CREATE TABLE IF NOT EXISTS` on start.

## Technical Context

**Language/Version**: Go 1.26 (module `go 1.26.0`), toolchain 1.27.1

**Primary Dependencies**: Fiber v2.52.16, templ v0.3.1118 (library),
templ generator v0.3.1100, SQLite driver v1.62.0 (modernc.org)

**Storage**: One SQLite file per instance, STRICT tables, WAL mode.
The new `custom_sections` table follows the `places` and
`hero_backgrounds` pattern: its own table, a `sort_order` that
mirrors the dashboard form order, and a wipe-and-reinsert save
inside the `SaveSettings` transaction.

**Testing**: Unit tests sit beside the code they cover (`*_test.go`).
`templ generate` runs before the tests. CI runs
`go test -race -p 1 -cover ./...` and fails any package below 90%
line coverage. The templates package sits at 90.1% and the handlers
package at 90.8% today, so the new helpers and the new handler code
need tests that cover them, not tests that happen to pass through
them.

**Target Platform**: linux/arm64 container, deployed on lobster as
two instances (`foedus-davideagnese`, `foedus-lorenzomargherita`),
each with its own SQLite file at
`/var/lib/foedus/<instance>/data.db`.

**Project Type**: Server-rendered web application, single Go binary.

**Constraints**: Code-side migrations create only the canonical
schema. The new table goes into `migrate()` as a `CREATE TABLE IF
NOT EXISTS` block. The live databases gain the table on the next
start; no manual `ALTER` is needed, and no manual rollout is
proposed. The application must open the existing live SQLite files
without data loss, which a new empty table satisfies.

**Scale/Scope**: One table, one model, two database helpers, one
dashboard accordion with add/remove JavaScript, one template
component, one order computation, one CSS block. No new endpoint,
no new route, no new dependency.

## Constitution Check

- Principle I (one repository, one deployable unit): the change adds
  no root entry and no new deployable. It stays inside the one
  binary.
- Principle II (CI is the source of truth): the merge waits for a
  green pipeline on the exact head pushed. Local checks mirror the
  pipeline invocation.
- Principle III (one linting system): super-linter runs on the
  branch. New Go code passes `gofumpt` before every push. New
  Markdown passes prettier. `templ generate` runs before the tests.
- Principle IV (tests and the coverage gate): new tests live next to
  the code they cover. The per-package 90% gate binds every touched
  package. Non-test code never changes to satisfy a test. The
  ordering computation is a pure function, so its tests assert the
  merge directly instead of through a render.
- Principle V (minimal owned surface): no new dependency. The
  background rendering reuses the hero `.vp-hero-background-layer`
  and wash classes. The dashboard card reuses the hero-background
  managed-image field and the add/remove/reindex script pattern.
  The save reuses `insertAll`, `Replace*`, and `resolveImageMediaIDFromSet`.
- Principle VI (commits and pull requests): one commit per plan
  phase. Each file appears in exactly one commit. The owner merges.
- Principle VII (containers and releases): no Dockerfile change.
  Publish never runs on a pull request.
- Principle VIII (writing and consistency): the change follows the
  conventions of each file it touches. It does not reformat parts it
  does not touch.
- Database Migrations (§26): the code-side change is the canonical
  schema only, one new `CREATE TABLE IF NOT EXISTS` block. No
  `ALTER`, no rename, no value rewrite.
- Production Care: backup both live databases before the deploy.
  Verify both instances answer after the deploy. Never write on
  production data to test.
- Hypothesis First: the investigation of 2026-11-26 verified the
  gap on the real code at `origin/master` 96e32da: no
  custom-section mechanism exists, the sections are hardcoded in
  render order, and the arrow chain is hardcoded. The owner
  confirmed the background form, the per-language texts, the
  full-bleed pair, and the never-replace rule on 2026-11-26, and
  the specification records those answers.

No violation. The complexity tracking section stays empty.

## Project Structure

### Documentation (this feature)

```text
specs/002-custom-sections/
├── spec.md              # Ratified specification
├── plan.md              # This file
└── tasks.md             # Ordered task list
```

### Source Code (repository root)

```text
internal/models/models.go             # CustomSection struct
internal/database/database.go         # Canonical CREATE TABLE custom_sections
internal/database/custom_sections.go  # Get/ReplaceCustomSections (new file)
internal/handlers/home.go             # loadHomeContent loads custom sections
internal/handlers/dashboard.go        # loadSettingsContent, parse, SaveSettings
templates/content.go                  # HomeContent/SettingsContent gain CustomSections
templates/home.templ                  # Order plan, CustomSection component, arrows
templates/dashboard.templ             # Custom Sections accordion
static/css/styles.css                 # .vp-custom-section rules
```

**Structure Decision**: The change follows the existing collection
pattern end-to-end: one table per relation, one struct, one
`Get*`/`Replace*` pair, one `parse*` function in the dashboard
handler, one dashboard accordion, one wipe-and-reinsert write
inside the existing transaction. The only new shape is the order
plan: the six hardcoded arrow helpers become one computation over
the rendered order, because a hardcoded chain cannot name sections
that do not exist yet.

## Approach

### Ordering model

The rendered built-in sections, in canonical order, are landing
(always), venues (always), soundtrack (only with a playlist),
places (only with story places), honeymoon (only with honeymoon
places), and registry (only when `bankConfigured`). Their rendered
places are 1 through N in that order.

A custom section with position P renders before the built-in
section at rendered place P. Position 1 renders before landing.
A position past N renders after the last built-in section. Two
custom sections at the same position render in creation order,
which is the dashboard form order (`sort_order`, then ID).
Gaps in the numbering collapse: positions are merge keys, not
slots, so no empty slot ever renders.

The merge is one pure computation, `planHomeSections`, in
`templates/home.templ`, next to the helpers it replaces. It takes
the playlist, the places, the honeymoon places, `bankConfigured`,
the custom sections, and the language, and returns a plan with the
ordered anchor list (`#landing`, `#venues`, `#soundtrack`,
`#places`, `#honeymoon`, `#registry`, `#custom-<id>`) and, for
each built-in anchor, the custom sections to render before it
(the past-the-end customs ride under the empty key). Two small
helpers read the plan: `prevAnchor` and `nextAnchor` return the
neighbour of an anchor in the order, or the empty string at the
ends. The language filters the customs inside the plan: a section
renders in a language only when that language has a header and a
body or a background image (assumption A1).

The custom section ID in the anchor comes from the database
autoincrement ID. The wipe-and-reinsert save gives the rows new
IDs on every save, as it does for places and hero backgrounds.
Anchors are computed fresh on every render, so no stale anchor
persists anywhere.

### Phase 0: Foundation — model, table, and database helpers

Add `CustomSection` to `internal/models/models.go`:

```go
type CustomSection struct {
  ID             int
  Position       int
  HeaderIt       string
  HeaderEn       string
  SubtitleIt     string
  SubtitleEn     string
  BodyIt         string
  BodyEn         string
  DesktopMediaID int
  MobileMediaID  int
  SortOrder      int
  CreatedAt      Timestamp
}
```

Add the `custom_sections` table to `migrate()` in
`internal/database/database.go`, STRICT, after the
`hero_backgrounds` block:

```sql
CREATE TABLE IF NOT EXISTS custom_sections (
  id               INTEGER PRIMARY KEY AUTOINCREMENT,
  position         INTEGER NOT NULL DEFAULT 0,
  header_it        TEXT NOT NULL DEFAULT '',
  header_en        TEXT NOT NULL DEFAULT '',
  subtitle_it      TEXT NOT NULL DEFAULT '',
  subtitle_en      TEXT NOT NULL DEFAULT '',
  body_it          TEXT NOT NULL DEFAULT '',
  body_en          TEXT NOT NULL DEFAULT '',
  desktop_media_id INTEGER REFERENCES media(id),
  mobile_media_id  INTEGER REFERENCES media(id),
  sort_order       INTEGER NOT NULL DEFAULT 0,
  created_at       INTEGER NOT NULL DEFAULT (unixepoch())
) STRICT
```

Add `internal/database/custom_sections.go`, following
`hero_backgrounds.go`: `scanCustomSection` (nullable media IDs
through `idOrZero`), `GetCustomSections` (`ORDER BY sort_order
ASC, id ASC`), and `ReplaceCustomSections` (delete all, then
`insertAll` with the index as `sort_order`).

Add `CustomSections []models.CustomSection` to `HomeContent` and
to `SettingsContent` in `templates/content.go`.

Tests: database tests cover the round trip (insert through
`ReplaceCustomSections`, read back through `GetCustomSections`
in form order) and the empty table.

### Phase 1: Ordering and rendering

Load the sections in `loadHomeContent` in
`internal/handlers/home.go`, next to the hero backgrounds load.
No other handler change: `Home` already passes `content`
through to the template.

In `templates/home.templ`, delete the six hardcoded arrow
helpers (`prevSectionBeforeRegistry`, `nextSectionAfterVenues`,
`nextSectionAfterSoundtrack`, `nextSectionAfterPlaces`,
`prevSectionBeforeHoneymoon`, `nextSectionAfterHoneymoon`) and
replace them with `planHomeSections`, `prevAnchor`,
`nextAnchor`, plus two small helpers:

- `customSectionText(section, lang)` returns the header,
  subtitle, and body for the language, and whether the section
  renders in that language under A1 (a header and a body or a
  background image in that language).
- `customSectionBackground(section)` returns the desktop and
  mobile media IDs with the A3 fallback applied each way, as
  `pickHeroBackground` already does per side for the hero.

Add a `CustomSection` templ component. It renders
`<section id="custom-<id>" class="vp-section vp-custom-section">`,
with the full-bleed background when media IDs survive the
fallback: a `<picture class="vp-hero-background-layer">` with
the desktop `<source media="(min-width: 768px)">` and the
fallback `<img>`, plus the `.vp-hero-background-wash`, exactly
as the landing section does. The texts render inside
`.vp-section-inner` with the shared `.section-heading`,
`.section-title`, and `.section-subtitle` classes, and the body
in a `<p class="custom-section-body">`. The up/down arrows come
from the plan, so a custom section navigates like a built-in
one.

In `Home`, compute the plan once at the top and replace every
hardcoded arrow with the plan neighbours: the landing down
arrow becomes the next anchor after landing, the venues up
arrow the previous anchor before venues, and so on for
soundtrack, places, honeymoon, and registry. Before each
built-in section block, render the customs the plan assigns
before it (`plan.before["#venues"]` and the like); after the
registry block, render the past-the-end customs
(`plan.before[""]`). A section with a background renders its
texts in ivory over the wash, as the hero does; without a
background it renders on the page background with the default
text colour.

Add the CSS in `static/css/styles.css`: `.vp-custom-section`
(centre the texts, reuse the hero background layer), the
with-background ivory text rule, and `.custom-section-body`
(Inter, the body size, readable measure). No other section's
rules change.

Tests: `helpers_test.go` covers the plan directly — the base
order with no customs, a custom at position 2, a custom past
the end, a tie at one position, a custom in Italian only (hidden
in English, shown in Italian), and the neighbours at the ends.
`home_test.go` renders `Home` with custom sections and asserts
the section ID, the full-bleed picture, the per-language texts,
the position 2 order, and the honeymoon coexistence (custom
honeymoon section with no honeymoon places renders alone; with
honeymoon places both render). The old `TestSectionNavigationHelpers`
is replaced by the plan tests. The render tests reuse
`homeFixtures()` and add a `CustomSections` field to its
content.

### Phase 2: Dashboard and save

Load the sections in `loadSettingsContent` in
`internal/handlers/dashboard.go`, next to the hero backgrounds
load, and add their media IDs to `collectExistingMediaIDs` so
the orphan cleanup keeps them.

Add `parseCustomSections` in the same file, following
`parseHeroBackgrounds`: the form declares
`custom_section_count`, and each card posts
`custom_section_position_<i>`,
`custom_section_header_it_<i>`,
`custom_section_header_en_<i>`,
`custom_section_subtitle_it_<i>`,
`custom_section_subtitle_en_<i>`,
`custom_section_body_it_<i>`,
`custom_section_body_en_<i>`,
`custom_section_desktop_<i>` with
`custom_section_desktop_media_id_<i>`, and the mobile pair,
resolved through `resolveImageMediaIDFromSet`. A card with every
text field empty, no position, and no media on either side is
skipped. Validation (assumption A2) rejects the whole save with
a 400: a non-empty card with no header in either language, a
card with neither a body nor a background image in either
language, or a card with a missing, non-numeric, or below-1
position. A declared count above zero that parses to no sections
at all rejects the save, as the hero-background guard does:
a form/server contract drift must not silently wipe the table.

Wire the parse into `SaveSettings` next to the hero-background
parse, add `ReplaceCustomSections` to the writes list, and add
the count to the success log line.

Add the Custom Sections accordion to `templates/dashboard.templ`,
after the Homepage Landing Backgrounds accordion, following its
card markup: a card per section with a number input for the
position, text inputs for the header and subtitle per language,
textareas for the body per language, and two managed-image fields
for the desktop/mobile pair (same `data-format`, `data-quality`,
and size limits as the hero pair). Add the `<template>` card,
the add/remove buttons, and the reindex script, mirroring
`addHomepageHeroBackground` / `removeHomepageHeroBackground` /
`reindexHeroBackgrounds` with the `custom_section_` field names
and the `custom-section-count` hidden input.

Tests: handler tests post the settings form through
`newTestApp()` and assert a saved section round-trips through
`GetCustomSections`, a section with a header and an image but
no body saves, a section with a header and a body but no image
saves, and the rejections (no header, neither body nor image,
bad position) return 400 and leave the previous sections
untouched. The templates package needs no new dashboard test:
the accordion renders through the existing settings render.

### Phase 3: Verification

Run `templ generate`, then `gofumpt`, then the full test suite
with race detection and coverage, then the per-package 90% gate
with the AWK check from the pipeline. Run the preflight checks:
fetch the remote, branch from the current remote default, read
the diff against it, scan for conflict markers and placeholder
text, and count the commits. Open the spec pull request first,
then the implementation pull request with one commit per phase
above, wait for CI on each, and fix a red branch by amending
the failing commit.

Note on the concurrent fix: a separate styling fix for the
coordinate gift tile (branch `fix/coordinate-gift-tile-style`)
touches `templates/home.templ`, `static/css/styles.css`, and
`templates/home_test.go` at the same time as this cycle. If it
merges before this implementation branch, rebase this branch
onto the new master (explicit-lease force push) and re-verify
CI before reporting. The spec branch touches only the spec
directory and cannot conflict with it.

## Complexity Tracking

Empty. No constitution violation needs a justification.
