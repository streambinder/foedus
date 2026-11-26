# Foedus Constitution

**Status**: Ratified by the repository owner on 2026-11-26.

The canonical standard for this repository is the unciae golden standard
(`streambinder/unciae`, file `AGENTS.md`). This constitution restates the
rules of that standard that bind this repository, in the form that the
Spec Kit workflow checks. When a rule here and the canonical standard
disagree, the canonical standard wins and this document gets amended.

## Core Principles

### I. One Repository, One Deployable Unit

The repository builds one application and one container image. The root
stays minimal: a new root entry needs a justification. Configuration
files live in their conventional locations, never ad hoc.
(unciae §1, §2)

### II. CI Is the Source of Truth

A change is done when the continuous integration pipeline is green.
Local checks mirror the pipeline; local checks never replace the
pipeline. Do not push on red. (unciae §1, §17, §18)

### III. One Linting System

super-linter is the linter. Its configuration files live only under
`.github/linters/`. No linter configuration sits at the repository
root, not even as a symlink. Format code with the canonical formatter
of the language before every push: for Go, `gofumpt`. Run `templ
generate` before the tests, as the pipeline does. (unciae §3)

### IV. Tests and the Coverage Gate

Unit tests live next to the code that the tests cover, in the layout
that the toolchain dictates. Integration tests live separately. CI
runs the tests on every push. A failing test blocks the merge. CI
measures coverage on every push. The minimum line coverage is 90%
per package. Coverage below 90% in any package blocks the merge.
Non-test code never changes to satisfy a test; the test changes
instead. (unciae §15)

### V. Minimal Owned Surface

The repository owns its product logic and nothing else. For every other
functionality, first simplify the problem away, then delegate to a
specialized library with an established reputation. Pin exact
dependency versions where possible. (unciae §16, §16.1)

### VI. Commits and Pull Requests

Commits follow Conventional Commits: one logical change per commit,
signed, with no artificial-intelligence attribution trailers. Commit
messages and pull request descriptions never name a verification
subject; describe the class of the subject and the outcome instead.
The owner merges. (unciae §7, §19, §24)

### VII. Containers and Releases

Build for linux/arm64 only. Pin every action and base image. Publish
never runs on a pull request. (unciae §5, §10, §11)

### VIII. Writing and Consistency

Technical writing follows ASD-STE100: short sentences, one term for
one concept, active voice. A change follows the conventions of the
file that the change touches. Do not reformat parts that the change
does not touch. (unciae §22, §23)

## Project Constraints

These constraints are decisions of the repository owner for this
project. Each constraint stands until the owner replaces it with a
recorded decision.

### Stack

Go 1.26, Fiber v2, templ, SQLite (modernc.org driver). Views render
server-side. The dashboard uses Datastar. Client code is vanilla
JavaScript. Maps use Leaflet.

### Maps

CARTO basemaps stay. No Protomaps. The CARTO key is optional and
comes from the `CARTO_API_KEY` environment variable. This constraint
records the owner decision of 2026-11-16.

### Production Care

The application serves real couples every day, in two live instances
on the same host, each with its own SQLite file. Before every deploy,
take a database backup of each instance and verify its snapshot.
After every deploy, verify that both instances answer and that the
real live content renders. Never write on production data to test.
Never change production without a local quality check first.

### Data Compatibility

Each production SQLite file is a contract. The application must open
the existing file without data loss. A schema change needs a recorded
decision and a migration path before implementation.

### Database Migrations

Code-side migrations create only the canonical schema: the schema a
clean database needs, built with idempotent `CREATE TABLE` blocks
that run on every start. Code-side migrations never alter an existing
table. Code-side migrations never rename a table or a column.
Code-side migrations never rewrite stored values. Code-side
migrations never insert seed data. A rename, an alter, and a value
rewrite are incremental migrations. An incremental migration never
runs from code. An incremental migration runs by hand, once, against
each live database. The agent proposes the commands before the
rollout: the backup command, the migration commands, and the
verification commands. The owner executes the commands or approves
the execution. A backup precedes every incremental migration. The
verification proves the new state and proves that no data was lost.
(unciae §26)

### Specification Cycle

Every feature change and every change to the domain model follows the
Spec Kit cycle: specify, plan, tasks, implement, verify, converge.
The specification describes the next change, never the whole system
retroactively. Completed feature directories stay as immutable
historical records. A significant decision goes to the decision
register under `.specify/memory/decisions/`, in MADR format, one file
per decision. A decision is a draft until the owner ratifies it. An
accepted decision is reused; a superseded decision stays in the
register and names its successor.

### Hypothesis First

Every feature request opens with an investigation of the hypothesis
on real data. The investigation verifies that the problem exists.
The investigation verifies that the assumed cause holds. The
investigation looks for a simpler alternative. The investigation has
three outcomes: specify the feature as proposed, specify a different
solution, or abandon the feature. The decision register records the
outcome of the investigation and the evidence of the investigation.
The specification is a draft until the owner ratifies it. The plan
step starts after the ratification of the specification.

### Recorded Drift

The templ library is at 0.3.1118 while the templ generator installed
by CI and the Dockerfile is at 0.3.1100. This drift is recorded
here as a fact. No cycle fixes it silently. A cycle that touches
templ records the alignment as a decision first.

## Working Agreements

This section restates the operative form of the principles, so the
repository stays self-contained for the points that the golden
standard states in full. A new session applies these agreements
without further sources.

### Branches and commits

- A feature branch carries the name `feat/<short-name>`.
- A commit follows Conventional Commits. The subject holds at most
  72 characters, uses the imperative, carries no terminal period,
  and contains none of the characters `()[]"'`/,:;`.
- A commit body holds one sentence of at most 100 characters and
  ends with a period. The body states why, not what.
- One logical change makes one commit.
- A pull request holds one commit or more. One commit stays
  the preference.
- Each file a pull request changes appears in exactly one
  commit of the pull request.
- A specification pull request holds one commit.
- An implementation pull request holds one commit per plan
  phase. A small change holds one commit.
- A fix for a red feature branch amends the failing commit. The
  push uses force with an explicit lease that names the remote
  commit. Nobody amends or force-pushes the default branch.

### Text and formatting

- Prettier formats every authored Markdown file. The author runs
  prettier, in the major version the lint image bundles, on each
  authored file before the push.
- Codespell checks every text file in CI. Authored text passes
  codespell before the push.
- `gofumpt` formats Go code before every push. `templ generate`
  runs before the tests.

### Tests

- Tests live next to the code they cover: handler tests beside the
  handlers, template tests beside the templates. New tests follow
  the test pattern of the package they touch.
- The per-package 90% line-coverage gate binds every package. A
  package that falls below 90% blocks the merge.

## Governance

The owner ratifies this constitution and every amendment. The agent
drafts amendments and records the ratified text. Every pull request
review checks compliance with this constitution. A rule that proves
wrong is amended through a recorded decision; the rule is never
silently ignored.

**Version**: 1.0.0 | **Ratified**: 2026-11-26
