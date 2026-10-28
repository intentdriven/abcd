---
schema_version: 1
id: "iss-2610262044240923"
slug: "the-manual-hold-draft-relies-on-the"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 1204 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/drafts/itd-57-manual-hold-sentinel.md:17"
remedy: "none (filed automatically)"
related_intents: [itd-57, itd-2609211116005482]
finding_ref: cfk-7fa84762ad78cd76
---

The manual-hold draft relies on the selector skipping specs with a non-done dependency, but the shipped pick selects planned intents by their blocked_by edges

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/drafts/itd-57-manual-hold-sentinel.md:17` — "the run seam's selector already skips specs blocked by a non-done dependency, which does the rest"
- End B: `.abcd/development/intents/shipped/itd-2609211116005482-abcd-build-next-picks-the-readiest.md:42` — "every intent in `planned/` that passes every refusal check `abcd build <itd-N>` runs before it starts"

itd-57 holds work by giving a spec a dependency on a sentinel spec, on the premise that the run seam's selector skips specs blocked by a non-done dependency. itd-2609211116005482 defines the candidate set as intents in `planned/` passing the build's refusal checks, including not held and nothing unshipped in `blocked_by`. Spec-to-spec dependencies are not what the shipped selector reads, so the sentinel would hold nothing.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 1204.
