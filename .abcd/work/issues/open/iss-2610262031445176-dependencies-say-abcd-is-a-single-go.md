---
schema_version: 1
id: "iss-2610262031445176"
slug: "dependencies-say-abcd-is-a-single-go"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 170 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/02-constraints/02-dependencies.md:41"
remedy: "none (filed automatically)"
related_intents: [itd-44]
finding_ref: cfk-b9885c4edb54d4a1
---

Dependencies say abcd is a single Go binary and that flow-next is neither a dependency nor built, but itd-44 treats Python 3.12 and flow-next as existing plumbing.

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/02-constraints/02-dependencies.md:41` — "flow-next is not a dependency and is not built."
- End B: `.abcd/development/intents/drafts/itd-44-fourth-intent-kind-decision.md:165` — "(Python 3.12, flow-next, the lint"

02-dependencies says abcd ships as a single Go binary with no required external tools, and that flow-next is not a dependency and is not built. itd-44's out-of-scope list includes migrating 'existing scattered plumbing (Python 3.12, flow-next, the lint orchestrator, the Codex backend)' into the ADR store, and its scope works through Python-style internals (`_choose_candidate_kind`, `_PLAN_CANDIDATE_KINDS`). The two documents describe different products.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 170.
