---
id: itd-46
slug: abcd-intent-quoted-text-create-symmetric
spec_id: spc-7
kind: standalone
suggested_kind: standalone
reclassification_history: []
related_adrs: []
builds_on: [itd-4]
severity: minor
impact: additive
---

# `/abcd:intent "<text>"` And `/abcd:capture "<text>"` Become Symmetric Create Paths

## Press Release

> **abcd's two ledger commands gain a symmetric, sub-verb-free create path.** Typing `/abcd:intent "I want users to feel the loyalty card respects their time"` files the new intent directly — no `new` sub-verb required. Typing `/abcd:capture "the export button looks dead-on dark mode"` works the same way for a quick observation. The bare command (zero arguments) keeps doing what the universal abcd convention says it must: shows status and help, never mutates state. The three invocation shapes are crisp: bare → status, quoted text → create, sub-verb [+ID] → act on existing. The product thinker reaches for the same shape no matter which ledger they want.
>
> "I stopped having to remember `new`," said Iris, product lead. "Quotes mean create. That's the whole rule for both commands. The cognitive overhead of `/abcd:intent new "…"` vs `/abcd:capture "…"` was tiny, but it was the difference between a flow that felt designed and a flow that felt grown — and now it feels designed."

## Why This Matters

A 2026-05-22 design discussion, recorded in a dated working-log entry, landed on a clear conclusion: **drop the `new` sub-verb from both `/abcd:intent` and `/abcd:capture`.** Quoted-text-as-create-signal is symmetric across the two commands and aligned with abcd's universal "bare = status + help, never mutates" convention. The full rationale is recorded in that working-log entry; this intent captures the work to implement it.

Three reasons the change matters beyond aesthetics:

1. **It honours the universal convention.** `docs/reference/commands.md` line 3
   states: "Every command, when called with no arguments, shows help and a
   status summary… Bare invocation never mutates state; every state-changing
   operation requires an explicit sub-verb." The current `/abcd:capture` is
   already shaped correctly (`/abcd:capture "<text>"` is the documented
   create path). `/abcd:intent` is the asymmetric outlier with its `new`
   sub-verb. This intent removes the asymmetry.
2. **It clarifies the bridge between the two commands.** `/abcd:capture promote
   <iss-N>` already hands an issue to `/abcd:intent new` as a seed. With the
   sub-verb removed, the promote path calls `/abcd:intent "<text>"` directly,
   and the conceptual model becomes "promote means: capture's content becomes
   intent's seed" rather than "promote means: invoke a specific sub-verb on
   intent". Cleaner mental model.
3. **It is a small, well-scoped surface change with no behaviour change at
   the *output* level.** The artefact a `/abcd:intent "<text>"` invocation
   produces is byte-identical to what `/abcd:intent new "<text>"` produced.
   Only the routing logic and the docs change.

This intent is project-agnostic in the same sense the rest of the command surface is — it shapes how the `/abcd:intent` and `/abcd:capture` commands behave in every project that installs abcd.

## What's In Scope

- **Update `abcd-cli/commands/abcd/intent.md`** — remove the `new` row from the
  sub-verb table; add a `"<text>"` row with the same description; update the
  routing logic so the first argument being a quoted string (not a known
  sub-verb) dispatches to the create path.
- **Update `abcd/docs/reference/commands.md`** `/abcd:intent` table — remove
  the `new "<text>"` row, replace with a `"<text>"` row carrying identical
  description.
- **Update `/abcd:capture promote` plumbing** — the promote path currently
  hands content to `/abcd:intent new`; change it to call `/abcd:intent`
  directly with the issue text as the quoted-text argument.
- **Add a one-line decision rule to both commands' bare-form help output** so
  users know which ledger to reach for:
  - *Half-formed observation, question, or nitpick?* → `/abcd:capture "…"`
  - *User-facing change you want to ship?* → `/abcd:intent "…"`
- **Confirm `/abcd:capture` already follows the shape** — the
  `docs/reference/commands.md` `/abcd:capture` row for `"<text>"` already
  exists; verify the implementation matches the documented shape (per
  a 2026-05-22 working-log entry the implementation is already correct here
  — confirm and move on).

## What's Out Of Scope

- **Renaming `iss-N`/`itd-N` to `C01`/`IN01` or any new ID scheme.** The
  2026-05-22 ledger entry explicitly rejects this as a third partial
  terminology migration (sibling to itd-43's epic→spec sweep).
- **Changing the sub-verb-required rule for mutating sub-verbs.** `refine`,
  `grill`, `plan`, `ship`, `review`, `reclassify`, `link`, `resolve`,
  `wontfix`, `promote` continue to require their target's ID. Corpus-scope
  scan verbs (`list`, `consistency`, `shape`) continue not to require an ID.
  Already-correct shape; this intent does not modify it.
- **Merging `/abcd:intent` and `/abcd:capture` into one command.** The
  ledger entry explicitly rejects a unified `/abcd:capture <kind>` design;
  this intent implements the conclusion of that discussion, not the
  rejected alternative.

## Acceptance Criteria

- *Given* a contributor running `/abcd:intent "I want users to feel X"`, *when* the command executes, *then* a new intent file is created under `intents/drafts/` with content seeded from the quoted text — byte-identical to what `/abcd:intent new "I want users to feel X"` would have produced.
- *Given* a contributor running `/abcd:intent new "I want users to feel X"`, *when* the command executes, *then* it either (a) routes to the create path as before (backwards-compatible alias), or (b) errors with a clear message naming the new shape. Decide at plan; lean (a) for a transition period, (b) eventually.
- *Given* a contributor running `/abcd:intent` (bare), *when* the command executes, *then* status + help is shown and no file is created — the universal bare-invocation convention is preserved.
- *Given* a contributor running `/abcd:capture promote iss-N`, *when* the command executes, *then* the issue text is handed to the intent create path and a new intent file appears under `intents/drafts/` — same outcome as before, different plumbing.
- *Given* a contributor reading the bare-form help output of either command, *when* they scan the help text, *then* the one-line decision rule ("nitpick → capture; user-facing change → intent") is visible.

## Open Questions

- **Backwards-compatibility window for `/abcd:intent new`.** Keep it as an
  alias for one phase to ease the transition, then remove it? Or remove
  immediately on landing? Lean: alias-for-now with a deprecation warning,
  remove in a later sweep.
- **Documentation rewrite extent.** Updating `commands.md` is trivial; the
  question is whether the supporting docs (the `facilitator.md` role doc,
  the `intent.md` skill files, any tutorial content) need parallel
  updates. Likely yes — record at plan.
- **Promote path testing.** `/abcd:capture promote <iss-N>` is a relatively
  rarely-exercised path; regression tests for it should land with this
  intent's spec, not as a side note.

## Related

- **a dated working-log entry (2026-05-22)** — the design discussion that
  produced this intent.
- **`docs/reference/commands.md`** — the doc that gets updated.
- **`commands/abcd/intent.md`** — the command-routing surface.
- **itd-4** (issue-capture intent) — the broader intent for `/abcd:capture`'s
  existence; this intent refines `/abcd:intent` to match `/abcd:capture`'s
  already-correct shape.
- **itd-43** (epic→spec terminology) — precedent for *not* introducing a
  partial terminology migration; this intent preserves `iss-N`/`itd-N`.

## Audit Notes

<!-- abcd-review: INGESTED receipt=rcp-69589b8b5103 -->
Fidelity review — receipt rcp-69589b8b5103 (verifier intent-fidelity-reviewer claude-opus-4-8).

Provenance: intent-fidelity-reviewer@claude-opus-4-8 · rubric_hash sha256:95792472ae74ca0469f69a51c618946e0d33cb1380032460099ed4b469d67e86 · prompt_hash sha256:9163e0a43a54575a50afb718052ae244a6d3763b0b2c8f20d64754457523da9f
Input attestations: diff:7289aa520d72c867de68205d8a2b87c7c05d8321@-;

Acceptance rollup: MET 4 · MET_WITH_CONCERNS 1 · NOT_MET 0 · INCONCLUSIVE 0

Per-criterion verdicts:
- ac-1 — MET: CreateFromText derives a validated slug and atomically writes drafts/itd-N-<slug>.md seeded from the text; both the quoted-text route and the new alias call the identical createIntentFromText, so the artefact is byte-identical, and the adjudication that no prior Go `intent new` verb existed is honest (verified absent in origin/main). Covered by TestCreateFromTextSeedsDraft and TestIntentQuotedTextCreates.
  evidence: internal/core/intent/create.go:59 — "rel := filepath.Join(IntentsRelDir, BucketDrafts, name)"
  evidence: internal/surface/cli/intent_cli_test.go:255 — "if got.ID != \"itd-1\" || got.Bucket != \"drafts\""
- ac-2 — MET: The `new <text>` subcommand is registered as a backwards-compatible alias (lean (a)) that routes to the same createIntentFromText and prints a deprecation warning naming the quoted-text shape on stderr only, keeping stdout identical; proven by TestIntentNewAliasWarnsAndCreates.
  evidence: internal/surface/cli/cli.go:1108 — "\"WARNING: `abcd intent new` is deprecated; use `abcd intent \\\"<text>\\\"` (quoted text is the create signal).\")"
  evidence: internal/surface/cli/intent_cli_test.go:289 — "if !strings.Contains(stderr, \"deprecat\")"
- ac-3 — MET: The create branch is guarded on len(args) > 0; a bare invocation falls through to intent.Status and mutates nothing. TestIntentBareCreatesNothing asserts the status render and zero files under drafts/.
  evidence: internal/surface/cli/cli.go:1070 — "if len(args) > 0 {"
  evidence: internal/surface/cli/intent_cli_test.go:311 — "t.Fatalf(\"bare intent created %d drafts files, want 0\", len(entries))"
- ac-4 — MET_WITH_CONCERNS: The create path is engine-backed and tested end-to-end (ac-1 tests), and commands/abcd/capture.md now instructs promote to hand the issue body to `abcd intent "<text>"`; but promote itself is skill-orchestrated markdown with no native verb and no automated regression test (grep for promote in internal/ finds no promote engine/test), despite the intent's Open Questions asking promote regression tests to land with this spec. The 'same outcome as before' and no-back-link caveats are honestly stated in the markdown.
  evidence: commands/abcd/capture.md:75 — "It hands the issue body to the intent create path — `abcd intent \"<issue text>\"`"
  evidence: commands/abcd/capture.md:78 — "It does **not** yet write the reciprocal `related_intents` back-link onto the `iss-N` record"
- ac-5 — MET: The ledgerDecisionRule constant (nitpick -> capture; user-facing change -> intent) is printed in both the bare intent status render and the bare capture status render; TestBareHelpsCarryDecisionRule asserts both, and the capture skill markdown carries the same rule.
  evidence: internal/surface/cli/cli.go:1162 — "const ledgerDecisionRule = \"  which ledger? half-formed observation, question, or nitpick -> `abcd capture \\\"…\\\"`; a user-facing change you want to ship -> `abcd intent \\\"…\\\"`\\n\""
  evidence: internal/surface/cli/intent_cli_test.go:327 — "if !strings.Contains(captureOut, \"user-facing change\") || !strings.Contains(captureOut, \"nitpick\")"

Gap audit:
- honoured:
  - Sub-verb-free quoted-text create path files a lint-valid seeded draft under intents/drafts/, symmetric with capture
    evidence: internal/core/intent/create.go:38 — "func CreateFromText(repoRoot, text string) (Intent, error) {"
    evidence: internal/core/intent/create_test.go:88 — "// TestCreateFromTextPassesRecordLint runs the real intent_lifecycle record-lint"
  - `intent new` preserved as a deprecation-warning alias (lean a) with byte-identical stdout
    evidence: internal/surface/cli/cli.go:1096 — "Use:   \"new <text>\","
  - Decision rule visible in both bare helps
    evidence: internal/surface/cli/cli.go:1629 — "fmt.Fprint(w, ledgerDecisionRule)"
- diverged:
  - Scope named updates to commands/abcd/intent.md and docs/reference/commands.md; neither file exists in this tree, so the intent verb family has no plugin markdown surface — the routing/table edits have no native counterpart (recorded honestly as iss-105). The AC-level outcomes are met at the CLI surface instead.
    evidence: .abcd/development/specs/closed/spc-7-abcd-intent-quoted-text-create-symmetric.md:54 — "Two scope bullets name files that do not exist in this tree"
  - Promote is delivered as a markdown instruction pointing at the CLI create path rather than an engine-tested end-to-end flow; the reciprocal related_intents back-link is not written (no engine verb does it today).
    evidence: commands/abcd/capture.md:73 — "skill-orchestrated, not a binary sub-verb. It hands the issue body to the intent"
  - Typo-guard asymmetry accepted: any non-sub-verb first token becomes create text, so a mistyped sub-verb files a draft (recorded as iss-104); not required by any AC.
    evidence: .abcd/development/specs/closed/spc-7-abcd-intent-quoted-text-create-symmetric.md:61 — "Typo-guard asymmetry accepted for now"
- missing:
  - No promote-path regression test landed with the spec despite the intent's Open Questions flagging it should; promote's issue-text-to-create handoff is exercised only via the shared create-engine tests, not a promote-specific test.
    evidence: .abcd/development/intents/shipped/itd-46-abcd-intent-quoted-text-create-symmetric.md:102 — "regression tests for it should land with this intent's spec, not as a side note"
