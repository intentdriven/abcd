---
id: itd-124
shipped_in: v0.6.0
spec_id: spc-29
kind: standalone
suggested_kind: null
reclassification_history: []
builds_on: []
severity: major
impact: breaking
slug: the-conformance-check-calls-itself-lint-abcd-audit-becomes-a
---

# The Conformance Check Calls Itself Lint

## Press Release

The conformance check calls itself what it is. `abcd lint` checks a repo
against the working conventions — deterministic rules, severities, exit codes
— and the word `audit` goes back to meaning "did we do what we said":
`/abcd:audit` returns to its reserved seat for the hash-chain fidelity checks.
"When a verb says lint I know it's checking form — no model, no judgement.
When one says audit I know it's checking a promise. I stopped having to
remember which one lied," says Bob, staff engineer.

## Why This Matters

itd-85 shipped deterministic rule-checking under the one name
`02-constraints/04-naming.md` reserves for itd-16's formal verification. Per
adr-40 the act is a lint, and the maintainer ruled the replacement name
`abcd lint` (2026-08-16, planning question 1): it matches the existing
`record-lint` / `docs-lint` / `lint-reviews` family, with `abcd docs lint`
read as the same word at a narrower scope. Clean break, no aliases —
pre-1.0.0, `--impact breaking` drives version derivation, users re-download.
About 57 references carry the current name; the scoped sweep is smaller (the
word "audit" in its family-2 sense — the intent audit — and third-party
senses never move).

## Scope Conditions

None stated.

## Acceptance Criteria

> _BDD format, per the itd-1 discipline. Walked and confirmed by the
> maintainer, 2026-08-16._

- **Given** the rename, **then** `abcd lint` replaces `abcd audit` with
  identical behaviour, flags, and the tri-state exit contract, and
  `abcd audit` fails as an unknown command — no alias, no shim.
- **Given** the plugin surface, **then** `commands/lint.md` replaces
  `commands/audit.md` (`/abcd:lint`), the `04-surfaces` registry row and
  surface file move, and `/abcd:audit` stands reserved for itd-16 again —
  its two `04-naming.md` listings intact.
- **Given** the sweep boundary, **then** historical and dated records keep
  the old name; the brief, commands, code, tests, `docs/`, and rules-loader
  references move. The scoped patterns are `abcd audit`, `/abcd:audit`, and
  the itd-85 surface's identifiers — never the intent audit or third-party
  senses of the word.
- **Given** anything abcd writes into other repos (managed-repo scaffolding,
  `prepare-this-repo` instructions), **then** it emits the new verb, and the
  breaking CHANGELOG entry names the re-download/re-scaffold step for
  managed repos.
- **Given** itd-122's extended `surface_coverage` armed, **then** the
  registry and sub-verb rows move in the same change — `record-lint` exit 0
  proves the migration complete; landing before the check is armed is
  forbidden.
- **Given** the task-class tokens, **then** conformance work carries `lint`,
  never `audit`, and the enum rows in `04-naming.md` are re-checked in the
  same change (adr-40's consequence).
- **Given** the release record, **then** the CHANGELOG entry is breaking.

## SOTA

Same family as itd-123: pre-1.0.0 breaking rename, no aliases (adr-40;
iss-171 precedent). Package ruling after adversarial review (2026-08-16):
the conformance code moves to `internal/core/repolint` — merging the two
lint engines was reviewed and deliberately deferred to iss-251, a future
consolidation intent, never smuggled into a rename. **Chosen path: bespoke
sweep**, proved complete by the armed itd-122 gate. No new dependency.

## Open Questions

_None gating. The engine-consolidation question is captured as iss-251._

## Audit Notes

<!-- abcd-review: INGESTED receipt=rcp-76175a58e4f7 -->
Fidelity review — receipt rcp-76175a58e4f7 (verifier abcd:intent-auditor claude-fable-5-1).

Provenance: abcd:intent-auditor@claude-fable-5-1 · rubric_hash sha256:effa65b3e9e88ff29433b443ec2be159522a8b0b71cf1434526514aa61edb13e · prompt_hash sha256:5bcc47af70779fe87bd6cd3824fa540c602728c19033e3bc05c5fa1543f34864
Input attestations: diff:tree at da7b7cf409b41758b3502d3873687dc8b817d8c4 (worktree HEAD; the host supplied no commit range, so the whole tree at that commit is the delivered reality)@-;

Acceptance rollup: MET 5 · MET_WITH_CONCERNS 2 · NOT_MET 0 · INCONCLUSIVE 0

Per-criterion verdicts:
- ac-1 — MET: lint is the registered command, the conformance core states the tri-state exit semantics and the surface tests exercise exit 0, exit 2 and the usage path, and a dedicated test proves audit is an unknown command at exit 2.
  evidence: internal/surface/cli/cli.go:440 — "Use: "lint","
  evidence: internal/core/repolint/repolint.go:9 — "// severity-to-exit-code semantics (0 clean, 1 warnings, 2 any error), and SARIF"
  evidence: internal/surface/cli/lint_surface_test.go:169 — "func TestAuditRenameCleanBreak(t *testing.T) {"
  evidence: internal/surface/cli/lint_surface_test.go:55 — "func TestLintConformingExitsZero(t *testing.T) {"
- ac-2 — MET: commands/lint.md exists and commands/audit.md does not, the registry row 16 reads /abcd:lint with its own surface file, and the naming register keeps both /abcd:audit listings as the reserved formal-verification seat.
  evidence: commands/lint.md:73 — "adding `abcd-lint:allow` on that line (the earlier `abcd-audit:allow` spelling is"
  evidence: .abcd/development/brief/04-surfaces/README.md:31 — "| 16 | `/abcd:lint` | shipped | Check whether this repo still conforms to the working conventions |"
  evidence: .abcd/development/brief/02-constraints/04-naming.md:45 — "- `/abcd:audit`: formal verification surface, **staged** (itd-16). Reserved, not"
  evidence: .abcd/development/brief/02-constraints/04-naming.md:72 — "| `/abcd:audit` | formal verification surface: hash-chain and Merkle audit trails, fidelity checks"
- ac-3 — MET_WITH_CONCERNS: No live surface, brief page, code path or doc still spells the verb abcd audit or /abcd:audit in the conformance sense, the decision log and older changelog sections keep it as history; concern: the privacy waiver token abcd-audit:allow is kept as an honoured legacy spelling by design, a signed-off carve-out from the identifier sweep the changelog names.
  evidence: internal/surface/cli/lint.go:17 — "// `/abcd:audit` returns to itd-16's reserved hash-chain fidelity surface): it evaluates the"
  evidence: internal/core/repolint/rule_privacy.go:61 — "const auditWaiver = "abcd-audit:allow""
  evidence: .abcd/work/DECISIONS.md:1129 — "`abcd audit` is deterministic rule-checking so it is a lint"
  evidence: CHANGELOG.md:1112 — "The privacy waiver's current spelling is `abcd-lint:allow`, and every committed `abcd-audit:allow` line stays honoured forever."
- ac-4 — MET: The prepare-this-repo page tells the managed repo to run abcd lint, the launch scaffold code carries neither spelling (it writes no conformance step), and the breaking changelog entry names the re-download and re-scaffold step for managed repos.
  evidence: commands/prepare-this-repo.md:104 — "Write the report — the `abcd lint` findings plus these supplements — to the"
  evidence: commands/prepare-this-repo.md:166 — "surfaces render from it. From then on `abcd lint` reports any surface that"
  evidence: CHANGELOG.md:1112 — "**Managed repos:** re-download the binary, and re-run `prepare-this-repo`/`launch scaffold` where a repo's own instructions or CI referenced `abcd audit`."
- ac-5 — MET_WITH_CONCERNS: The registry row and the lint surface's sub-verb table are in place and record-lint exits 0 over the tree at this commit (run by the auditor, warnings only); concern: the landing-order clause (armed before this landed) is a history fact the delivered tree cannot show.
  evidence: .abcd/development/brief/04-surfaces/README.md:31 — "| 16 | `/abcd:lint` | shipped |"
  evidence: .abcd/development/brief/04-surfaces/16-lint.md:30 — "| `outbound` | gate | shipped |"
  evidence: .abcd/development/brief/04-surfaces/README.md:46 — "The **Status** column is machine-checked: the `surface_coverage` record-lint rule"
  evidence: Makefile:164 — "record-lint:"
- ac-6 — MET: The naming register's bucket enum separates lint from audit by what is compared, the task-class enum carries both tokens, and no agent file claims the audit token for conformance work (the conformance check runs with no agent at all).
  evidence: .abcd/development/brief/02-constraints/04-naming.md:172 — "| `bucket` ∈ `{lint, review, audit, gate}` | Assessment-surface classifier, separated by *what is compared to what*"
  evidence: .abcd/development/brief/02-constraints/04-naming.md:180 — "| `task_classes` (capability_scope tokens) ∈ \`{oracle_review, intent_audit, spec_planning, code_rescue, principle_distillation, lifeboat_packing, audit, lint,"
  evidence: agents/docs-currency-reviewer.md:8 — "task_classes: [cross_document_audit]"
- ac-7 — MET: The shipped record declares impact breaking and the 0.6.0 changelog section carries the entry under the breaking heading.
  evidence: .abcd/development/intents/shipped/itd-124-the-conformance-check-calls-itself-lint-abcd-audit-becomes-a.md:10 — "impact: breaking"
  evidence: CHANGELOG.md:1112 — "- **The conformance check calls itself lint.** `abcd audit` is now `abcd lint`"

Gap audit:
- honoured:
  - abcd lint replaces abcd audit with the tri-state exit contract and no alias
    evidence: internal/surface/cli/lint_surface_test.go:174 — "t.Fatalf("abcd audit must be unknown (exit 2), got exit %d stderr %q", code, stderr.String())"
  - /abcd:audit returns to its reserved seat
    evidence: .abcd/development/brief/02-constraints/04-naming.md:63 — "> **Note:** `/abcd:audit` appears both here and in the exemptions above. The two"
  - the conformance core moves to its own package
    evidence: internal/core/repolint/repolint.go:9 — "// severity-to-exit-code semantics (0 clean, 1 warnings, 2 any error), and SARIF"
- diverged:
  - the itd-85 surface's identifiers all move
    evidence: internal/core/repolint/rule_privacy.go:61 — "const auditWaiver = "abcd-audit:allow""
- missing: (none)
