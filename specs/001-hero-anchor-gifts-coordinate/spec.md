# Feature Specification: Hero Text Anchor and Gifts Coordinate Mode

**Feature Branch**: `feat/hero-anchor-gifts-coordinate`

**Created**: 2026-11-26

**Status**: Ratified by the repository owner on 2026-11-26.

**Input**: User description: "Dalle impostazioni della prima sezione in
homepage, bisogna dare la possibilità all'admin di selezionare un
anchor type: top, center, bottom (bottom default e current). Bisogna
poter aggiungere sezioni custom. Switch per la sezione regali:
default è il current (regali con modal di conferma per l'aggiunta ai
doni su db), ma aggiungiamo modalità coordinate. Se scelta la seconda,
verrà mostrato soltanto un singolo box regalo con IBAN in monospace,
sotto il beneficiario (font standard, no monospace) e sotto invece del
pulsante Dona, il pulsante Copia per copiare in clipboard l'IBAN. Il
titolo di default in questo caso passa da Lista nozze a Regalo
(chiaramente overridabile per lingua come ogni altro label nel sito)."

## Clarifications

### Session 2026-11-26

- The anchor is the position of the text block of the first homepage
  section. The anchor is not the position of the background image.
  The background image stays exactly as it is. The investigation first
  read the anchor as the image position. The owner corrected that
  reading on 2026-11-26.
- The anchor values are top, center, bottom. Bottom is the default.
  Bottom equals the current rendering: the text block today sits at
  its current position, and bottom preserves that position exactly.
- The gifts section gains a mode switch: standard mode and coordinate
  mode. Standard mode is the default. Standard mode is the current
  behaviour.
- The title of the gifts section has separate keys per mode, per
  language. Each key is overridable in the dashboard, like every
  other label.
- In coordinate mode, the Copia button becomes disabled after one
  click. The button becomes clickable again after a few seconds.
- Custom sections are not part of this cycle. The owner confirmed
  that custom-section positions count all homepage sections, and that
  the built-in honeymoon section renders only when honeymoon places
  exist in the database. Those facts belong to the custom-sections
  cycle, `specs/002`, not to this specification.

## Hypothesis Challenge

The change rests on two hypotheses: the admin cannot control the
position of the first-section text, and the gifts section cannot show
bank coordinates without the claim flow. The investigation of
2026-11-26 tested both hypotheses on the real code at
`origin/master` 96e32da.

Evidence for the anchor:

- No anchor setting exists anywhere: no field in the settings model,
  no field in the database schema, no control in the dashboard.
  The admin cannot control the text position today.
- The first section renders the text block inside `#landing`
  (`templates/home.templ:214`). The text block today sits at its
  current position, set in `static/css/styles.css:275-277`.
- The first reading of the request was wrong. The investigation
  read "bottom is the default and the current behaviour" as a claim
  about the background image. That claim is false for the image:
  the image has `object-fit: cover` with no position declared
  (`styles.css:293-297`), so the browser default applies, which is
  center, not bottom. The owner corrected the reading: the anchor
  applies to the text, and bottom means the current text position.
  The specification records the correction here so no later reader
  repeats the wrong reading.
- Alternatives examined and rejected: one anchor per hero
  background pair. The text block is one block per page, so one
  global anchor is the simpler form. The per-pair form would add
  one control per background card for a text that does not change
  with the background.

Evidence for the coordinate mode:

- No coordinate mode exists: no mode switch, no single-box render.
  The gifts section always renders the gift tiles and the claim flow.
- The data the mode needs already exists. The IBAN and the
  beneficiary already live in the settings, and the dashboard
  already edits them (`templates/dashboard.templ:853-854`). The
  transfer modal already shows them
  (`templates/home.templ:659-667`). The copy pattern already exists:
  a copy button with clipboard code (`home.templ:967`) and the
  `btn.copy` and `btn.copied` labels (`internal/i18n/i18n.go`).
- The section renders only when both values are present
  (`internal/handlers/home.go:108`).
- Alternatives examined and rejected: hiding the registry items in
  the admin to fake the mode. The generic-gift tile and the whole
  claim flow would still render, so that is not equivalent to a
  single gift box.

## User Scenarios and Testing

### User Story 1: Admin anchors the first-section text (Priority: P1)

The admin opens the settings of the first homepage section and
selects an anchor: top, center, or bottom. The public homepage shows
the first-section text at the selected position. Bottom is the
default, and bottom shows the text exactly where it shows today.

**Why this priority**: The anchor is half of this cycle, and it is
the change the owner asked for first.

**Independent Test**: Change only the anchor setting and load the
homepage. The text moves to the selected position, and nothing else
on the page changes.

**Acceptance Scenarios**:

1. **Given** the anchor is unset, **When** the homepage loads,
   **Then** the text sits at its current position.
2. **Given** the admin selects top, **When** the homepage loads,
   **Then** the text sits at the top of the first section.
3. **Given** the admin selects center, **When** the homepage loads,
   **Then** the text sits at the center of the first section.
4. **Given** the admin selects bottom, **When** the homepage loads,
   **Then** the text sits exactly where it sits today.

### User Story 2: Admin switches the gifts section to coordinate mode (Priority: P1)

The admin switches the gifts section to coordinate mode. The public
homepage shows a single gift box instead of the gift tiles and the
claim flow. The box shows the IBAN in monospace, the beneficiary
below it in standard font, and a Copia button below the beneficiary.
The visitor clicks Copia, and the IBAN lands in the clipboard. The
section title reads "Regalo" in Italian by default.

**Why this priority**: The coordinate mode is the other half of this
cycle, and it is independently testable from the anchor.

**Independent Test**: Switch only the gifts mode to coordinate and
load the homepage. The single box renders with the IBAN, the
beneficiary, and the Copia button, and no gift tile or claim control
renders.

**Acceptance Scenarios**:

1. **Given** the gifts mode is standard, **When** the homepage loads,
   **Then** the gift tiles and the claim flow render as today.
2. **Given** the gifts mode is coordinate, **When** the homepage loads,
   **Then** only one gift box renders, with no gift tile and no claim
   control.
3. **Given** coordinate mode, **When** the visitor reads the box,
   **Then** the IBAN shows in monospace and the beneficiary below it
   shows in standard font.
4. **Given** coordinate mode, **When** the visitor clicks Copia,
   **Then** the IBAN is copied to the clipboard, the button disables,
   and the button becomes clickable again after a few seconds.
5. **Given** coordinate mode, **When** the homepage loads in Italian,
   **Then** the title reads "Regalo" by default.
6. **Given** coordinate mode, **When** the admin overrides the
   coordinate title for a language, **Then** the override shows in
   that language.

### Edge Cases

- What happens when the IBAN or the beneficiary is empty in
  coordinate mode? The section does not render, as today (see A2).
- What happens when the clipboard write fails? The button must not
  stay disabled. The button returns to a clickable state.
- What happens when the anchor or the mode is unset on existing
  data? The anchor falls back to bottom, and the mode falls back to
  standard.
- What happens when the visitor switches language in coordinate
  mode? The title follows the language, with its own override per
  language.

## Requirements

### Functional Requirements

- **FR-001**: The admin MUST select an anchor for the text of the
  first homepage section: top, center, or bottom.
- **FR-002**: The anchor MUST default to bottom.
- **FR-003**: Bottom MUST preserve the current rendering of the text
  exactly.
- **FR-004**: The admin MUST switch the gifts section between
  standard mode and coordinate mode.
- **FR-005**: The gifts mode MUST default to standard mode.
- **FR-006**: Coordinate mode MUST show a single gift box, with no
  gift tile and no claim control.
- **FR-007**: The coordinate box MUST show the IBAN in monospace.
- **FR-008**: The coordinate box MUST show the beneficiary below the
  IBAN in standard font, not monospace.
- **FR-009**: The coordinate box MUST show a Copia button below the
  beneficiary instead of the Dona button.
- **FR-010**: The Copia button MUST copy the IBAN to the clipboard.
- **FR-011**: The Copia button MUST disable after one click and
  become clickable again after a few seconds.
- **FR-012**: The coordinate title MUST default to "Regalo" in
  Italian and to "Gift" in English. The English default is proposed
  (see A5).
- **FR-013**: The standard title MUST keep its current defaults:
  "Lista nozze" in Italian and "Wedding List" in English.
- **FR-014**: Each mode title MUST have its own overridable key per
  language, editable in the dashboard like every other label.
- **FR-015**: Coordinate mode MUST NOT write a gift record
  [ASSUMPTION A1 — to confirm at ratification].
- **FR-016**: The coordinate box MUST render only when the IBAN and
  the beneficiary are both configured [ASSUMPTION A2 — to confirm at
  ratification].
- **FR-017**: The Copia button MUST stay available after the event
  date, while the Dona button disables [ASSUMPTION A3 — to confirm at
  ratification].

### Key Entities

- **Homepage settings**: one settings record per instance. It holds
  the anchor and the gifts mode.
- **Gifts section mode**: standard or coordinate. Standard is the
  current behaviour. Coordinate is display only.
- **Gift coordinates**: the IBAN and the beneficiary. Both already
  exist in the settings. No new data source is needed.
- **Section title**: one title per mode, per language. Each title
  has its own default and its own dashboard override.

## Success Criteria

### Measurable Outcomes

- **SC-001**: The admin selects each anchor value, and the homepage
  shows the text at that position in each case.
- **SC-002**: With the anchor unset, the homepage shows the text
  exactly where it shows today.
- **SC-003**: The admin switches the gifts mode, and the homepage
  shows the single box in coordinate mode and the tiles in standard
  mode.
- **SC-004**: A visitor copies the IBAN with one click in coordinate
  mode, and the button disables and becomes clickable again after a
  few seconds.
- **SC-005**: The coordinate title reads "Regalo" in Italian by
  default, and an override for one mode and one language changes no
  other mode or language.
- **SC-006**: The change ships with the per-package 90% coverage
  gate green and the CI pipeline green on both live instances.

## Assumptions

- **A1 — to confirm at ratification**: Coordinate mode writes
  nothing to the database. The mode is pure display plus clipboard.
  It has no amount, no donor, and no item, so it has nothing to
  record.
- **A2 — to confirm at ratification**: The coordinate box still
  requires the IBAN and the beneficiary to be configured. The
  section does not render without both values, as today.
- **A3 — to confirm at ratification**: The Copia button stays
  available after the event date. The Dona button disables after
  the event date, as today; the copy buttons in the transfer modal
  are not gated by the event date today, and Copia follows them.
- **A4 — to confirm at ratification**: The section description
  (`home.registry_description`) still shows in coordinate mode.
  The specification does not state otherwise.
- **A5 — to confirm at ratification**: The English default title in
  coordinate mode is "Gift". The owner stated only the Italian
  default, "Regalo".
- **A6 — to confirm at ratification**: The anchor is one global
  setting for the first section. The text block is one block per
  page, so one setting covers it.

## Out of Scope

- Custom sections. Custom sections belong to `specs/002`. Their
  open questions (background type, per-language texts, image form
  and placement) are unanswered and are not decided here.
- Any change to the background image of the first section. The
  image stays exactly as it is.
- Any change to the built-in honeymoon section. Its rendering rule
  (only when honeymoon places exist) does not change.
- Any change to the standard gift claim flow. The claim flow
  behaves exactly as today in standard mode.
