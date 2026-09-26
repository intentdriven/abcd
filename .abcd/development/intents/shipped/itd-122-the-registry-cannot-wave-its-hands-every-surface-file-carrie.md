---
id: itd-122
spec_id: spc-27
kind: standalone
suggested_kind: null
reclassification_history: []
builds_on: []
severity: major
impact: additive
slug: the-registry-cannot-wave-its-hands-every-surface-file-carrie
---

# The Registry Cannot Wave Its Hands

## Press Release

The registry can no longer wave its hands. Every surface file under
`04-surfaces/` carries a sub-verb table — which bucket each verb is
(lint / review / audit / gate) and whether it exists — and `surface_coverage`
now checks every row against the binary's own command tree, both ways: a
sub-verb that ships without a row fails the build, and a row claiming an
unregistered sub-verb fails the build. "shipped means shipped, at every grain —
when a rename lands, the gate proves the migration complete instead of me
grepping and hoping," says Bob, staff engineer.

## Why This Matters

`surface_coverage` is blind inside a row, so `shipped` means "the top-level
verb exists" and every unbuilt sub-verb hides behind it — six of twenty rows
qualify themselves in prose the lint does not read, and documents outside the
registry then cite those sub-verbs as live (refines iss-246, its "two defects,
one fix"). This is the detector adr-40 needs (decision 6), and it is the
ordering keystone of the process-coherence plan: the check lands **armed, with
rows reflecting current names, before any rename** — so the renames are proved
complete by a gate rather than asserted complete by an agent. Per the
2026-08-16 planning rulings, the `Status` enum stays two-valued at both grains;
sub-verb rows are what carry the granularity.

## Scope Conditions

None stated.

## Acceptance Criteria

> _BDD format, per the itd-1 discipline. Walked and confirmed by the
> maintainer, 2026-08-16._

- **Given** a surface file under `04-surfaces/`, **then** it carries a
  sub-verb table with two facts per verb: bucket (`lint` / `review` / `audit`
  / `gate`, or `—` for a non-assessment verb) and existence (`shipped` /
  `staged`).
- **Given** the extended `surface_coverage`, **then** a `shipped` row must
  match a registered sub-command in the committed command-tree snapshot
  (`.abcd/development/release/surface.json`, itself drift-checked against the
  cobra tree) **and** every registered sub-command of that surface must have a
  row — both directions are lint failures.
- **Given** a `staged` row, **then** no registered sub-command may back it.
- **Given** a host-delegated surface (no Go verb: `consult`, `ingest`,
  `prepare-this-repo`) or an operator-internal verb with no surface file
  (`spec`, `rules`, `hook`, `completion`), **then** the exemption is explicit
  rule config — mirroring today's `bare_command` mechanism — never a silent
  skip: host-delegated rows are exempt from the cobra check only; the
  operator-internal list is enumerated.
- **Given** the population pass, **then** all twenty surface files gain their
  tables in the **same change** the extended check arms — nothing ships
  unbucketed, and rows reflect current (pre-rename) names.
- **Given** the bucketings adr-40 calls arguable, **then** the maintainer's
  pre-rulings bind: `identity` is an **audit** (rendered surfaces vs the
  recorded canonical block), the launch changelog guardrail is a **gate**, and
  `guard check` is a **gate**. Any other genuinely ambiguous bucket is a STOP
  for an unattended run, never a guess into the closed list.
- **Given** the bucket enum, **then** it is registered under Reserved
  vocabulary in `02-constraints/04-naming.md` (closed list, PR-to-extend).
- **Given** the `Status` enum, **then** it stays two-valued (`shipped` /
  `staged`) at both grains — no `partial`.
- **Given** the sweep, **then** the `04-surfaces/README.md` prose describing
  the check, and `CHANGELOG.md`, reflect the extended rule.

## SOTA

Registry-vs-implementation cross-checks are standard drift tooling (OpenAPI
spec-vs-handler checkers, terraform schema drift, this repo's own
`surface_coverage` and `index_drift` rules); nothing importable — the registry
format and command tree are native. **Chosen path: bespoke**, extending the
existing `checkSurfaceCoverage` lint against the existing committed
`surface.json` snapshot (no import-cycle: both inputs are committed
artefacts). No new dependency.

## Open Questions

_None gating. The three arguable bucketings were pre-ruled at the grill,
2026-08-16, and are binding on the build._

## Audit Notes

<!-- abcd-review: INGESTED receipt=rcp-4bf2cc7ba486 -->
Fidelity review — receipt rcp-4bf2cc7ba486 (verifier intent-auditor claude-fable-5-1).

Provenance: intent-auditor@claude-fable-5-1 · rubric_hash sha256:effa65b3e9e88ff29433b443ec2be159522a8b0b71cf1434526514aa61edb13e · prompt_hash sha256:6ffba6cc5323cd2e8bee44c02c9b374fdb45516d950ef1813794979997cd1741
Input attestations: diff:tree at 4c09b5c749de3dc3d1a1e0ab85d9dcd58ffcb4d5 (main lineage, itd-122 shipped)@-;

Acceptance rollup: MET 9 · MET_WITH_CONCERNS 0 · NOT_MET 0 · INCONCLUSIVE 0

Per-criterion verdicts:
- ac-1 — MET: all 28 files under 04-surfaces/ carry a ## Sub-verbs table with Verb | Bucket | Status rows; the parser anchors on the exact heading and TestSubVerbMissingTableFailsOnEverySurfaceFile makes the table mandatory
  evidence: internal/core/lint/subverbs.go:43 — "var subVerbHeadingRe = regexp.MustCompile(`^##\s+Sub-verbs\s*$`)"
  evidence: .abcd/development/brief/04-surfaces/04-launch.md:35 — "| Verb | Bucket | Status |"
  evidence: internal/core/lint/subverbs_test.go:158 — "func TestSubVerbMissingTableFailsOnEverySurfaceFile(t *testing.T) {"
- ac-2 — MET: checkSubVerbCoverage loads the committed surface.json, fails a shipped row with no registered sub-command and a registered sub-command with no row; TestSubVerbBothDirections covers both, and TestSurfaceSnapshotMatchesCommittedBaseline holds the snapshot to the cobra tree
  evidence: internal/core/lint/subverbs.go:175 — "is marked shipped but the command tree registers no \`abcd"
  evidence: internal/core/lint/subverbs.go:197 — "has no row in the '## Sub-verbs' table"
  evidence: .abcd/record-lint.json:343 — ""snapshot": ".abcd/development/release/surface.json""
  evidence: internal/core/lint/subverbs_test.go:105 — "func TestSubVerbBothDirections(t *testing.T) {"
  evidence: internal/surface/cli/surface_test.go:200 — "func TestSurfaceSnapshotMatchesCommittedBaseline(t *testing.T) {"
- ac-3 — MET: a staged row backed by a registered sub-command is a finding telling the author to mark it shipped; TestSubVerbStagedButRegisteredFails asserts it
  evidence: internal/core/lint/subverbs.go:182 — "is marked staged but \`abcd"
  evidence: internal/core/lint/subverbs_test.go:135 — "func TestSubVerbStagedButRegisteredFails(t *testing.T) {"
- ac-4 — MET: record-lint.json enumerates host_delegated (consult, ingest, prepare-this-repo) and operator_internal (spec, rules, hook, …) beside bare_command; host-delegated verbs skip only the cobra comparison while their tables are still format-checked, and operator-internal verbs are excluded from the reverse sweep alone; TestSubVerbHostDelegatedFormatOnly and TestSubVerbExclusions cover both
  evidence: .abcd/record-lint.json:344 — ""host_delegated": ["
  evidence: .abcd/record-lint.json:349 — ""operator_internal": ["
  evidence: internal/core/lint/subverbs.go:114 — "exemptFromCobra := verb == cfg.BareCommand || hostDelegated[verb]"
  evidence: internal/core/lint/subverbs.go:212 — "if len(subsByVerb[v]) == 0 || seenVerbs[v] || v == cfg.BareCommand || operatorInternal[v] {"
  evidence: internal/core/lint/subverbs_test.go:226 — "func TestSubVerbHostDelegatedFormatOnly(t *testing.T) {"
- ac-5 — MET: at BASE every one of the 28 surface files carries the table (the registry grew past the twenty the intent counted) and the rule is armed with a snapshot in record-lint.json, so nothing ships unbucketed; the lint package is green at BASE
  evidence: .abcd/record-lint.json:343 — ""snapshot": ".abcd/development/release/surface.json""
  evidence: .abcd/development/brief/04-surfaces/README.md:61 — "every surface file in this directory carries a `## Sub-verbs` table"
  evidence: internal/core/lint/subverbs_test.go:318 — "func TestSubVerbUnarmedIsInert(t *testing.T) {"
- ac-6 — MET: the pre-ruled bucketings hold in the tables: identity render is an audit, launch ship (the changelog guardrail's verb) is a gate, guard check is a gate; an unknown bucket is a finding against the closed list, so no guess enters it
  evidence: .abcd/development/brief/04-surfaces/19-identity.md:33 — "| `render` | audit | shipped |"
  evidence: .abcd/development/brief/04-surfaces/04-launch.md:40 — "| `ship` | gate | shipped |"
  evidence: .abcd/development/brief/04-surfaces/17-guard.md:29 — "| `check` | gate | shipped |"
  evidence: internal/core/lint/subverbs.go:164 — "if !subVerbBuckets[r.bucket] {"
- ac-7 — MET: the bucket enum is a row under Reserved vocabulary in 02-constraints/04-naming.md, stated as a closed list PR-to-extend, and the code's subVerbBuckets is the same closed set with TestSubVerbVocabularyEnforced
  evidence: .abcd/development/brief/02-constraints/04-naming.md:172 — "| `bucket` ∈ `{lint, review, audit, gate}` |"
  evidence: internal/core/lint/subverbs.go:36 — "var subVerbBuckets = map[string]bool{"lint": true, "review": true, "audit": true, "gate": true, "—": true}"
  evidence: internal/core/lint/subverbs_test.go:207 — "func TestSubVerbVocabularyEnforced(t *testing.T) {"
- ac-8 — MET: the row status accepts shipped or staged only and any other token is a finding; the README records that the surface-grain Status stays two-valued with no partial
  evidence: internal/core/lint/subverbs.go:188 — "has unknown status '" + r.status + "' (want shipped|staged)"
  evidence: .abcd/development/brief/04-surfaces/README.md:75 — "The surface-grain `Status` enum stays two-valued: there is no `partial`"
- ac-9 — MET: the 04-surfaces/README.md prose describes the sub-verb pass, both directions and the explicit exemptions, and CHANGELOG.md carries the itd-122 line in a shipped section
  evidence: .abcd/development/brief/04-surfaces/README.md:67 — "The rule checks each table against the committed command-tree snapshot in both directions"
  evidence: CHANGELOG.md:35 — "**Every surface chapter carries a sub-verb table**"

Gap audit:
- honoured:
  - surface_coverage checks every sub-verb row against the binary's own command tree both ways
    evidence: internal/core/lint/subverbs.go:61 — "func checkSubVerbCoverage(repoRoot string, cfg RuleConfig) ([]Finding, error) {"
  - an armed pass with a missing snapshot is a loud finding, never a silent skip
    evidence: internal/core/lint/subverbs_test.go:299 — "func TestSubVerbMissingSnapshotFailsLoudly(t *testing.T) {"
  - exemptions are explicit config mirroring bare_command
    evidence: .abcd/record-lint.json:340 — ""bare_command": "abcd""
- diverged:
  - the bucket cell is checked for closed-list membership only; the snapshot carries no bucket, so a legal but wrong bucket passes and the cell stays a review-grain claim (stated on every surface file's table preamble)
    evidence: .abcd/development/brief/04-surfaces/04-launch.md:31 — "The bucket cell is checked for membership of the closed adr-40 > vocabulary only"
- missing: (none)
