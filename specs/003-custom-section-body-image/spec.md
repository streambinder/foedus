# 003 — Body image for custom sections

Status: Ratified 2026-11-27
Date: 2026-11-27
Amends: specs/002-custom-sections/spec.md (ratified 2026-11-26)

## Problem

Custom sections (cycle 002) accept only background images (desktop/mobile pair).
The dashboard offers no way to add a photo inside the section body, and the
background fields are labelled only "Desktop / horizontal" and
"Mobile / vertical", which does not say they are background images.

## Decisions

- D1. The background image fields keep their current behaviour (desktop/mobile
  pair, full-bleed, fallback to the other viewport when one is missing), but
  their dashboard labels say explicitly that they are background images:
  "Background — Desktop / horizontal" and "Background — Mobile / vertical"
  (English; Italian equivalents in the dashboard).
- D2. Each custom section gains an optional body image: a single image, with no
  desktop/mobile distinction. It is stored like every other image in the site
  (a reference to the `media` table, served at `/media/:id`), using the same
  upload/resolve/orphan-cleanup flow as the background images.
- D3. A body image counts as body content for validation: a section is valid
  with a header plus body text, a background image, or a body image.
- D4. The body image renders inside the section content area, below the body
  text, centred, at content width (not full-bleed). On mobile it scales to the
  viewport width.

## Assumptions (to confirm at ratification)

- A1. Validation: a body image alone (header + body image, no body text, no
  background) is a valid section.
- A2. Placement: the body image renders below the body text (D4). If the owner
  prefers it above the text, this changes at implementation.
- A3. Migration: `custom_sections` already exists on the live databases (cycle
  002 deployed 2026-11-26), so the new nullable column
  `body_media_id INTEGER REFERENCES media(id)` is added by a manual ALTER on
  both live databases (backup + ALTER + verify, proposed by Tiro, executed or
  approved by Davide), exactly as cycle 001 did for `hero_text_anchor` and
  `gifts_mode`. The code-side `CREATE TABLE` is updated to include the column
  for fresh databases. No data loss: the column is nullable with no default.

## Out of scope

- Multiple body images per section.
- Desktop/mobile variants of the body image.
- Captions or alt text editing for the body image.
- Any change to the background image behaviour beyond the labels (D1).
- Any change to built-in sections.

## Verification

- `templ generate`, gofumpt, full `go test -race` with every package ≥90%.
- Dashboard: background fields labelled as background; body image field
  present, single, uploads and previews like the other image fields.
- Rendering: a section with only header + body image renders; a section with
  body text + body image renders the image below the text; removing a body
  image cleans up the orphaned media row.
- Live databases open unchanged after the manual ALTER; both instances answer
  200 after deploy.
