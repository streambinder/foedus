# Tasks: Hero Text Anchor and Gifts Coordinate Mode

**Input**: Design documents from `/specs/001-hero-anchor-gifts-coordinate/`

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

## Phase 1: Foundational (Blocking Prerequisites)

**Purpose**: Settings columns, validation, and the coordinate title
that both user stories depend on

**⚠️ CRITICAL**: No user story work can begin until this phase is complete

- [ ] T001 Add `hero_text_anchor` and `gifts_mode` columns to the
      `CREATE TABLE settings` block in `internal/database/database.go`
- [ ] T002 [P] Add `HeroTextAnchor` and `GiftsMode` fields to
      `models.Settings` in `internal/models/models.go`
- [ ] T003 Add both columns to `settingsColumns` and both assignments
      to `UpdateSettings` in `internal/database/settings.go`
- [ ] T004 Add validation for the anchor values (`top`, `center`,
      `bottom`) and the mode values (`standard`, `coordinate`), rejected
      with a 400 in `SaveSettings`
- [ ] T005 [P] Add the `home.registry_coordinate` key to
      `HomepageKeys` in `internal/i18n/i18n.go`, with the defaults
      `Gift` (English) and `Regalo` (Italian)
- [ ] T006 [P] Add database tests for the new columns: round-trip
      through `GetSettings` and `UpdateSettings`, and defaults on a
      fresh database in `internal/database/settings_test.go`
- [ ] T007 [P] Add i18n tests for the coordinate title defaults per
      language in `internal/i18n/i18n_test.go`

**Checkpoint**: Foundation ready — user story implementation can now begin

## Phase 2: User Story 1 — Admin anchors the first-section text (Priority: P1) 🎯 MVP

**Goal**: The admin selects top, center, or bottom for the text of
the first homepage section. Bottom is the default and preserves the
current rendering exactly.

**Independent Test**: Change only the anchor setting and load the
homepage. The text moves to the selected position, and nothing else
on the page changes.

### Tests for User Story 1

- [ ] T010 [P] [US1] Dashboard handler test: each anchor value
      saves, an invalid value returns 400, an unset value falls back to
      bottom, in `internal/handlers/dashboard_test.go`
- [ ] T011 [P] [US1] Template test: `Home` renders the anchor class
      for top, center, and bottom, in `templates/home_test.go`

### Implementation for User Story 1

- [ ] T012 [P] [US1] Add the anchor select (`hero_text_anchor`) to
      the Homepage Landing Backgrounds accordion in
      `templates/dashboard.templ`
- [ ] T013 [US1] Parse and validate `hero_text_anchor` in
      `SaveSettings` in `internal/handlers/dashboard.go` (depends on T004)
- [ ] T014 [US1] Render the anchor class on the landing section in
      `templates/home.templ`
- [ ] T015 [US1] Add the top and center rules in
      `static/css/styles.css`; leave the bottom rule untouched

**Checkpoint**: At this point, User Story 1 should be fully
functional and testable independently

## Phase 3: User Story 2 — Admin switches the gifts section to coordinate mode (Priority: P1)

**Goal**: The admin switches the gifts section to coordinate mode.
The homepage shows one box with the IBAN in monospace, the
beneficiary below it in standard font, and a Copia button below
the beneficiary instead of the Dona button.

**Independent Test**: Switch only the gifts mode to coordinate and
load the homepage. The single box renders with the IBAN, the
beneficiary, and the Copia button, and no gift tile or claim control
renders.

### Tests for User Story 2

- [ ] T020 [P] [US2] Dashboard handler test: both mode values save
      and an invalid value returns 400, in
      `internal/handlers/dashboard_test.go`
- [ ] T021 [P] [US2] Template test: `Home` in coordinate mode
      renders the single box and no gift tile or claim control, and in
      standard mode renders the tiles, in `templates/home_test.go`
- [ ] T022 [P] [US2] I18n test: an override of the coordinate title
      in one language changes no other mode or language, in
      `internal/i18n/i18n_test.go`

### Implementation for User Story 2

- [ ] T023 [P] [US2] Add the gifts mode select (`gifts_mode`) near
      the bank account fields in `templates/dashboard.templ`
- [ ] T024 [US2] Parse and validate `gifts_mode` in `SaveSettings`
      in `internal/handlers/dashboard.go` (depends on T004)
- [ ] T025 [US2] Branch the registry section on `settings.GiftsMode`
      in `templates/home.templ`: coordinate mode renders the single box
      with the IBAN, the beneficiary, and the Copia button; standard
      mode renders as today
- [ ] T026 [US2] Move the `.copy-btn` binding out of the
      transfer-modal guard in `templates/home.templ`, disable the button
      on click, restore the label and re-enable it after two seconds,
      and return it to a clickable state when the clipboard write fails

**Checkpoint**: At this point, User Stories 1 AND 2 should both work
independently

## Phase 4: Verification

**Purpose**: Prove the change against the pipeline before filing

- [ ] T030 Run `templ generate`, then `gofumpt`, then the full test
      suite with race detection and coverage
- [ ] T031 Run the per-package 90% coverage gate with the AWK check
      from `.github/workflows/push.yml`
- [ ] T032 Run the preflight checks: fetch the remote, read the diff
      against the remote default branch, scan for conflict markers and
      placeholder text, count the commits
- [ ] T033 Open the implementation pull request with one commit per
      plan phase, wait for CI, and fix a red branch by amending the
      failing commit

## Dependencies & Execution Order

### Phase Dependencies

- **Foundational (Phase 1)**: No dependencies — blocks all user stories
- **User Story 1 (Phase 2)**: Depends on Foundational completion
- **User Story 2 (Phase 3)**: Depends on Foundational completion —
  independent of User Story 1
- **Verification (Phase 4)**: Depends on all user stories

### Parallel Opportunities

- All tasks marked [P] can run in parallel (different files, no
  dependencies)
- User Stories 1 and 2 can proceed in parallel once Foundational
  completes

## Implementation Strategy

### MVP First (User Story 1 Only)

1. Complete Phase 1: Foundational
2. Complete Phase 2: User Story 1
3. **STOP and VALIDATE**: Test User Story 1 independently

### Incremental Delivery

1. Complete Foundational → Foundation ready
2. Add User Story 1 → Test independently
3. Add User Story 2 → Test independently
4. Verify → File the implementation pull request

## Notes

- [P] tasks = different files, no dependencies
- [Story] label maps task to specific user story for traceability
- Each user story should be independently completable and testable
- Commit after each phase, one commit per phase
- Stop at any checkpoint to validate the story independently
- Avoid: vague tasks, same file conflicts, cross-story dependencies
  that break independence
