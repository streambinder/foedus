# Tasks: Custom Homepage Sections

**Input**: Design documents from `/specs/002-custom-sections/`

**Prerequisites**: plan.md (required), spec.md (required for user stories)

**Tests**: Tests are required by the constitution: the per-package 90%
line-coverage gate binds every touched package, and non-test code
never changes to satisfy a test.

**Organization**: Tasks are grouped by phase and by user story so each
story can be implemented and tested independently.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (e.g., US1, US2)
- Include exact file paths in descriptions

## Phase 0: Foundational (Blocking Prerequisites)

**Purpose**: Model, table, database helpers, and content structs
that both the render and the dashboard depend on

**⚠️ CRITICAL**: No user story work can begin until this phase is complete

- [ ] T001 Add the `custom_sections` table to `migrate()` in
      `internal/database/database.go`, STRICT, after the
      `hero_backgrounds` block
- [ ] T002 [P] Add the `CustomSection` struct to
      `internal/models/models.go`
- [ ] T003 [P] Add `internal/database/custom_sections.go` with
      `scanCustomSection`, `GetCustomSections`, and
      `ReplaceCustomSections`, following `hero_backgrounds.go`
- [ ] T004 [P] Add `CustomSections []models.CustomSection` to
      `HomeContent` and to `SettingsContent` in
      `templates/content.go`
- [ ] T005 [P] Add database tests for the new table: round trip
      through `ReplaceCustomSections` and `GetCustomSections` in
      form order, and the empty table, in
      `internal/database/collections_test.go`

**Checkpoint**: Foundation ready — user story implementation can now begin

## Phase 1: Ordering and rendering (User Stories 1–4) 🎯 MVP

**Goal**: Custom sections render at their position among the
built-in sections, in the visitor's language, with the arrows
navigating through them.

**Independent Test**: Add one custom section at position 2 and
load the homepage. It renders between the landing section and the
next section, with its texts and its full-bleed background, and
the arrows move through it.

### Tests for ordering and rendering

- [ ] T010 [P] [US2] Template helper tests for `planHomeSections`:
      base order with no customs, a custom at position 2, a custom
      past the end, a tie at one position, neighbours at the ends,
      in `templates/helpers_test.go` (replaces
      `TestSectionNavigationHelpers`)
- [ ] T011 [P] [US3] Template helper tests for `customSectionText`
      and `customSectionBackground`: per-language visibility (A1)
      and per-viewport fallback (A3), in
      `templates/helpers_test.go`
- [ ] T012 [P] [US1, US2, US3, US4] Template render tests: `Home`
      renders a custom section with its ID, full-bleed picture,
      per-language texts, and position 2 order; a custom honeymoon
      section renders alone with no honeymoon places and beside
      the built-in section with honeymoon places, in
      `templates/home_test.go`

### Implementation for ordering and rendering

- [ ] T013 Load custom sections in `loadHomeContent` in
      `internal/handlers/home.go`
- [ ] T014 [P] Replace the six hardcoded arrow helpers in
      `templates/home.templ` with `planHomeSections`,
      `prevAnchor`, `nextAnchor`, `customSectionText`, and
      `customSectionBackground`
- [ ] T015 Add the `CustomSection` templ component in
      `templates/home.templ`: full-bleed background picture and
      wash reused from the hero, texts in `.section-heading`,
      `.section-title`, `.section-subtitle`, and
      `.custom-section-body`, arrows from the plan
- [ ] T016 Render the plan in `Home` in `templates/home.templ`:
      compute the plan once, replace every hardcoded arrow with
      the plan neighbours, render the customs assigned before
      each built-in section, and the past-the-end customs after
      the registry block
- [ ] T017 [P] Add the `.vp-custom-section` and
      `.custom-section-body` rules in `static/css/styles.css`,
      with ivory text over the wash when a background is present

**Checkpoint**: At this point, custom sections render and order
correctly; only the dashboard editing is missing

## Phase 2: Dashboard and save (User Stories 1–3)

**Goal**: The admin adds, edits, and deletes custom sections from
the dashboard, and an invalid section rejects the whole save.

**Independent Test**: Post the settings form with one custom
section and load the homepage. The section renders. Post an
invalid section and the save returns 400 with the previous
sections untouched.

### Tests for dashboard and save

- [ ] T020 [P] [US1, US3] Handler tests: a section with a header
      and a body saves and round-trips through
      `GetCustomSections`; a section with a header and an image
      but no body saves; a section with a header and a body but
      no image saves, in `internal/handlers/settings_test.go`
- [ ] T021 [P] [US1] Handler tests: the rejections return 400 and
      leave the previous sections untouched — no header, neither
      body nor image, bad position — in
      `internal/handlers/settings_test.go`

### Implementation for dashboard and save

- [ ] T022 Load custom sections in `loadSettingsContent` and add
      their media IDs to `collectExistingMediaIDs` in
      `internal/handlers/dashboard.go`
- [ ] T023 Add `parseCustomSections` in
      `internal/handlers/dashboard.go`, following
      `parseHeroBackgrounds`, with the A2 validation and the
      declared-count drift guard
- [ ] T024 Wire the parse into `SaveSettings` in
      `internal/handlers/dashboard.go`, add
      `ReplaceCustomSections` to the writes list, and the count to
      the success log line
- [ ] T025 Add the Custom Sections accordion to
      `templates/dashboard.templ`, after the Homepage Landing
      Backgrounds accordion: card per section with the position
      input, per-language header, subtitle, and body fields, two
      managed-image fields, the `<template>` card, and the
      add/remove/reindex script mirroring the hero-background card

**Checkpoint**: At this point, User Stories 1, 2, and 3 should
all work independently

## Phase 3: Verification

**Purpose**: Prove the change against the pipeline before filing

- [ ] T030 Run `templ generate`, then `gofumpt`, then the full test
      suite with race detection and coverage
- [ ] T031 Run the per-package 90% coverage gate with the AWK check
      from `.github/workflows/push.yml`
- [ ] T032 Run the preflight checks: fetch the remote, read the diff
      against the remote default branch, scan for conflict markers
      and placeholder text, count the commits
- [ ] T033 Open the implementation pull request with one commit per
      plan phase, wait for CI, and fix a red branch by amending the
      failing commit

## Dependencies & Execution Order

### Phase Dependencies

- **Foundational (Phase 0)**: No dependencies — blocks all user stories
- **Ordering and rendering (Phase 1)**: Depends on Foundational completion
- **Dashboard and save (Phase 2)**: Depends on Foundational completion —
  independent of Phase 1 except for the shared `CustomSection` shape
- **Verification (Phase 3)**: Depends on all phases

### Parallel Opportunities

- All tasks marked [P] can run in parallel (different files, no
  dependencies)
- Phases 1 and 2 can proceed in parallel once Foundational completes

## Implementation Strategy

### MVP First (Phase 1 Only)

1. Complete Phase 0: Foundational
2. Complete Phase 1: Ordering and rendering
3. **STOP and VALIDATE**: Test the render independently

### Incremental Delivery

1. Complete Foundational → Foundation ready
2. Add ordering and rendering → Test independently
3. Add dashboard and save → Test independently
4. Verify → File the implementation pull request

## Notes

- [P] tasks = different files, no dependencies
- [Story] label maps task to specific user story for traceability
- Each user story should be independently completable and testable
- Commit after each phase, one commit per phase
- Stop at any checkpoint to validate the story independently
- Avoid: vague tasks, same file conflicts, cross-story dependencies
  that break independence
