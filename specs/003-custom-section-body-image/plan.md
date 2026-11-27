# Implementation Plan: Body image for custom sections

**Branch**: `feat/custom-sections-body-image` | **Date**: 2026-11-27 | **Spec**: [spec.md](spec.md)

**Input**: Feature specification from `specs/003-custom-section-body-image/spec.md`
(amends specs/002-custom-sections/spec.md, ratified 2026-11-26)

## Summary

Custom sections (cycle 002) gain an optional single body image, with no
desktop/mobile distinction, rendered below the body text, centred, at
content width. The dashboard background fields are relabelled to say
explicitly that they are background images. A body image counts as body
content for validation, so a section with a header and a body image (and
no body text, no background) is valid. The change adds one nullable
column to the existing `custom_sections` table, one dashboard
managed-image field per card, one render branch, and one CSS rule.
The live databases need a manual `ALTER` before this ships (see
Migration below): the homepage selects the new column explicitly, so a
live database without it breaks the site.

## Technical Context

**Language/Version**: Go 1.26 (module `go 1.26.0`), toolchain 1.27.1

**Primary Dependencies**: Fiber v2.52.16, templ v0.3.1118 (library),
templ generator v0.3.1100, SQLite driver v1.62.0 (modernc.org)

**Storage**: One SQLite file per instance, STRICT tables, WAL mode.
The `custom_sections` table gains one nullable column,
`body_media_id INTEGER REFERENCES media(id)`, following the existing
`desktop_media_id` / `mobile_media_id` pattern. The save stays
wipe-and-reinsert through `ReplaceCustomSections` inside the
`SaveSettings` transaction.

**Testing**: Unit tests sit beside the code they cover (`*_test.go`).
`templ generate` runs before the tests. CI runs
`go test -race -p 1 -cover ./...` and fails any package below 90%
line coverage. The templates package sits at 90.04% and the handlers
package at 90.8% today, so the new helpers and the new handler code
need tests that cover them, not tests that happen to pass through
them.

**Target Platform**: linux/arm64 container, deployed on lobster as
two instances (`foedus-davideagnese`, `foedus-lorenzomargherita`),
each with its own SQLite file at
`/var/lib/foedus/<instance>/data.db`.

**Project Type**: Server-rendered web application, single Go binary.

**Constraints**: Code-side migrations create only the canonical
schema. The code-side change updates the `CREATE TABLE IF NOT
EXISTS` block for fresh databases. The live databases already have
the `custom_sections` table (cycle 002, deployed 2026-11-26), so the
column is added there by a manual `ALTER`, once, before the
implementation merges (see Migration). The application must open
the existing live SQLite files without data loss, which a new
nullable column satisfies.

**Scale/Scope**: One column, one model field, one database-helper
update, one render branch, one CSS rule, one dashboard field per
card, two label changes. No new endpoint, no new route, no new
dependency.

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
  package. Non-test code never changes to satisfy a test.
- Principle V (minimal owned surface): no new dependency. The body
  image reuses the `media` table, the managed-image field, and the
  `resolveImageMediaIDFromSet` resolver. The render reuses
  `publicImageURL` and the `.vp-section-inner` content width.
- Principle VI (commits and pull requests): one commit per plan
  phase. Each file appears in exactly one commit. The owner merges.
  Both pull requests are based on master, not stacked: the
  implementation branch does not need the spec files, so it does
  not sit on the spec branch (the stacked-base mistake of cycles
  001 and 002).
- Principle VII (containers and releases): no Dockerfile change.
  Publish never runs on a pull request.
- Principle VIII (writing and consistency): the change follows the
  conventions of each file it touches. It does not reformat parts
  it does not touch.
- Database Migrations (§26): the code-side change is the canonical schema only, one column added to the `CREATE TABLE IF NOT EXISTS` block. No `ALTER`, no rename, no value rewrite in code. The live rollout is manual, once, proposed below.
- Production Care: backup both live databases, run the manual
  `ALTER` on both, verify the column, then merge and deploy.
  Verify both instances answer after the deploy. Never write on
  production data to test.
- Hypothesis First: the gap was verified on the real code at
  `origin/master` 68eb45d: `custom_sections` has only the
  background pair, the dashboard offers no body image, and the
  background labels say only "Desktop / horizontal" and
  "Mobile / vertical". The owner confirmed the single body image,
  the explicit background labels, and the below-text placement on
  2026-11-27, and the specification records those answers.

No violation. The complexity tracking section stays empty.

## Project Structure

### Documentation (this feature)

```text
specs/003-custom-section-body-image/
├── spec.md              # Ratified specification
├── plan.md              # This file
└── tasks.md             # Ordered task list
```

### Source Code (repository root)

```text
internal/models/models.go             # CustomSection gains BodyMediaID
internal/database/database.go         # Canonical CREATE TABLE gains body_media_id
internal/database/custom_sections.go  # scan/Get/Replace gain body_media_id
internal/handlers/dashboard.go        # collectExistingMediaIDs, parseCustomSections
templates/home.templ                  # customSectionText, CustomSection component
templates/dashboard.templ             # Background labels, body image field
static/css/styles.css                 # .custom-section-body-image rule
```

**Structure Decision**: The change follows the existing image
pattern end-to-end: one nullable media-id column per image, the
same `resolveImageMediaIDFromSet` resolver as the background pair
in the same parse function, the same post-save orphan sweep, and
the same managed-image field markup in the dashboard. The body
image is a single field, not a pair, because the spec fixes one
image for both viewports.

## Approach

### Migration

The code-side `CREATE TABLE IF NOT EXISTS custom_sections` gains
`body_media_id INTEGER REFERENCES media(id),` after
`mobile_media_id`. That covers fresh databases only. The live
databases already have the table, so before the implementation
merges, run once per live database (paths verified 2026-11-26,
directory without the `foedus-` prefix):

```sh
sqlite3 /var/lib/foedus/davideagnese/data.db ".backup /var/lib/foedus/davideagnese/data.db.bak.20261127"
sqlite3 /var/lib/foedus/davideagnese/data.db "ALTER TABLE custom_sections ADD COLUMN body_media_id INTEGER REFERENCES media(id);"
sqlite3 /var/lib/foedus/lorenzomargherita/data.db ".backup /var/lib/foedus/lorenzomargherita/data.db.bak.20261127"
sqlite3 /var/lib/foedus/lorenzomargherita/data.db "ALTER TABLE custom_sections ADD COLUMN body_media_id INTEGER REFERENCES media(id);"
```

Verify on both:

```sh
sqlite3 /var/lib/foedus/davideagnese/data.db "PRAGMA table_info(custom_sections);"
sqlite3 /var/lib/foedus/lorenzomargherita/data.db "PRAGMA table_info(custom_sections);"
```

Both must list `body_media_id`. The column is nullable with no
default, so no data changes and no existing row is affected. The
implementation pull request must not merge or deploy before this
runs: `GetCustomSections` selects the column explicitly, so a live
database without it fails the homepage and the dashboard loads.

### Phase 0: Foundation — model, table, and database helpers

Add `BodyMediaID int` to `CustomSection` in
`internal/models/models.go`, after `MobileMediaID`, and update the
struct comment: a single body image (or none), stored as a media
reference like the background pair.

Add the column to the `custom_sections` block in `migrate()` in
`internal/database/database.go`:

```sql
body_media_id INTEGER REFERENCES media(id),
```

Update `internal/database/custom_sections.go`: `scanCustomSection`
reads a third nullable media ID through `idOrZero`,
`GetCustomSections` selects `body_media_id`, and
`ReplaceCustomSections` inserts it (column list and value list gain
`body_media_id` / `nullableID(section.BodyMediaID)`).

Tests: extend the round-trip test in
`internal/database/collections_test.go` so a section with a body
media ID round-trips through `ReplaceCustomSections` and
`GetCustomSections`, and a section without one reads back as zero.

### Phase 1: Rendering

In `templates/home.templ`, `customSectionText` gains the body
image as body content (assumption A1 / decision D3): the section
renders in a language when it has a header in that language and a
body text, a background image, or a body image in it. The body
image is language-neutral, so it satisfies the content check for
either language:

```go
if body == "" && section.DesktopMediaID <= 0 && section.MobileMediaID <= 0 && section.BodyMediaID <= 0 {
  return "", "", "", false
}
```

In the `CustomSection` component, after the body `<p>`, render
the body image below the body text (decision D4) when
`section.BodyMediaID > 0`:

```templ
if section.BodyMediaID > 0 {
  <img class="custom-section-body-image" src={ publicImageURL(section.BodyMediaID) } alt={ header } loading="lazy" decoding="async"/>
}
```

The alt text is the section header, as the background `<img>`
already uses it. No `<picture>` and no viewport variant: one
image for both viewports.

Add the CSS in `static/css/styles.css`, next to
`.custom-section-body`: `.custom-section-body-image` is a block,
centred, at content width, scaling down on mobile:

```css
.custom-section-body-image {
  display: block;
  max-width: 100%;
  height: auto;
  margin: var(--sp-4) auto 0;
}
```

It sits inside `.vp-section-inner`, so it inherits the content
width; `max-width: 100%` scales it to the viewport width on
mobile. No ivory-text rule is needed: the image carries no text.
The with-background text rule is unchanged.

Tests: `helpers_test.go` covers `customSectionText` with a body
image only — header plus body image renders in both languages
with no body text and no background, and a header with nothing
else still does not render. `home_test.go` renders `Home` with a
section that has body text and a body image and asserts the image
element follows the body paragraph, and with a section that has
only a header and a body image and asserts the section renders
with the image and no body paragraph. The render tests reuse
`homeFixtures()` with a `BodyMediaID` on the fixture section.

### Phase 2: Dashboard and save

In `internal/handlers/dashboard.go`, add `section.BodyMediaID`
to `collectExistingMediaIDs`, next to the desktop/mobile adds, so
the post-save orphan sweep drops a body image that is no longer
referenced.

In `parseCustomSections`, resolve the body image through
`resolveImageMediaIDFromSet` — the set variant, not the
single-image `resolveImageMediaID`. The single-image variant
deletes the replaced media row immediately, while the set variant
leaves that to the post-save sweep, which is how the desktop and
mobile fields in the same function already behave; mixing the two
would delete a row the sweep still expects to see. Field names:

```text
custom_section_body_image_<i>
custom_section_body_image_media_id_<i>
```

Add `bodyMediaID == 0` to the empty-card skip check, and to the
content validation, which becomes: a non-empty card needs a header
in at least one language and a body text, a background image, or a
body image in at least one language — otherwise the whole save is
rejected with a 400. Keep the existing error style and update the
message to name the body image. Add `BodyMediaID` to the struct
literal. The declared-count drift guard is unchanged.

In `templates/dashboard.templ`, in the Custom Sections accordion:

- Relabel the background fields (decision D1), in the rendered
  cards and in the `<template>` card: "Desktop / horizontal"
  becomes "Background — Desktop / horizontal" and
  "Mobile / vertical" becomes "Background — Mobile / vertical".
  The dashboard labels are English throughout (for example
  "Header (Italian)"), so the labels are the English strings from
  D1; there are no Italian dashboard labels to translate.
- Add one managed-image field per card, labelled "Body image",
  after the background pair grid: a file input with
  `data-target-input="custom-section-body-image-data-<idx>"`,
  `data-preview-target="custom-section-body-image-preview-<idx>"`,
  `data-media-id-input="custom-section-body-image-media-id-<idx>"`,
  `data-format="image/jpeg"`, `data-quality="0.78"`,
  `data-max-width="1600"`, `data-max-height="1600"`,
  `data-max-bytes="262144"`, the hidden data and media-id inputs
  (`custom_section_body_image_<idx>` /
  `custom_section_body_image_media_id_<idx>`, classes
  `custom-section-image-hidden` /
  `custom-section-media-id-hidden`), and the preview `<img>`
  (`hero-background-preview`, alt "body image preview"), following
  the desktop field exactly. The limits are the desktop
  background limits with the height raised to 1600, so a portrait
  body photo is not squeezed to a landscape box.
- Add the same field to the `<template>` card with `__INDEX__`
  placeholders.
- Extend `reindexCustomSections` with a body-image block mirroring
  the desktop/mobile blocks (file, hidden, media-id, preview).
- Update the accordion help text: "A section needs a header and a
  body text, a background image, or a body image." Update the
  accordion description to mention the optional body image.

Tests: handler tests in `internal/handlers/settings_test.go`,
posting the settings form through `newTestApp()`:

- A section with a header and a body image but no body text and
  no background saves and round-trips through
  `GetCustomSections` with the `BodyMediaID` set.
- A section with a header and a body image saves when the body
  image is the only content (same case, asserted on the 200 and
  the stored row).
- A section with a header only (no body, no background, no body
  image) still returns 400 and leaves the previous sections
  untouched.
- Removing a body image on a later save deletes the orphaned
  media row (`database.GetMedia` on the old ID returns no rows),
  following the existing orphan-cleanup assertion pattern in the
  same file.

The templates package needs no new dashboard test: the accordion
renders through the existing settings render.

### Phase 3: Verification

Run `templ generate`, then `gofumpt`, then the full test suite
with race detection and coverage, then the per-package 90% gate
with the AWK check from the pipeline. Run the preflight checks:
fetch the remote, branch from the current remote default, read
the diff against it, scan for conflict markers and placeholder
text, and count the commits. Open the spec pull request first,
then the implementation pull request with one commit per phase
above, both based on master, wait for CI on each, and fix a red
branch by amending the failing commit.

Note on the concurrent fix: a separate styling fix for the
coordinate gift card (branch `fix/coordinate-card-labels`)
touches `templates/home.templ`, `static/css/styles.css`, and
`templates/home_test.go` at the same time as this cycle. If it
merges before this implementation branch, rebase this branch
onto the new master (explicit-lease force push) and re-verify
CI before reporting. The spec branch touches only the spec
directory and cannot conflict with it.

## Complexity Tracking

Empty. No constitution violation needs a justification.
