# Feature Specification: Custom Homepage Sections

**Feature Branch**: `feat/custom-sections`

**Created**: 2026-11-26

**Status**: Ratified 2026-11-26

**Input**: User description: "Bisogna poter aggiungere sezioni custom.
Per ciascuna: sfondo / no sfondo custom + header sezione + sottotitolo

- testo body + immagine da renderizzare (almeno header e (testo body
  o immagine) devono essere presenti). Bisogna poter indicare anche dove
  posizionare la sezione nella homepage (posizione 1, 2, etc)."

## Clarifications

### Session 2026-11-26

- The custom-section background is an image. There is no colour
  background. A section may also have no custom background.
- The section texts (header, subtitle, body) exist per language,
  Italian and English, like every other label on the site.
- The image is a desktop/mobile pair, as for the hero backgrounds.
  It renders full-bleed, as the hero background does.
- The background image and the image to render are one and the same
  image: the full-bleed background pair. There is no second image.
- Positions count all homepage sections, not only the custom ones.
  The landing section is position 1.
- A custom section never replaces a built-in section. Built-in
  sections render by their own rules. A custom section only inserts
  itself among them.
- The motivating use case is a custom honeymoon layout: the owners
  of the second live instance want their own honeymoon section. The
  built-in honeymoon section renders only when honeymoon places
  exist in the database. When no honeymoon places exist, the
  built-in section does not render, and the custom section stands
  alone in that place.

## Hypothesis Challenge

The change rests on one hypothesis: the admin cannot add a homepage
section of their own, so every homepage layout is fixed in code.
The investigation of 2026-11-26 tested this hypothesis on the real
code at `origin/master` 96e32da.

Evidence for the gap:

- No custom-section mechanism exists: no table, no model, no
  template, and no dashboard UI. A search for `CustomSection` and
  `custom_section` returns nothing.
- The homepage sections are hardcoded in render order in
  `templates/home.templ:214-530`: landing, venues, soundtrack,
  places, honeymoon, registry. Each renders only under its own
  condition.
- The up/down arrow navigation is a hardcoded chain of helpers
  (`prevSectionBeforeRegistry`, `nextSectionAfterVenues`,
  `nextSectionAfterSoundtrack`, `nextSectionAfterPlaces`,
  `nextSectionAfterHoneymoon`). The helpers name fixed sections.
  A new section cannot join the chain without changing them.
- The closest precedents are the `places` table (a kind
  discriminator with `sort_order` per kind) and the registry-item
  ordering (`MoveRegistryItem`, which swaps `sort_order`).

Alternatives examined and rejected:

- Reuse the `places` table with a new kind. Rejected: a place
  carries timeline and map logic that a free section does not need.
- Reuse registry items. Rejected: a registry item carries a price
  and a claim flow that a free section does not need.
- A free-text field on the settings record. Rejected: one field
  cannot express an ordered, repeatable, image-bearing list.
- A new relation with a position is the only form that fits. It
  follows the existing `places` and `hero_backgrounds` pattern:
  its own table, a sort order, and a wipe-and-reinsert save inside
  one transaction.

## User Scenarios and Testing

### User Story 1: Admin adds a custom section (Priority: P1)

The admin adds a custom section from the dashboard: a background
image pair (or no background), a header, a subtitle, a body text,
and a position. The public homepage shows the new section with the
background full-bleed, the header, the subtitle, and the body.

**Why this priority**: Adding the section is the change itself.

**Independent Test**: Add only one custom section and load the
homepage. The section renders with its texts and its background,
and nothing else on the page changes.

**Acceptance Scenarios**:

1. **Given** the admin adds a section with a header, a body, and a
   background image pair, **When** the homepage loads, **Then** the
   section renders with the background full-bleed and the texts.
2. **Given** the admin adds a section with a header and a background
   image pair but no body, **When** the homepage loads, **Then** the
   section renders with the header and the background.
3. **Given** the admin adds a section with a header and a body but no
   background image, **When** the homepage loads, **Then** the
   section renders on the page background.
4. **Given** the admin adds a section with a subtitle, **When** the
   homepage loads, **Then** the subtitle renders below the header.

### User Story 2: Admin positions a custom section among all sections (Priority: P1)

The admin gives the custom section a position. Positions count all
homepage sections, with the landing section at position 1. The
homepage renders the custom section at that place in the full
order, and the up/down arrows navigate through it.

**Why this priority**: Position is part of the request, and the
arrow chain is hardcoded today, so the order is only real when the
arrows follow it.

**Independent Test**: Add one custom section at position 2 and load
the homepage. The section renders between the landing section and
the next section, and the arrows move through it.

**Acceptance Scenarios**:

1. **Given** a custom section at position 2, **When** the homepage
   loads, **Then** the section renders after the landing section
   and before the next section.
2. **Given** a custom section at a position past the last section,
   **When** the homepage loads, **Then** the section renders last.
3. **Given** two custom sections at the same position, **When** the
   homepage loads, **Then** the earlier-created section renders
   first.
4. **Given** a custom section between two built-in sections, **When**
   the visitor uses the arrows, **Then** the arrows move through
   the custom section in order.

### User Story 3: Visitor reads a custom section in their language (Priority: P1)

The admin writes the header, the subtitle, and the body in Italian
and in English. The visitor reads the section in the language they
selected on the site.

**Why this priority**: The site is bilingual, and the owner
confirmed that the section texts are per language.

**Independent Test**: Write the section in both languages, switch
the site language, and load the homepage. The section shows the
texts of the selected language.

**Acceptance Scenarios**:

1. **Given** a section written in Italian and English, **When** the
   visitor selects Italian, **Then** the section shows the Italian
   texts.
2. **Given** a section written in Italian and English, **When** the
   visitor selects English, **Then** the section shows the English
   texts.
3. **Given** a section with Italian texts only, **When** the visitor
   selects English, **Then** the section does not render in
   English [ASSUMPTION A1 — to confirm at ratification].

### User Story 4: Owners build their own honeymoon section (Priority: P1)

The owners of the second live instance add a custom section for
their honeymoon, at the honeymoon place in the order. They add no
honeymoon places, so the built-in honeymoon section does not
render. The homepage shows only their custom section there.

**Why this priority**: This is the use case the owner named, and
it proves that a custom section never replaces a built-in one:
the built-in section stays governed by its own rule.

**Independent Test**: Add a custom honeymoon section with no
honeymoon places in the database, and load the homepage. Only the
custom section renders in that place.

**Acceptance Scenarios**:

1. **Given** a custom section at the honeymoon place and no
   honeymoon places in the database, **When** the homepage loads,
   **Then** only the custom section renders there.
2. **Given** a custom section at the honeymoon place and honeymoon
   places in the database, **When** the homepage loads, **Then**
   both the custom section and the built-in honeymoon section
   render, each in its own place.

### Edge Cases

- What happens when a section has a header but neither a body nor
  an image? The section is invalid and the save is rejected
  (see A2).
- What happens when a section has a body but no header? The section
  is invalid and the save is rejected (see A2).
- What happens when a section has an empty subtitle? The subtitle
  is optional. The section renders without it.
- What happens when a section has no background image? The section
  renders on the page background.
- What happens when a section has a background image for one
  viewport only? The pair is optional per viewport: a missing
  mobile image falls back to the desktop image, and a missing
  desktop image falls back to the mobile image [ASSUMPTION A3 —
  to confirm at ratification].
- What happens when the admin deletes a custom section? The section
  disappears from the homepage, and its position closes up: no
  empty slot stays behind.

## Requirements

### Functional Requirements

- **FR-001**: The admin MUST add a custom section to the homepage.
- **FR-002**: A custom section MUST have a header.
- **FR-003**: A custom section MUST have a body text or a
  background image, or both.
- **FR-004**: A custom section MAY have a subtitle. An empty
  subtitle MUST NOT block the save.
- **FR-005**: A custom section MAY have no custom background.
- **FR-006**: A custom section background MUST be a desktop/mobile
  image pair. The pair MUST render full-bleed, as the hero
  background does.
- **FR-007**: The header, the subtitle, and the body MUST exist per
  language, Italian and English.
- **FR-008**: A custom section MUST have a position. Positions MUST
  count all homepage sections. The landing section MUST be
  position 1.
- **FR-009**: The homepage MUST render custom sections in position
  order, merged with the built-in sections in their current
  relative order.
- **FR-010**: A custom section MUST NOT replace a built-in section.
  Built-in sections MUST render by their own rules.
- **FR-011**: The up/down arrows MUST navigate through custom
  sections in rendered order.
- **FR-012**: A custom section MUST render in a language only when
  that language has a header and (a body or a background image)
  [ASSUMPTION A1 — to confirm at ratification].
- **FR-013**: The admin MUST delete a custom section, and the
  deletion MUST close up the order with no empty slot left.

### Key Entities

- **Custom section**: one section added by the admin. It holds a
  background image pair (or none), a header, a subtitle, and a body
  text per language, and a position.
- **Position**: a 1-based number across all homepage sections.
  The landing section is position 1.
- **Background image pair**: a desktop image and a mobile image,
  rendered full-bleed as the hero background.

## Success Criteria

### Measurable Outcomes

- **SC-001**: The admin adds a section with a header, a body, and
  a background pair, and the homepage shows it full-bleed at its
  position.
- **SC-002**: The admin adds a section with a header and an image
  but no body, and the homepage shows it.
- **SC-003**: The admin adds a section with a header and a body
  but no image, and the homepage shows it on the page background.
- **SC-004**: The admin places a section at position 2, and it
  renders between the landing section and the next section, with
  the arrows moving through it.
- **SC-005**: A section written in both languages shows the
  Italian texts in Italian and the English texts in English.
- **SC-006**: A custom honeymoon section with no honeymoon places
  renders alone in the honeymoon place.
- **SC-007**: The change ships with the per-package 90% coverage
  gate green and the CI pipeline green on both live instances.

## Assumptions

- **A1 — to confirm at ratification**: A section renders in a
  language only when that language has a header and (a body or a
  background image). A section written in Italian only renders in
  Italian and is skipped in English.
- **A2 — to confirm at ratification**: Validation runs in
  `SaveSettings`, as for the other collections: a section with no
  header, or with neither a body nor a background image in any
  language, rejects the whole settings save. Per-row validation
  in the dashboard UI is not part of this cycle.
- **A3 — to confirm at ratification**: The image pair is optional
  per viewport. A missing mobile image falls back to the desktop
  image, and a missing desktop image falls back to the mobile
  image, as the hero background does.
- **A4 — to confirm at ratification**: Equal positions order by
  creation order, with the earlier-created section first.
  Positions past the last section render last. Gaps in the
  numbering collapse: no empty slot stays behind.

## Out of Scope

- Any change to the hero text anchor of cycle 001.
- Any change to the gifts section, in either mode, of cycle 001.
- Any change to the built-in honeymoon section or to its rendering
  rule. It renders only when honeymoon places exist, as today.
- A colour background for custom sections. The background is an
  image or nothing.
- A second image per section besides the background pair.
- Hiding or disabling a built-in section from the dashboard.
  Custom sections never replace built-in sections.
