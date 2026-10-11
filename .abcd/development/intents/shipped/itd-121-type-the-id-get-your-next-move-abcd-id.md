---
id: itd-121
spec_id: spc-26
kind: standalone
suggested_kind: null
reclassification_history: []
builds_on: []
supersedes: [itd-20]
severity: major
impact: additive
slug: type-the-id-get-your-next-move-abcd-id
---

# Type The Id, Get Your Next Move

## Press Release

Type the id, get your next move. `abcd itd-119` answers *what is this and what
happens next* — a planned intent whose spec body is written: implement it, then
`abcd spec close spc-24`. The twelve-step walk stops being a directory hunt:
every record names its own next verb, and nobody needs to know that shipping an
intent is a `spec` verb. "I stopped keeping the lifecycle in my head — the
record tells me where it stands and what I'd do next," says Nia, facilitator.

## Why This Matters

The record walk is observable only to someone who already knows the verb map.
`iss-`, `itd-`, `spc-` (and `adr-`) prefixes are globally unique and
regex-validated, so the id alone is the routing. The mental model, per the
process-coherence plan: bare `/abcd` answers *what can I do*; `abcd <id>`
answers *what is this, and what is my next move*. SD001-safe — a positional on
the namespace root is not a `show` sub-verb. Duplicate-checked against itd-86
(a blind document-review pass) and itd-112 (the startup banner): no overlap.

## Scope Conditions

None stated.

## Acceptance Criteria

> _BDD format, per the itd-1 discipline. Walked and confirmed by the
> maintainer, 2026-08-16._

- **Given** `abcd <id>` where the positional matches
  `^(iss|itd|spc|adr)-[0-9]+$`, **when** it runs, **then** the record is
  located in its store (any status folder or bucket) and rendered read-only —
  what it is, its key fields and links, and the next move. Zero writes, ever.
- **Given** an `itd-N`, **then** the next move follows the lifecycle:
  `drafts/` → planning interview + `intent plan`; `planned/` with a `_Draft:`
  spec body → write the spec body (path shown); ready (the `intent ready`
  checks pass) → implement; `shipped/` → audit state shown; `superseded/` →
  pointer to the superseding record.
- **Given** an `iss-N`, **then**: open and unpromoted → `capture promote` /
  `resolve` / `wontfix`; promoted → the `itd-N` it graduated into; resolved or
  wontfix → the trail, including `resolved_by` when present.
- **Given** an `spc-N`, **then**: open with its linked intent ready →
  implement against the spec body, then `spec close`; closed → done, linked
  intent shown.
- **Given** an `adr-N`, **then** a read-only render — id, status, title,
  path; next move: none, decisions are read.
- **Given** a shape-matching id that exists in no store, **then** a structural
  fault with a diagnostic and non-zero exit.
- **Given** any other positional, **then** the current unknown-command
  behaviour is byte-for-byte unchanged.
- **Given** `--json`, **then** the same facts structured.
- **Given** the next-move mapping, **then** it lives in one Go table and a
  test asserts every recommended verb resolves to a registered command in the
  cobra tree — a rename breaks the test instead of shipping stale advice.
- **Given** the sweep, **then** the root surface page and the `04-surfaces`
  registry document the positional.

## SOTA

Id-dispatch on a root command is a common CLI affordance (`gh issue view
<number>`, `git show <ref>`-style polymorphic lookups, issue trackers' global
id search); nothing importable — the record stores and lifecycles are native.
**Chosen path: bespoke**, a regex-gated positional over the existing store
readers (`capture`, `intent`, `spec`, plus a thin adr reader). No new
dependency.

## Open Questions

_None gating. Extending dispatch to further families (e.g. plans, principles)
is a future consideration, deliberately not an AC._

## Audit Notes

<!-- abcd-review: INGESTED receipt=rcp-e85e544ece36 -->
Fidelity review — receipt rcp-e85e544ece36 (verifier intent-auditor claude-fable-5-1).

Provenance: intent-auditor@claude-fable-5-1 · rubric_hash sha256:effa65b3e9e88ff29433b443ec2be159522a8b0b71cf1434526514aa61edb13e · prompt_hash sha256:1a3bed57f704412fe98cfd99c5d3424c42ed8272fec836dd3864f608e68a7e41
Input attestations: diff:tree at 7c476185 (main lineage, itd-121 shipped)@-;

Acceptance rollup: MET 10 · MET_WITH_CONCERNS 0 · NOT_MET 0 · INCONCLUSIVE 0

Per-criterion verdicts:
- ac-1 — MET: the root command's one-positional branch calls record.Describe, gated by IDRe ^(iss|itd|spc|adr)-[0-9]+$, and renders id, family, status, title, path, links and next moves; TestRootDispatchZeroWrites proves the tree is untouched and TestCaptureLoneRecordIDNeverWrites / TestIntentLoneRecordIDNeverWrites cover the family verbs; all green at BASE
  evidence: internal/core/record/record.go:29 — "var IDRe = regexp.MustCompile(`^(iss|itd|spc|adr)-[0-9]+$`)"
  evidence: internal/surface/cli/cli.go:234 — "if len(args) == 1 {"
  evidence: internal/surface/cli/cli.go:235 — "d, err := record.Describe(cwd, args[0])"
  evidence: internal/surface/cli/root_dispatch_test.go:85 — "func TestRootDispatchZeroWrites(t *testing.T) {"
- ac-2 — MET: describeIntent maps drafts to the planning interview and intent plan, planned to the spec body or implement (re-checked through intent ready), shipped to the audit state, superseded to the superseding record; TestDescribeIntentLifecycleMoves and TestDescribeMultiSpecIntentAndItsClosedSpec are green
  evidence: internal/core/record/record.go:209 — "func describeIntent(repoRoot, id string) (Description, error) {"
  evidence: internal/core/record/record.go:284 — "d.NextMoves = []string{"none — shipped; its audit state lives in the record's Audit Notes"}"
  evidence: internal/core/record/record.go:290 — "d.NextMoves = []string{"read the superseding record: " + target}"
  evidence: internal/core/record/record_test.go:155 — "func TestDescribeIntentLifecycleMoves(t *testing.T) {"
- ac-3 — MET: describeIssue offers capture promote / resolve / wontfix to an open unpromoted issue, the graduated intent to a promoted one, and the trail with resolved_by to a resolved or wontfix one; TestDescribeIssueNextMoves and TestDescribeResolvedIssueShowsTrail are green, and the front door names the ledger it read (TestRecordDispatcherNamesTheLedgerForAnIssue)
  evidence: internal/core/record/record.go:151 — "d.NextMoves = []string{"
  evidence: internal/core/record/record.go:160 — "d.NextMoves = []string{"none — the issue is " + string(iss.Status) + "; the trail is above"}"
  evidence: internal/core/record/record_test.go:62 — "func TestDescribeIssueNextMoves(t *testing.T) {"
  evidence: internal/core/record/record_test.go:115 — "func TestDescribeResolvedIssueShowsTrail(t *testing.T) {"
- ac-4 — MET: describeSpec offers implement-then-spec-close to an open spec whose intent is ready, and for a closed spec names the linked intent (and the sibling specs still open on a multi-spec intent); TestDescribeSpecMoves is green
  evidence: internal/core/record/record.go:358 — "func describeSpec(repoRoot, id string) (Description, error) {"
  evidence: internal/core/record/record.go:388 — "d.NextMoves = []string{"none — closed; the linked intent is " + sp.Intent}"
  evidence: internal/core/record/record_test.go:223 — "func TestDescribeSpecMoves(t *testing.T) {"
- ac-5 — MET: describeADR renders id, status, title and path as family adr with no next move; TestDescribeADRReadOnly and TestDescribeADRAdmitsBothIDVintages are green
  evidence: internal/core/record/record.go:412 — "func describeADR(repoRoot, id string) (Description, error) {"
  evidence: internal/core/record/record.go:450 — "Family: "adr","
  evidence: internal/core/record/record_test.go:266 — "func TestDescribeADRReadOnly(t *testing.T) {"
- ac-6 — MET: a shape-matching id in no store is an error naming the stores searched, which the front door turns into a non-zero exit after the peer consult; TestDescribeUnknownIDFaults and TestRootDispatchUnknownIDFaults are green
  evidence: internal/core/record/record.go:91 — "// the read-only description. A shape-matching id found in no store is an"
  evidence: internal/surface/cli/cli.go:239 — "return peerHeldRefusal(cwd, "", args[0], err)"
  evidence: internal/core/record/record_test.go:351 — "func TestDescribeUnknownIDFaults(t *testing.T) {"
  evidence: internal/surface/cli/root_dispatch_test.go:51 — "func TestRootDispatchUnknownIDFaults(t *testing.T) {"
- ac-7 — MET: TestRootNonIDPositionalUnchanged pins the unknown-command behaviour for any other positional; green at BASE
  evidence: internal/surface/cli/root_dispatch_test.go:65 — "func TestRootNonIDPositionalUnchanged(t *testing.T) {"
- ac-8 — MET: the same Description renders through the shared render helper under --json with next_moves as a field; TestRootDispatchJSONContract decodes it and checks the moves
  evidence: internal/core/record/record.go:58 — "NextMoves []string `json:"next_moves,omitempty"`"
  evidence: internal/surface/cli/root_dispatch_test.go:16 — "func TestRootDispatchJSONContract(t *testing.T) {"
- ac-9 — MET: the recommended verbs are named constants gathered by RecommendedVerbPaths in the record package, TestRecommendedVerbPathsClosed pins the set and TestNextMoveVerbsResolveInLiveTree resolves every one against the live cobra tree; both green
  evidence: internal/core/record/record.go:83 — "func RecommendedVerbPaths() []string {"
  evidence: internal/core/record/record_test.go:367 — "func TestRecommendedVerbPathsClosed(t *testing.T) {"
  evidence: internal/surface/cli/root_dispatch_test.go:129 — "func TestNextMoveVerbsResolveInLiveTree(t *testing.T) {"
- ac-10 — MET: the root surface page carries a Record-id dispatch section, the 04-surfaces registry row for /abcd names the id form, and its chapter documents the positional and its not-found fault
  evidence: commands/abcd.md:60 — "## Record-id dispatch"
  evidence: .abcd/development/brief/04-surfaces/README.md:23 — "| 8 | `/abcd` | shipped | Find out where you are, or what one record id is and what to do with it | [`08-abcd.md`] (08-abcd.md) |"
  evidence: .abcd/development/brief/04-surfaces/08-abcd.md:43 — "**`abcd <record-id>`** takes a single positional matching `iss-N`, `itd-N`,"

Gap audit:
- honoured:
  - the id alone is the routing: one regex-gated positional over the existing store readers, no new dependency
    evidence: internal/core/record/record.go:29 — "var IDRe = regexp.MustCompile(`^(iss|itd|spc|adr)-[0-9]+$`)"
  - zero writes, ever
    evidence: internal/surface/cli/root_dispatch_test.go:85 — "func TestRootDispatchZeroWrites(t *testing.T) {"
  - a rename breaks the test instead of shipping stale advice
    evidence: internal/surface/cli/root_dispatch_test.go:129 — "func TestNextMoveVerbsResolveInLiveTree(t *testing.T) {"
- diverged:
  - the not-found path consults peer worktrees before faulting (itd-2609091416295622) and an issue answer names the ledger it read (iss-2609202053570475), both later refinements of the plain structural fault the criterion states
    evidence: internal/surface/cli/cli.go:239 — "return peerHeldRefusal(cwd, "", args[0], err)"
    evidence: internal/surface/cli/cli.go:246 — "if strings.HasPrefix(args[0], "iss-") {"
- missing: (none)
