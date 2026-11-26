# Adopt GitHub Spec Kit on Foedus

## Status

Accepted on 2026-11-26.

## Context and Problem Statement

One agent executes all the work on this repository. Decisions live in
chat threads and in the agent memory, outside the repository. A new
session cannot resume the work from the repository files alone: the
session must reconstruct the state from conversation history. The
owner wants the decision-to-verification cycle made persistent:
research, human decision, persistent document, specification, plan
and tasks, implementation, verification.

The repository holds no method artifacts today. The baseline check
of 2026-11-26 on `origin/master` found no `.specify/`, no `specs/`,
no constitution, and no decisions register. The same cycle already
runs on the pilot repository, where the owner ratified the
constitution and closed the first cycles from repository files alone.

## Considered Options

1. Keep the current method. Decisions stay in chat and agent memory.
2. Adopt GitHub Spec Kit with a custom layout. Decisions go to a
   `docs/` directory, against the layout of the framework.
3. Adopt GitHub Spec Kit with the framework layout. Decisions go to a
   register under `.specify/memory/decisions/`, in MADR format.

## Decision Outcome

Chosen option: option 3.

The adoption uses GitHub Spec Kit 1.2.2, pinned, with the `generic`
integration and the command files under `.specify/commands/`. The
repository commits the Spec Kit artifacts, per the framework
guidelines; the machine-local files stay excluded through the
`.gitignore` that the CLI manages. The constitution derives from the
unciae golden standard. A completed feature specification stays as
an immutable historical record. The first bounded cycle is
`specs/001-hero-anchor-gifts-coordinate`: the text anchor of the
first homepage section and the coordinate mode of the gifts section.

Justification: option 1 leaves the problem unchanged. Option 2 fights
the framework at every upgrade for no functional gain. Option 3
keeps one authoritative location for method artifacts and one for
decisions, and it repeats the layout the pilot already proved.

## Consequences

- Positive: a new session resumes from repository files alone.
- Positive: every significant decision gains an authority, a status,
  and a supersession chain.
- Negative: the project carries method artifacts that must stay
  current, or the register decays into noise.
- Neutral: the `generic` integration gives command files as
  documents. The agent follows the documents; the harness gains no
  native commands. The value is the artifact discipline, not tooling.
- Neutral: the adoption itself changes no application behaviour. The
  first behaviour change lands in the first cycle, not in the
  adoption.

## Acceptance

The adoption is complete when a fresh session resumes the work from
the repository files alone, completes a scoped change, and shows the
verifications of the change. That check is the continuity test.
