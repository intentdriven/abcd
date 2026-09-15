---
id: itd-182
slug: the-record-s-discipline-failures-are-themselves-recorded-a-l
spec_id: spc-60
kind: standalone
suggested_kind: standalone
reclassification_history: []
builds_on: []
severity: minor
impact: additive
---

# The record's discipline failures are themselves recorded — a lapse is a capture category, timestamped at the lapse

## Press Release

> **When the recording discipline is suspended, deferred, or evaded, that
> is captured too.** A `lapse` entry carries the point in the process at
> which the discipline gave way and a timestamp at the lapse rather than at
> write-up. The log is not merely a disclosure obligation: the working
> claim is that recording at the point of commitment prevents retrospective
> reconstruction, and the lapse log is the evidence bearing on that claim.

## What's In Scope

- The `lapse` value in capture's validated category list.
- The first three entries, written at the outset rather than discovered:
  the pre-tooling window (which entries were hand-authored before their
  surfaces existed); anticipation (those populating the record know what
  the readings will look for, and the instrument is specified alongside the
  record it will read); any commitment made outside the tooling during the
  build.

## Acceptance Criteria

- **Given** a lapse, **when** it is captured, **then** the entry carries
  the category, the point in the process at which the discipline was
  suspended, deferred or evaded, and a timestamp at the lapse rather than
  at write-up.

## Audit Notes

<!-- abcd-review: INGESTED receipt=rcp-a98036202299 -->
Fidelity review — receipt rcp-a98036202299 (verifier abcd:intent-auditor claude-fable-5).

Provenance: abcd:intent-auditor@claude-fable-5 · rubric_hash sha256:f3bba86a84b329fcbfbd3df64ec5d77d382dda9b4b1a61baf887e316bc41f38e · prompt_hash sha256:f5d0d1ff49565622f1179588454aef242aac986870b562e8ebadefaf028165c6
Input attestations: diff:543aaf19...d4b288a0 (build/itd-182 merged into experiment/cold-reading)@sha256:2b84b6a57ae67782d0f3a56df2c0ec3037017f4e31c4c23938a697a54c700259;

Acceptance rollup: MET 0 · MET_WITH_CONCERNS 1 · NOT_MET 0 · INCONCLUSIVE 0

Per-criterion verdicts:
- ac-1 — MET_WITH_CONCERNS: All three elements are realised by mechanism and pinned by passing tests: category via the lapse enum value, the point in the process via the already-required found_during, and the lapse-time via the new lapsed_at property that the reader and the committed-ledger gate both require for category lapse and that the --lapsed-at flag never defaults. Concern: the shipped lapse entries were back-stamped by commit 267e2c74 with midnight-UTC day-granularity instants (2026-08-28T00:00:00Z) after the records existed, so the content honours 'at the lapse' only at day resolution and by later stamping; the gate can enforce presence and shape of the instant, not that it was recorded at the lapse.
  evidence: internal/core/issueschema/issueschema.go:89 — "\"lapse\","
  evidence: internal/core/issueschema/issueschema.go:114 — "func LapsedAtRequired(category string) bool {"
  evidence: internal/core/issueschema/issueschema.go:123 — "func ValidLapsedAt(v string) bool {"
  evidence: internal/core/capture/validate.go:72 — "if strings.TrimSpace(fm[\"found_during\"].(string)) == \"\" {"
  evidence: internal/core/capture/validate.go:117 — "if issueschema.LapsedAtRequired(fm[\"category\"].(string)) && lapsedAt == \"\" {"
  evidence: internal/core/capture/validate.go:121 — "if lapsedAt != \"\" && !issueschema.ValidLapsedAt(lapsedAt) {"
  evidence: internal/surface/cli/cli.go:2402 — "if issueschema.LapsedAtRequired(string(req.Category)) && strings.TrimSpace(req.LapsedAt) == \"\" {"
  evidence: internal/surface/cli/cli.go:2432 — "captureCmd.Flags().StringVar(&lapsedAt, \"lapsed-at\", \"\", \"RFC 3339 instant a discipline gave way (the lapse, not the write-up)\")"
  evidence: internal/core/lint/schema.go:580 — "issueschema.LapsedAtRequired(issueScalar(f.value)) && lapsedAt == \"\" {"
  evidence: internal/core/capture/parse_test.go:183 — "func TestValidateStrictLapseRequiresLapsedAt(t *testing.T) {"
  evidence: internal/core/capture/parse_test.go:239 — "func TestValidateStrictLapseCarriesFoundDuring(t *testing.T) {"
  evidence: internal/surface/cli/capture_surface_test.go:513 — "func TestCaptureLapsedAtHasNoDefault(t *testing.T) {"
  evidence: .abcd/work/issues/open/iss-2608300002547145-lapse-pre-tooling-window.md:6 — "category: \"lapse\""
  evidence: .abcd/work/issues/open/iss-2608300002547145-lapse-pre-tooling-window.md:8 — "found_during: \"cold-reading workstream preparation, 2026-08-28\""
  evidence: .abcd/work/issues/open/iss-2608300002547145-lapse-pre-tooling-window.md:10 — "lapsed_at: \"2026-08-28T00:00:00Z\""

Gap audit:
- honoured:
  - lapse is a value in capture's validated category list, judged by the same strict path every issue takes
    evidence: internal/core/issueschema/issueschema.go:89 — "\"lapse\","
    evidence: internal/core/capture/parse_test.go:152 — "func TestValidateStrictLapseCategory(t *testing.T) {"
  - A lapse entry carries the instant the discipline gave way: lapsed_at is required exactly for category lapse, must parse as RFC 3339, and both gates read the one definition in core/issueschema
    evidence: internal/core/issueschema/issueschema.go:51 — "\"lapsed_at\": true,"
    evidence: internal/core/capture/validate.go:117 — "if issueschema.LapsedAtRequired(fm[\"category\"].(string)) && lapsedAt == \"\" {"
    evidence: internal/core/lint/schema.go:595 — "case lapsedAt != \"\" && !issueschema.ValidLapsedAt(lapsedAt):"
    evidence: internal/core/lint/schema_parity_test.go:197 — "func TestIssueRecordShapeFlagsLapseWithoutLapsedAt(t *testing.T) {"
    evidence: internal/core/capture/parse_test.go:210 — "func TestValidateStrictLapsedAtMustBeRFC3339(t *testing.T) {"
    evidence: internal/core/capture/parse_test.go:225 — "func TestValidateStrictNonLapseAcceptsAbsentLapsedAt(t *testing.T) {"
  - The timestamp is at the lapse rather than at write-up: --lapsed-at has no default, a lapse capture omitting it exits non-zero and writes nothing, and the given instant round-trips unaltered
    evidence: internal/surface/cli/cli.go:2432 — "captureCmd.Flags().StringVar(&lapsedAt, \"lapsed-at\", \"\", ..."
    evidence: internal/surface/cli/cli.go:2403 — "return &exitError{Code: 2, Msg: \"abcd capture --category \" + issueschema.CategoryLapse +"
    evidence: internal/core/capture/workflow.go:162 — "if lapsedAt := strings.TrimSpace(req.LapsedAt); lapsedAt != \"\" {"
    evidence: internal/surface/cli/capture_surface_test.go:485 — "func TestCaptureLapsedAtWritesTheGivenInstant(t *testing.T) {"
    evidence: internal/core/capture/serialize_test.go:201 — "func TestLapsedAtRoundTrips(t *testing.T) {"
  - The point in the process is carried by found_during, already required and refused when blank
    evidence: internal/core/capture/validate.go:73 — "return fmt.Errorf(\"%w: found_during must be non-empty\", ErrMalformedFrontmatter)"
    evidence: internal/core/capture/parse_test.go:239 — "func TestValidateStrictLapseCarriesFoundDuring(t *testing.T) {"
  - The flag is documented on the plugin surface and in the regenerated CLI reference
    evidence: commands/capture.md:51 — "`--category lapse` **requires** `--lapsed-at`: the flag has no default, and a"
    evidence: docs/reference/cli/commands.md:169 — "--lapsed-at string      RFC 3339 instant a discipline gave way (the lapse, not the write-up)"
  - The first three lapse entries (pre-tooling window, anticipation, outside-tooling commitments) exist in the ledger as category lapse with found_during and lapsed_at
    evidence: .abcd/work/issues/open/iss-2608300002547145-lapse-pre-tooling-window.md:6 — "category: \"lapse\""
    evidence: .abcd/work/issues/open/iss-2608300002540238-lapse-anticipation.md:10 — "lapsed_at: \"2026-08-28T00:00:00Z\""
    evidence: .abcd/work/issues/open/iss-2608300002544153-lapse-outside-tooling-commitments.md:8 — "found_during: \"cold-reading workstream preparation, 2026-08-27/28\""
- diverged:
  - The spec says the first three entries are 'each an ordinary abcd capture call ... with a --lapsed-at carrying the instant'; in the delivered range the entries pre-existed without lapsed_at and were back-stamped in a separate chore commit (267e2c74) with midnight-UTC day-granularity values, so the instant recorded is a day reconstructed at stamping rather than a captured moment
    evidence: .abcd/work/issues/open/iss-2608300002547145-lapse-pre-tooling-window.md:10 — "lapsed_at: \"2026-08-28T00:00:00Z\""
    evidence: .abcd/work/issues/open/iss-2608300002557321-lapse-inferred-go-ahead.md:10 — "lapsed_at: \"2026-08-29T00:00:00Z\""
    evidence: .abcd/work/issues/open/iss-2608300002557321-lapse-inferred-go-ahead.md:13 — "Timestamp: 2026-08-29 (at the lapse; written up 2026-08-30 at the restart)."
  - The intent scopes 'the first three entries'; the delivered ledger carries four category-lapse records (a fourth, lapse-inferred-go-ahead, is additional to the three named)
    evidence: .abcd/work/issues/open/iss-2608300002557321-lapse-inferred-go-ahead.md:6 — "category: \"lapse\""
  - The one-line committed-ledger mirror the spec describes grew into a multi-case gate (padded, list-valued, map-valued, block-spelled lapsed_at) across four fix commits, each disclosed as an issue record rather than absorbed silently
    evidence: internal/core/lint/schema.go:574 — "if hasLapseField && lapsedAt == \"\" && strings.TrimSpace(lapseField.value) == \"\" {"
    evidence: internal/core/lint/schema.go:594 — "add(lapseField.line, \"lapsed_at is spelled as an indented block; ..."
    evidence: .abcd/work/issues/open/iss-2608300212513349-itd-182-review-residue.md:1 — "---"
- missing:
  - No delivered mechanism can establish that a lapsed_at instant was recorded at the lapse rather than chosen at write-up; the gates enforce presence and RFC 3339 shape only, and the spec places that judgement out of scope
    evidence: internal/core/issueschema/issueschema.go:123 — "func ValidLapsedAt(v string) bool {"
    evidence: internal/core/lint/schema.go:596 — "add(lapseField.line, \"lapsed_at '\"+lapsedAt+\"' is not an RFC 3339 instant (want 2026-08-28T00:00:00Z); capture refuses the record and skips it\")"

## Grounds

- pursued: Pursued now because the lapse log is not merely a disclosure obligation: the working claim under test in this cycle is that recording at the point of commitment prevents retrospective reconstruction, and the lapse log is the evidence bearing on that claim. A log opened after the cycle's lapses have already accumulated is reconstruction, and so cannot be evidence about reconstruction.
