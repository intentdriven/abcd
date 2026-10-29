---
schema_version: 1
id: "iss-2610262030391056"
slug: "meta-s-truth-rule-bans-aspirational"
severity: "nitpick"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 27 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/00-meta.md:9"
remedy: "In the press release's Open Questions section, drop the sentence 'A future intent (likely paired with itd-15 ...) may quantify it' and register the round-trip fidelity floor question in 03-evidence/03-open-questions.md (or carry it as an inline 'Open question (adr-35)' note); retire the stale Pass B bullet in the same edit."
finding_ref: cfk-d8712271d2a7c944
---

Meta's truth rule bans aspirational maybes from the brief; the press release's open questions carry one outside the registered question homes

A premise contradiction (severity nitpick) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/00-meta.md:9` — "the brief records no proposals, no alternatives still under consideration, no aspirational maybes."
- End B: `.abcd/development/brief/01-product/01-press-release.md:51` — "A future intent (likely paired with itd-15 self-dogfooded SOTA audit) may quantify it."

00-meta says the brief records no proposals and no aspirational maybes, and keeps unsettled questions in `03-open-questions.md` or in inline `Open question (adr-35)` notes. The press release carries its own Open Questions section, which neither registers, and it ends on a maybe about a future intent. Under the truth rule that passage has no legitimate home.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 27.
