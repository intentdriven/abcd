---
id: itd-104
slug: abcd-gates-a-new-idea-before-it-becomes-a-record-entry-resea
spec_id: spc-18
kind: standalone
suggested_kind: null
reclassification_history: []
builds_on: []
severity: minor
impact: additive
---

# **abcd gates a new idea before it becomes a record entry: research it against primary sources, grill it against the existing record, and let an independent adversary try to kill it.** A product thinker's exciting idea is cheapest to kill in its first hour — and most expensive to kill after it has minted intents, specs, and branches. `/abcd:ideate` runs the admission interview: SOTA research in which every load-bearing claim is checked against its primary source; a record grill — does the brief, an intent, an ADR, or a principle already cover, contradict, or supersede this?; and an independent adversarial review, evaluator outside the loop. The output is a verdict plus a recorded decision with rejected alternatives; only a surviving idea graduates to a draft intent. The protocol is already proven by hand: three ideas entered it on 2026-07-14 — two were killed (one by a measured null result its premise could not survive, one by three independently fatal methodological defects) and one was reframed by adversarial review before a line was built. The manual run also named the two load-bearing steps automation must keep: check the record first (one idea was already written and superseded), and open the primary (three secondary-source claims were falsified, one of them fabricated by tool extraction). "The ideas I'm proudest of are the ones abcd killed in an afternoon," said Iris, a product thinker. "Each one left a recorded reason behind, so I never talk myself back into them six months later."

## Press Release

> **abcd gates a new idea before it can mint a single record — researching it against primary sources, grilling it against the existing record, and letting an independent adversary try to kill it.** An idea is cheapest to kill in its first hour and most expensive to kill once it has spawned intents, specs, and branches. `/abcd:ideate` runs the admission interview: state-of-the-art research in which every load-bearing claim is checked against its primary source; a record grill that asks whether the brief, an intent, an ADR, or a principle already covers, contradicts, or supersedes the idea; and an independent adversarial review by an evaluator outside the loop. The output is a verdict and a recorded decision that keeps the rejected alternatives, and only a surviving idea graduates to a draft intent — so the ideas abcd kills leave a reason behind that no one has to re-litigate six months later.

## Why This Matters

**abcd gates a new idea before it becomes a record entry: research it against primary sources, grill it against the existing record, and let an independent adversary try to kill it.** A product thinker's exciting idea is cheapest to kill in its first hour — and most expensive to kill after it has minted intents, specs, and branches. `/abcd:ideate` runs the admission interview: SOTA research in which every load-bearing claim is checked against its primary source; a record grill — does the brief, an intent, an ADR, or a principle already cover, contradict, or supersede this?; and an independent adversarial review, evaluator outside the loop. The output is a verdict plus a recorded decision with rejected alternatives; only a surviving idea graduates to a draft intent. The protocol is already proven by hand: three ideas entered it on 2026-07-14 — two were killed (one by a measured null result its premise could not survive, one by three independently fatal methodological defects) and one was reframed by adversarial review before a line was built. The manual run also named the two load-bearing steps automation must keep: check the record first (one idea was already written and superseded), and open the primary (three secondary-source claims were falsified, one of them fabricated by tool extraction). "The ideas I'm proudest of are the ones abcd killed in an afternoon," said Iris, a product thinker. "Each one left a recorded reason behind, so I never talk myself back into them six months later."

## Acceptance Criteria

- Given a one-line idea, when the user runs the intent or capture verb, then no ideate step is required or suggested as blocking — ideate is an optional verb, never a pre-capture gate; the routing help names it for big, unproven ideas.
- Given /abcd:ideate runs, then it executes three legs in order: primary-source research (every load-bearing claim checked against its primary), a record grill (does the brief, an intent, an ADR, or a principle already cover, contradict, or supersede this?), and an adversarial review that is fresh-context and off-policy — conducted by a session that did not do the research, receiving the idea as an artefact of unknown authorship.
- Given a verdict, then the decision is recorded with rejected alternatives whether the idea survives or dies; only a surviving idea graduates to a draft intent.
- Given the record grill leg runs, then a hit on an existing record entry (covered, contradicted, or superseded) is cited by id in the verdict — the check-the-record-first step that killed one of the three ideas in the proven manual run.

## Open Questions

- The verdict's schema and where it is stored (ledger entry vs a dedicated ideate record family) — a spec-time decision.
- Whether ideate can consume a lifeboat or external document as the idea source.

## Grill Settlements (2026-07-27)

- Optional verb, never a gate: capture friction stays at one line, and the routing help carries the nudge.
- The adversarial leg codifies the fresh-context, off-policy, unknown-authorship pattern — the only measured debiasing effect, per the salvaged 2026-07-14 research record.

## Audit Notes

<!-- abcd-review: INGESTED receipt=rcp-37a162874f75 -->
Fidelity review — receipt rcp-37a162874f75 (verifier abcd:intent-fidelity-reviewer claude-fable-5).

Provenance: abcd:intent-fidelity-reviewer@claude-fable-5 · rubric_hash sha256:1aa9117683cdc84ea4bcbae37c95edab738938331fdb6d39d5d00ea051db4141 · prompt_hash sha256:95792472ae74ca0469f69a51c618946e0d33cb1380032460099ed4b469d67e86
Input attestations: diff:PR 157 main..e05a58ef44b970f7d8db65d7502675bf9a46c3d9: internal/core/ideate/ (ideate.go, record.go, render.go), internal/core/recordid/resolve.go, internal/surface/cli/ideate.go + cli.go ideate wiring, commands/abcd/ideate.md, routing help in commands/abcd/{intent,capture}.md@sha256:e05a58ef44b970f7d8db65d7502675bf9a46c3d9; rubric:.abcd/.work.local/reviews/rcp-37a162874f75.request.md@sha256:1aa9117683cdc84ea4bcbae37c95edab738938331fdb6d39d5d00ea051db4141;

Acceptance rollup: MET 3 · MET_WITH_CONCERNS 1 · NOT_MET 0 · INCONCLUSIVE 0

Per-criterion verdicts:
- ac-1 — MET: The bare intent and capture status renders both print ideateRoutingRule (an optional pointer for a big, unproven idea) with no gate wording, and nothing in the ideate package is reachable from capture or intent-create; TestIdeateRoutingHintIsAPointerNotAGate enforces both facts and passes.
  evidence: internal/surface/cli/cli.go:1255 — "const ideateRoutingRule = \"  a big, unproven idea? `abcd ideate` runs the optional admission gauntlet and records the verdict either way\\n\""
  evidence: internal/surface/cli/cli.go:1119 — "fmt.Fprint(w, ideateRoutingRule)  // bare `abcd intent` render; mirrored at cli.go:1746 for `abcd capture`"
  evidence: internal/surface/cli/ideate_surface_test.go:177 — "for _, forbidden := range []string{\"must\", \"required\", \"before you\", \"warning\"} { ... presents ideate as a gate"
- ac-2 — MET_WITH_CONCERNS: The three legs and their required order are binary-enforced — legOrder is [research, record-grill, adversarial-review] and validate() refuses any leg out of position or carrying another leg's evidence — but the adversarial leg's defining fresh-context/off-policy/unknown-authorship property is delegated to the ideate.md prompt and is by design unverifiable at ingest (validateAdversarial only checks non-empty kill attempts and outcomes), so that debiasing property rests on prompt compliance, not on the recorded artefact.
  evidence: internal/core/ideate/ideate.go:145 — "var legOrder = []LegKind{LegResearch, LegRecordGrill, LegAdversarialReview}"
  evidence: internal/core/ideate/record.go:274 — "if leg.Kind != legOrder[i] { ... the legs run in order (%s), and the order is what each leg is looking at"
  evidence: internal/core/ideate/record.go:367 — "The leg's fresh-context, off-policy, unknown-authorship conduct is the orchestrating prompt's obligation — the binary cannot observe how an agent was run"
  evidence: commands/abcd/ideate.md:71 — "Strip every authorship signal before handing the artefact over ... This is the only measured debiasing effect in the whole protocol"
- ac-3 — MET: Record() writes the durable verdict for every registered outcome (survives/killed/reframed) and refuses a payload whose rejected-alternatives arrive empty by omission (the explicit no_rejected_alternatives marker is required), while graduation is gated on survival — Result.Graduates is verdict==survives and renderOutcome offers the draft-intent path only for survives, ideate minting no intent itself.
  evidence: internal/core/ideate/record.go:405 — "if !p.NoRejectedAlternatives { ... the verdict records no rejected alternatives and does not say so explicitly — nothing was written"
  evidence: internal/core/ideate/record.go:474 — "Graduates: v.verdict == VerdictSurvives,"
  evidence: internal/core/ideate/render.go:119 — "case VerdictSurvives: ... may graduate to a draft intent through\\nthe ordinary quoted-text create (`abcd intent \"<text>\"`). Ideate mints no\\nintent itself"
- ac-4 — MET: resolveCitations builds one recordid.Resolver over the repository and refuses the whole verdict (CitationError, nothing written) if any grill hit cites an id that does not resolve; each hit is shape-checked against CitedIDRe first, and the resolved set is carried into the record as CitedRecords — the check-the-record-first gate, exercised green by the surface test that feeds an unresolvable itd-9999. <!-- record-lint: illustrative -->
  evidence: internal/core/ideate/record.go:454 — "if _, ok := r.Lookup(h.Record); !ok { unresolved = append(unresolved, h.Record); continue }"
  evidence: internal/core/ideate/record.go:348 — "if !recordid.CitedIDRe.MatchString(h.Record) { ... which is not a record id (want adr-N, itd-N, iss-N, or spc-N)"
  evidence: internal/core/recordid/resolve.go:76 — "func NewResolver(repoRoot string) (*Resolver, error)"
  evidence: internal/surface/cli/ideate_surface_test.go:117 — "\"ideate\", \"record\", \"the-ideate-gate\", \"--verdict-json\", writeVerdict(t, ideateVerdictJSON(\"itd-9999\"))" <!-- record-lint: illustrative -->

Gap audit:
- honoured:
  - Ideate is an optional verb named in the routing help, never a pre-capture gate
    evidence: internal/surface/cli/cli.go:1255 — "a big, unproven idea? `abcd ideate` runs the optional admission gauntlet"
    evidence: internal/core/ideate/ideate.go:19 — "NEVER A GATE. Nothing here is reachable from capture or intent-create."
  - The record is written whether the idea survives or dies, and rejected alternatives cannot arrive silently empty
    evidence: internal/core/ideate/render.go:123 — "case VerdictKilled: ... The idea is closed. Before proposing it again, read this record"
    evidence: internal/core/ideate/record.go:405 — "the verdict records no rejected alternatives and does not say so explicitly — nothing was written"
  - Every record-grill hit is cited by id and validated against the live repository, refusing the whole verdict on an unresolvable id
    evidence: internal/core/ideate/record.go:460 — "if len(unresolved) > 0 { ... return nil, &CitationError{Unresolved: unresolved}"
  - Only a survivor graduates; ideate mints no intent itself
    evidence: internal/core/ideate/record.go:474 — "Graduates: v.verdict == VerdictSurvives,"
  - Wired on both planes: the abcd ideate verb (CLI) and the /abcd:ideate command surface
    evidence: internal/surface/cli/cli.go:150 — "root.AddCommand(newIdeateCommand(&asJSON))"
    evidence: commands/abcd/ideate.md:7 — "# `/abcd:ideate` — the idea-admission gauntlet"
- diverged:
  - The adversarial leg's fresh-context/off-policy/unknown-authorship conduct is codified in the orchestrating prompt but not enforceable by the binary at ingest — the recorded artefact cannot prove the evaluator was a fresh, off-policy session
    evidence: internal/core/ideate/record.go:367 — "the binary cannot observe how an agent was run, and pretending to check it would be theatre"
    evidence: commands/abcd/ideate.md:66 — "The evaluator must not be the session that ran legs 1 and 2. Dispatch it as a separate agent with its own context."
- missing: (none)