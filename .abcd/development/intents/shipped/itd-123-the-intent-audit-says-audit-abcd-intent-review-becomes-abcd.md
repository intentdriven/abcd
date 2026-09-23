---
id: itd-123
shipped_in: v0.6.0
spec_id: spc-28
kind: standalone
suggested_kind: null
reclassification_history: []
builds_on: []
severity: major
impact: breaking
slug: the-intent-audit-says-audit-abcd-intent-review-becomes-abcd
---

# The Intent Audit Says Audit

## Press Release

The verb that answers the product thinker's question finally says so.
"'Review' told me someone judged the code. What this verb actually answers is
whether I got what I asked for — that's an audit, and now it's named as mine,"
says Iris, product lead. `abcd intent audit` emits the same per-criterion
verdicts it always did; what changed is that the name now tells you whose
question it answers.

## Why This Matters

`abcd intent review` emits family 2 (`MET` / `MET_WITH_CONCERNS` / `NOT_MET` /
`INCONCLUSIVE`) — per adr-40 that comparison is an audit, and the brief's own
mental model already calls it "the intent audit". The name crossed with
itd-85's verb, so a maintainer cannot reason from verb to bucket. This is
adr-40's first named rename: clean break, no aliases, no deprecation shims —
abcd is pre-1.0.0, `--impact breaking` drives version derivation, users
re-download. About 37 files reference the current name. The
`intent-fidelity-reviewer` agent becomes `intent-auditor` (the intent-grain
auditor of the three audit grains), and the `intent_review` task-class token
becomes `intent_audit`.

## Scope Conditions

None stated.

## Acceptance Criteria

> _BDD format, per the itd-1 discipline. Walked and confirmed by the
> maintainer, 2026-08-16._

- **Given** the rename, **then** `abcd intent audit [<itd-N>]` and
  `abcd intent audit ingest --verdict-json <file>` replace the `review`
  spellings, and the old sub-verb is gone — no alias, no shim;
  `abcd intent review` fails as an unknown sub-command.
- **Given** the sweep, **then** every live reference moves:
  `commands/intent.md`, brief prose, the agent file and its registered name
  (`intent-fidelity-reviewer` → `intent-auditor`), the `intent_review`
  task-class token → `intent_audit` with its `04-naming.md`
  reserved-vocabulary row updated in the same change, code, tests, and the
  verb's JSON field names.
- **Given** the sweep boundary, **then** historical and dated records are not
  rewritten — ADRs, dated research notes, resolved issues, shipped intents,
  and `DECISIONS.md` keep the old name as history; the brief (current-state
  per adr-5), commands, code, and `docs/` move.
- **Given** itd-122's extended `surface_coverage` armed, **then** the
  `intent` sub-verb table row flips in the same change and `record-lint`
  exit 0 proves the migration complete; landing this rename before that
  check is armed is forbidden.
- **Given** stored artefacts (receipts, previously ingested verdicts),
  **then** they are not rewritten, and any parser that reads them continues
  to accept them.
- **Given** the release record, **then** the CHANGELOG entry is breaking and
  version derivation reads it.

## SOTA

Breaking CLI renames without aliases are the pre-1.0.0 norm (semver §4;
established precedent in this repo via iss-171's `--impact breaking` path);
alias/deprecation machinery was considered and rejected in adr-40 (it
collides with the change-narration ban). **Chosen path: bespoke sweep**,
proved complete by the armed itd-122 gate. No new dependency.

## Open Questions

_None gating. The agent name (`intent-auditor`) was ruled by the maintainer
at the walk, 2026-08-16._

## Audit Notes

<!-- abcd-review: INGESTED receipt=rcp-12cab97fb543 -->
Fidelity review — receipt rcp-12cab97fb543 (verifier abcd:intent-auditor claude-fable-5-1).

Provenance: abcd:intent-auditor@claude-fable-5-1 · rubric_hash sha256:effa65b3e9e88ff29433b443ec2be159522a8b0b71cf1434526514aa61edb13e · prompt_hash sha256:a3bbc2399efb521c82d8b77e10cc87923383c1f194a98c55fc11343d7bdba8d9
Input attestations: diff:tree at da7b7cf409b41758b3502d3873687dc8b817d8c4 (worktree HEAD; the host supplied no commit range, so the whole tree at that commit is the delivered reality)@-;

Acceptance rollup: MET 4 · MET_WITH_CONCERNS 2 · NOT_MET 0 · INCONCLUSIVE 0

Per-criterion verdicts:
- ac-1 — MET: The intent command registers audit and audit ingest --verdict-json, the retired review spelling is refused through the retired-sub-verb guard with the successor named (never swallowed as a free-text create), and the surface test drives both retired shapes to a refusal.
  evidence: internal/surface/cli/cli.go:2227 — "Use: "audit [< itd-N>]","
  evidence: internal/surface/cli/cli.go:2251 — "Use: "ingest --verdict-json < path>","
  evidence: internal/surface/cli/cli.go:3762 — ""intent": {"review": "audit"}, // spc-28 (adr-40)"
  evidence: internal/surface/cli/intent_surface_test.go:126 — "t.Fatalf("intent review must be an unknown sub-command after the rename, got: %v", err)"
- ac-2 — MET_WITH_CONCERNS: The agent file is registered as intent-auditor, the naming register's task-class enum carries intent_audit, the plugin page and the sub-verb table say audit, and no verdict json field carries the review name; concern: the brief's verification matrix still titles the row for this verb 'Intent fidelity review (Role 1)', a residue of the pre-rename vocabulary in current-state prose.
  evidence: agents/intent-auditor.md:2 — "name: intent-auditor"
  evidence: .abcd/development/brief/02-constraints/04-naming.md:180 — "| `task_classes` (capability_scope tokens) ∈ \`{oracle_review, intent_audit, spec_planning,"
  evidence: commands/intent.md:490 — "intent audit < itd-N> --json # re-emit a shipped intent's review request"
  evidence: .abcd/development/brief/04-surfaces/05-intent.md:59 — "| `audit` | audit | shipped |"
  evidence: internal/core/intent/audit.go:120 — "ReceiptID string `json:"receipt_id"`"
  evidence: .abcd/development/brief/06-delivery/02-verification-matrix.md:56 — "| Intent fidelity review (Role 1) |"
- ac-3 — MET: The decision log and the ruling ADR keep the old spelling as history while the live surfaces carry none of it; the only remaining old-name hits outside dated records are closed spec records and one open spec's slug, all records the boundary leaves alone.
  evidence: .abcd/work/DECISIONS.md:1129 — "`intent review` emits family 2 so it is an audit"
  evidence: .abcd/development/decisions/adrs/0040-review-audit-lint-are-three-verbs.md:201 — "- **`abcd intent review` → `abcd intent audit`.** It emits family 2; the brief"
  evidence: agents/CHANGELOG.md:416 — "### intent-auditor 0.1.1 (renamed from intent-fidelity-reviewer)"
- ac-4 — MET_WITH_CONCERNS: The intent sub-verb table carries audit and audit ingest as shipped, the registry README states the surface_coverage rule machine-checks the column, and record-lint exits 0 over the tree at this commit (run by the auditor, warnings only); concern: the landing-order clause (armed before this landed) is a history fact the delivered tree cannot show.
  evidence: .abcd/development/brief/04-surfaces/05-intent.md:59 — "| `audit` | audit | shipped |"
  evidence: .abcd/development/brief/04-surfaces/05-intent.md:60 — "| `audit ingest` | audit | shipped |"
  evidence: .abcd/development/brief/04-surfaces/README.md:46 — "The **Status** column is machine-checked: the `surface_coverage` record-lint rule"
  evidence: Makefile:164 — "record-lint:"
- ac-5 — MET: The verdict payload type and the audit-note marker vocabulary are unchanged constants the ingest parses, and shipped intents ingested before this rename still carry the same marker line the regex matches.
  evidence: internal/core/intent/audit.go:49 — "const VerdictType = "abcd/intent-fidelity-verdict/v1""
  evidence: internal/core/intent/audit.go:95 — "markerRe = regexp.MustCompile(`(?m)^< !-- abcd-review: (OWED|INGESTED|DEAD_LETTER) receipt=(rcp-[0-9a-f]+) -- >\r?$`)"
  evidence: CHANGELOG.md:1114 — "Stored artefacts are format-frozen — the `abcd-review:` audit-note markers, the `abcd/intent-fidelity-verdict/v1` payload type, and every previously ingested verdict remain valid"
- ac-6 — MET: The shipped record declares impact breaking, which is the field version derivation reads, and the 0.6.0 changelog section carries the entry under the breaking heading.
  evidence: .abcd/development/intents/shipped/itd-123-the-intent-audit-says-audit-abcd-intent-review-becomes-abcd.md:10 — "impact: breaking"
  evidence: CHANGELOG.md:1114 — "- **The intent audit says audit.** `abcd intent review` is now `abcd intent audit`"

Gap audit:
- honoured:
  - the verb emits the same per-criterion verdicts under the audit name, with no alias
    evidence: internal/surface/cli/cli.go:3762 — ""intent": {"review": "audit"}, // spc-28 (adr-40)"
  - the agent becomes intent-auditor and the task-class token becomes intent_audit
    evidence: agents/intent-auditor.md:11 — "task_classes: [intent_audit]"
- diverged:
  - every live brief reference moves to the audit vocabulary
    evidence: .abcd/development/brief/06-delivery/02-verification-matrix.md:56 — "| Intent fidelity review (Role 1) |"
- missing: (none)
