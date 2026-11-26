# Implementation Plan: Hero Text Anchor and Gifts Coordinate Mode

**Branch**: `feat/hero-anchor-gifts-coordinate` | **Date**: 2026-11-26 | **Spec**: [spec.md](spec.md)

**Input**: Feature specification from `specs/001-hero-anchor-gifts-coordinate/spec.md`

## Summary

Add a global text-anchor setting for the first homepage section and a
coordinate mode for the gifts section. The anchor moves the text block
of the first section to the top, the center, or the bottom. Bottom is
the default and preserves the current rendering exactly. The gifts
mode switch keeps the current claim flow as the standard mode and adds
a coordinate mode that shows one box with the IBAN, the beneficiary,
and a Copia button. The change adds two columns to the single settings
row, one dashboard select per setting, one i18n key for the coordinate
title, and one template branch per story. The background image of the
first section does not change.

## Technical Context

**Language/Version**: Go 1.26 (module `go 1.26.0`), toolchain 1.27.1

**Primary Dependencies**: Fiber v2.52.16, templ v0.3.1118 (library),
templ generator v0.3.1100, SQLite driver v1.62.0 (modernc.org)

**Storage**: One SQLite file per instance, STRICT tables, WAL mode.
The settings table holds exactly one row (`CHECK (id = 1)`).

**Testing**: Unit tests sit beside the code they cover (`*_test.go`).
`templ generate` runs before the tests. CI runs
`go test -race -p 1 -cover ./...` and fails any package below 90%
line coverage.

**Target Platform**: linux/arm64 container, deployed on lobster as
two instances (`foedus-davideagnese`, `foedus-lorenzomargherita`),
each with its own SQLite file at
`/var/lib/foedus/<instance>/data.db`.

**Project Type**: Server-rendered web application, single Go binary.

**Constraints**: Code-side migrations create only the canonical
schema. The two new columns go into the `CREATE TABLE settings`
block. The live databases gain the columns through a manual rollout,
once, before the deploy: the new code reads the columns on every
page load, so the deploy must follow the migration, not precede it.
The old code ignores the extra columns, so the migration is safe to
run while the old build still serves.

**Scale/Scope**: Two settings columns, two dashboard selects, one
i18n key, one template branch per story, one CSS addition per anchor
value. No new table, no new endpoint, no new route.

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
- Principle V (minimal owned surface): no new dependency. The copy
  flow reuses the existing clipboard code. The anchor reuses the
  existing flex layout of `.vp-section`.
- Principle VI (commits and pull requests): one commit per plan
  phase. Each file appears in exactly one commit. The owner merges.
- Principle VII (containers and releases): no Dockerfile change.
  Publish never runs on a pull request.
- Principle VIII (writing and consistency): the change follows the
  conventions of each file it touches. It does not reformat parts it
  does not touch.
- Database Migrations (§26): the code-side change is the canonical
  schema only. The `ALTER TABLE` rollout is manual, once, against
  each live database, with backup, migration, and verification
  commands proposed first.
- Production Care: backup both live databases before the deploy.
  Verify both instances answer after the deploy. Never write on
  production data to test.
- Hypothesis First: the investigation of 2026-11-26 verified both
  gaps on the real code at `origin/master` 96e32da. The owner
  corrected the anchor reading from image to text, and the
  specification records that correction.

No violation. The complexity tracking section stays empty.

## Project Structure

### Documentation (this feature)

```text
specs/001-hero-anchor-gifts-coordinate/
├── spec.md              # Ratified specification
├── plan.md              # This file
└── tasks.md             # Ordered task list
```

### Source Code (repository root)

```text
internal/models/models.go             # Settings struct: two new fields
internal/database/database.go         # Canonical CREATE TABLE settings
internal/database/settings.go         # settingsColumns, Get/UpdateSettings
internal/handlers/dashboard.go        # SaveSettings: parse and validate
internal/handlers/home.go             # bankConfigured (unchanged)
internal/i18n/i18n.go                 # home.registry_coordinate key
templates/dashboard.templ             # Two new selects
templates/home.templ                  # Anchor class, coordinate box
static/css/styles.css                 # Anchor rules for top and center
```

**Structure Decision**: The change follows the existing settings
pattern end-to-end: one column per setting, one struct field, one
explicit entry in `settingsColumns` and in `UpdateSettings`, one
dashboard form field parsed in `SaveSettings` inside the existing
transaction. No new package and no new file outside the two spec
artifacts.

## Approach

### Phase 0: Foundation — settings columns and the coordinate title

Add `hero_text_anchor TEXT NOT NULL DEFAULT 'bottom'` and
`gifts_mode TEXT NOT NULL DEFAULT 'standard'` to the `CREATE TABLE
settings` block in `internal/database/database.go`. Add the matching
`HeroTextAnchor string` and `GiftsMode string` fields to
`models.Settings`, the two column names to `settingsColumns`, and the
two assignments to `UpdateSettings` in
`internal/database/settings.go`. Add one validation helper per
setting, or one helper over an allowed set, in the models or
handlers package: the anchor accepts only `top`, `center`,
`bottom`; the mode accepts only `standard`, `coordinate`. Any other
value is a 400 from `SaveSettings`, through the existing
`fiber.NewError(400, ...)` pattern, so a rejected save rolls the
whole transaction back.

Add the i18n key `home.registry_coordinate` to `HomepageKeys` in
`internal/i18n/i18n.go`, with the defaults `Gift` (English) and
`Regalo` (Italian). The key becomes dashboard-overridable per
language through the existing `homepage_labels` mechanism, with no
new storage. The standard title keeps the `home.registry` key and
its current defaults.

The manual rollout for the two live databases runs before the
deploy, once per instance, and the owner runs it or approves it:

```sql
ALTER TABLE settings ADD COLUMN hero_text_anchor TEXT NOT NULL DEFAULT 'bottom';
ALTER TABLE settings ADD COLUMN gifts_mode TEXT NOT NULL DEFAULT 'standard';
```

The implementation pull request carries the backup, migration, and
verification commands. The deploy follows the migration.

### Phase 1: User Story 1 — hero text anchor

Add one select to the first-section settings in the dashboard, in
the Homepage Landing Backgrounds accordion of
`templates/dashboard.templ`. The select posts `hero_text_anchor`
and offers top, center, bottom. `SaveSettings` in
`internal/handlers/dashboard.go` parses the value, validates it
against the allowed set, and stores it in the transaction.

Render the anchor as a class on the landing section in
`templates/home.templ`: the settings value maps to
`vp-hero-anchor-top`, `vp-hero-anchor-center`, or
`vp-hero-anchor-bottom`. The `Home` handler already receives the
settings, so no handler change is needed for the render.

Add CSS in `static/css/styles.css` for top and center only. The
bottom rule stays exactly as it is today:
`#landing.vp-hero > .vp-section-inner` keeps
`transform: translateY(25vh)` and `translateY(25dvh)`. The section
is a flex column with `justify-content: center`, so center needs
only the transform removed, and top needs the transform removed
plus `justify-content: flex-start` on the anchored section. The
background image rules do not change.

Tests: dashboard handler tests cover an accepted value per anchor,
a rejected value with a 400, and the bottom default on an unset
value. Template tests render `Home` with each anchor value and
assert the class. The bottom render asserts the class that keeps
the current transform.

### Phase 2: User Story 2 — gifts coordinate mode

Add one select to the dashboard near the bank account fields in
`templates/dashboard.templ`. The select posts `gifts_mode` and
offers standard and coordinate. `SaveSettings` parses and validates
it like the anchor.

In `templates/home.templ`, branch the registry section on
`settings.GiftsMode`. Standard mode renders exactly as today:
title from `home.registry`, gift tiles, free-gift tile, claim
modals. Coordinate mode renders the title from
`home.registry_coordinate`, the section description, and one box
only: the IBAN in a `<code>` element, the beneficiary below it in
standard font, and a Copia button below the beneficiary. No gift
tile, no claim control, and no claim modal renders in coordinate
mode. The section still renders only when `bankConfigured` is
true. The Copia button carries no `EventPassed` gate, unlike the
Dona buttons.

Rework the copy wiring. The current `.copy-btn` handler lives
behind the `if (!transferModal) return` guard, and coordinate mode
renders no transfer modal, so the handler never binds there. Move
the copy-button binding out of that guard so it binds whenever a
`.copy-btn` exists on the page. On click, the handler copies the
IBAN through `navigator.clipboard`, disables the button, swaps the
label to the copied label, and after two seconds restores the label
and re-enables the button. A failed clipboard write must not leave
the button disabled: the button returns to a clickable state on
failure. No new endpoint is added, and coordinate mode performs no
database write.

Tests: dashboard handler tests cover both mode values and a
rejected value. Template tests render `Home` in coordinate mode
and assert the single box, the IBAN element, the beneficiary
element, the Copia button, and the absence of gift tiles and claim
controls; and render standard mode and assert the tiles return.
I18n tests assert the coordinate defaults per language and that an
override of one mode title in one language changes no other mode
or language.

### Phase 3: Verification

Run `templ generate`, then `gofumpt`, then the full test suite
with race detection and coverage, then the per-package 90% gate
with the AWK check from the pipeline. Run the preflight checks:
fetch the remote, branch from the current remote default, read
the diff against it, scan for conflict markers and placeholder
text, and count the commits. Open the implementation pull request
with one commit per phase above, wait for CI, and fix a red
branch by amending the failing commit.

## Complexity Tracking

Empty. No constitution violation needs a justification.
