# Tasks: Body image for custom sections

**Input**: Design documents from `/specs/003-custom-section-body-image/`

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

**Purpose**: Model field, canonical table column, and database
helpers that both the render and the dashboard depend on

**⚠️ CRITICAL**: No user story work can begin until this phase is complete

- [ ] T001 Add the `body_media_id INTEGER REFERENCES media(id)`
      column to the `custom_sections` block in `migrate()` in
      `internal/database/database.go`, after `mobile_media_id`
- [ ] T002 [P] Add the `BodyMediaID int` field to the
      `CustomSection` struct in `internal/models/models.go`,
      after `MobileMediaID`, and update the struct comment
- [ ] T003 [P] Update `scanCustomSection`, `GetCustomSections`,
      and `ReplaceCustomSections` in
      `internal/database/custom_sections.go` to read, select, and
      insert `body_media_id`
- [ ] T004 [P] Extend the database round-trip test so a section
      with a body media ID round-trips through
      `ReplaceCustomSections` and `GetCustomSections`, and a
      section without one reads back as zero, in
      `internal/database/collections_test.go`

**Checkpoint**: Foundation ready — user story implementation can now begin

## Phase 1: Rendering (User Stories 1–2) 🎯 MVP

**Goal**: A custom section with a body image renders the image
below its body text, and a section with only a header and a body
image renders at all.

**Independent Test**: Add a custom section with a header and a
body image and no body text, and load the homepage. It renders
with the image. Add body text and the image sits below the text.

### Tests for rendering

- [ ] T010 [P] [US1] Template helper tests for
      `customSectionText`: header plus body image renders in both
      languages with no body text and no background, and a header
      with nothing else still does not render, in
      `templates/helpers_test.go`
- [ ] T011 [P] [US1, US2] Template render tests: `Home` renders a
      section with body text and a body image with the image
      element after the body paragraph, and a section with only a
      header and a body image with the image and no body
      paragraph, in `templates/home_test.go`

### Implementation for rendering

- [ ] T012 Update `customSectionText` in `templates/home.templ`
      so a body image counts as body content for the
      per-language render check
- [ ] T013 Add the body image branch to the `CustomSection`
      component in `templates/home.templ`: an `<img>` below the
      body paragraph when `BodyMediaID > 0`, with
      `publicImageURL`, the header as alt text, lazy loading
- [ ] T014 [P] Add the `.custom-section-body-image` rule in
      `static/css/styles.css`, next to `.custom-section-body`:
      block, centred, content width, scaling down on mobile

**Checkpoint**: At this point, custom sections with a body image
render correctly; only the dashboard editing is missing

## Phase 2: Dashboard and save (User Stories 1–3)

**Goal**: The admin adds, edits, and deletes a body image from
the dashboard, the background fields say they are background
images, and an invalid section rejects the whole save.

**Independent Test**: Post the settings form with a custom
section that has a header and a body image but no body text.
It saves and the homepage renders it. Post a header-only section
and the save returns 400 with the previous sections untouched.

### Tests for dashboard and save

- [ ] T020 [P] [US1, US3] Handler tests: a section with a header
      and a body image but no body text and no background saves
      and round-trips through `GetCustomSections` with the
      `BodyMediaID` set, in `internal/handlers/settings_test.go`
- [ ] T021 [P] [US1] Handler tests: a header-only section still
      returns 400 and leaves the previous sections untouched, in
      `internal/handlers/settings_test.go`
- [ ] T022 [P] [US3] Handler tests: removing a body image on a
      later save deletes the orphaned media row
      (`database.GetMedia` on the old ID returns no rows),
      following the existing orphan-cleanup assertion pattern in
      `internal/handlers/settings_test.go`

### Implementation for dashboard and save

- [ ] T023 Add `section.BodyMediaID` to `collectExistingMediaIDs`
      in `internal/handlers/dashboard.go`
- [ ] T024 Resolve the body image in `parseCustomSections` in
      `internal/handlers/dashboard.go` through
      `resolveImageMediaIDFromSet` with the
      `custom_section_body_image_<i>` /
      `custom_section_body_image_media_id_<i>` fields, add it to
      the empty-card skip check and to the content validation, and
      add `BodyMediaID` to the struct literal
- [ ] T025 Relabel the background fields in
      `templates/dashboard.templ` (rendered cards and `<template>`
      card) to "Background — Desktop / horizontal" and
      "Background — Mobile / vertical", update the accordion help
      text and description, and add one managed-image field
      labelled "Body image" per card and in the `<template>` card
      with the reindex block in `reindexCustomSections`

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
      plan phase, both pull requests based on master, wait for CI,
      and fix a red branch by amending the failing commit

## Dependencies & Execution Order

### Phase Dependencies

- **Foundational (Phase 0)**: No dependencies — blocks all user stories
- **Rendering (Phase 1)**: Depends on Foundational completion
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
2. Complete Phase 1: Rendering
3. **STOP and VALIDATE**: Test the render independently

### Incremental Delivery

1. Complete Foundational → Foundation ready
2. Add rendering → Test independently
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
