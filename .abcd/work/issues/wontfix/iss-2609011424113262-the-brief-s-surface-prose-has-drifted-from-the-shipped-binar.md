---
schema_version: 1
id: "iss-2609011424113262"
slug: "the-brief-s-surface-prose-has-drifted-from-the-shipped-binar"
severity: "major"
category: "drift"
source: "agent-finding"
found_during: "the v0.7.0 cut, iss35 full-tier crosscheck"
found_at: ".abcd/development/brief/04-surfaces/"
origin: researcher-authored
production_mode: hand-written
wontfix_reason: "duplicate of iss-2609011424033603: the same twenty-one pre-existing findings captured twice from one receipt; this record carries the fuller body, and the resolution with fixing-commit provenance is recorded on the earlier id"
---

The brief's surface prose has drifted from the shipped binary in twenty-one
places the machine-checked layer cannot see. These are the crosscheck findings
that predate v0.7.0, kept apart from the fifteen this release introduced
([[iss-2609011424116881]]). All are recorded with evidence in the v0.7.0 receipt
at
`.abcd/work/reviews/4b4076a10f89d4d02da359274dad8994e30cae0e/iss35-brief-surface-crosscheck.json`,
dispositioned `deferred-pre-existing`.

## The shape of it

The machine-checked layer is clean. `.abcd/development/release/surface.json`
against the live binary is byte-exact across 82 command nodes and 109 flags with
zero drift, and every `## Sub-verbs` table in every chapter verified correct.
Every finding here is in prose that `surface_coverage` does not scan, which is
exactly the gap `04-surfaces/README.md:33` names when it says prose is not
machine-scanned and review must cover it.

That is the finding behind the findings: the parts of the brief a rule can check
are right, and the parts only a reader can check have rotted. Counting them is
less useful than noticing they cluster in flag enumerations, criteria lists,
templates and cross-surface counts, none of which any gate reads.

## Three that actively mislead

Following the brief as written produces something the binary refuses.

- `04-surfaces/06-capture.md:33,35,115,116` documents `capture promote` and
  `capture resolve` without `--grounds`, which is REQUIRED on both. The chapter's
  own acceptance criteria give invocations that exit non-zero.
- `04-surfaces/05-intent.md:216-269` gives a canonical intent template with no
  `## Grounds` section. The shipped `intent ready` gate requires one, so an intent
  authored to the brief's own template cannot pass it.
- `04-surfaces/05-intent.md:3,200` names six readiness criteria; seven ship, and
  the missing one is `grounds`, which is the check that actually fails in practice.

Note the pattern: all three are the same `--grounds` and `## Grounds` work landing
in the binary without landing in the chapters that describe it.

## The rest

- `05-intent.md:3,200` calls `intent ready` read-only; `--grounds` records the
  conjecture, so it writes.
- `06-capture.md:28` omits `--production-mode` (ships on create, promote, resolve
  and wontfix); `:35` omits `--shipped-in`; `:51-81` asserts its frontmatter block
  mirrors the schema exactly while omitting `origin` and `production_mode`.
- `01-ahoy.md:288-296` omits `--attribution` and the `prepare-commit-msg` hook it
  installs; `:266-275` omits that same shipped default template; `:36-39,433-440`
  omits the provenance record `uninstall` removes and its `--bin-dir` flag.
- `02-disembark.md:28-34` presents sub-verb arguments as the shipped surface and
  names no flags; `--include-ignored` ships on `pack`, `probe` and `plan`.
- `04-launch.md:22` says the plain-text `--dry-run` render prints only five
  things; it also prints a `receipts:` line.
- `19-identity.md:35` quotes a Tagline that does not match abcd's own canonical
  identity block. Title and Pitch match byte-for-byte; the Tagline does not. This
  is the drift class `iss-143` and the `identity-positioning` rule exist to catch,
  and the brief chapter is not a registered surface in `.abcd/positioning.json`,
  so nothing gates it.
- `22-site.md:6-7,54,97` states the bare form is strictly read-only and `build` is
  the render; `site check` renders when the output directory holds no `index.html`,
  a write path the chapter never records. The plugin page documents it; the
  chapter does not.
- `05-internals/03-configuration.md:361` names a fictional `ahoy destroy`
  sub-verb.
- `06-delivery/01-build-sequence.md:29` names `skills/` as a shipped markdown
  surface directory; it does not exist, as three other locations correctly say.
- `04-surfaces/README.md:12,16,17` index rows omit `capture disposition`,
  the `docs cite` sub-tree, and `history drain`/`history staged`. All three are
  correct in their own chapters; only the index is stale.
- `17-guard.md:141-143` cites `iss-315` as tracking an unimplemented design
  target; `iss-315` is resolved and explicitly scopes that work out, so no open
  issue tracks it. `internal/core/guard/payload.go:191` makes the same citation.
- `14-ingest.md:10-11` says no Go verb backs ingest. True at top level, but
  `reading ingest` and `memory ingest` now exist as binary sub-verbs, so the
  phrase is falsifiable as written.

## Grounds

- deferred: not fixed in the v0.7.0 cut, because none of it is caused by or
  blocking this release and the cut was already two rounds deep in gate
  corrections; deferring was the maintainer's recorded decision at PROMOTE

- declined: duplicate of iss-2609011424033603: the same twenty-one pre-existing findings captured twice from one receipt; this record carries the fuller body, and the resolution with fixing-commit provenance is recorded on the earlier id

## Worth deciding, not just fixing

Three of these are index or enumeration rows that are correct in one place and
stale in another, and two chapters maintain the same bare-status-board list
independently. Fixing the instances leaves the duplication that produced them.
Whether any of these enumerations should be derived rather than restated is the
question underneath the list, and it is the same question
[[iss-2609011423385217]] raises about the release-gate manifest's pinned content
list.
