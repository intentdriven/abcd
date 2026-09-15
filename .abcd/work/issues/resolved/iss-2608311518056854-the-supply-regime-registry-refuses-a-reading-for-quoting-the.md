---
schema_version: 1
id: "iss-2608311518056854"
slug: "the-supply-regime-registry-refuses-a-reading-for-quoting-the"
severity: "major"
category: "bug"
source: "agent-finding"
found_during: "itd-185 fidelity audit rcp-fe3450ca55ff"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/reading/ingest_regime.go"
resolution: "Ruled twice. The 2026-08-31 degradation of the four semantic signatures from enforce to flag is SUPERSEDED by the ruling of 2026-09-01, which rejects accept-and-flag: a gate that records the hit is still a gate that reads prose. The registry is withdrawn, with the review-flag plumbing it exclusively fed, and the supply-regime gate matches the structural shape only — a reserved name carried as a KEY of the reader's own output, its own fields and the keys of any nested object or list of objects the contract does not define, never the name inside a sentence or a quotation. Keys are compared through the same fold the blankness rules use, then trimmed and lower-cased, so a compatibility respelling of a key still refuses; no fold reaches a value. The reserved-name tables are unchanged and their pinning test is untouched — only WHERE they are matched changed. internal/core/reading/ingest_corpus_test.go carries the thirty-four-case realistic corpus and holds the gate in both directions, with the ruling's own examples as named cases passing as values and refusing as keys. itd-185 ac-5 and ac-9 and its residue, spc-63, the brief chapter, the plugin page, the CLI help and the widening definition are rewritten to the structural rule."
impact: internal
resolved_by:
  intent: "itd-185"
  spec: "spc-63"
  commit: "985fddf934d820f45814a47f654b6fccb70cb156"
---

The supply-regime registry refuses a reading for quoting the document it read, and the rate is high: fourteen of thirty-four realistic reading outputs were refused, all by one failure mode. The disposition detector matches the bare token followed by a colon or equals, and this repository's records carry that token everywhere, so any explicative claim that quotes a record refuses. So do a claim reporting that a clause settles a licensing question, a claim quoting a paper that closes by recommending further study, a claim reporting that a suite scores below a threshold, and a detection reading that one section says a fix is merged while another says it is pending, which is the canonical shape of a detection finding. This is not the NFKC normalisation, which was measured innocent: an A/B over thirty-four cases flipped five outcomes and every one was the registry's own phrase in compatibility code points, with no innocent case newly refused. The noise predates it and is structural, because the detectors cannot distinguish a reading that PROPOSES from a reading that REPORTS someone else proposing, and the second is most of what a reading legitimately does. The intent records signature noise as its open question, but the disclosed residue names only under-catching; on this evidence over-catching is the larger risk and the residue should say so.

## Grounds

- pursued: a detector that cannot separate a reading proposing from a reading reporting somebody else proposing belongs on the observed side of the line until it can, and the flags a first real reading raises are the calibration to reconsider it on; a real reading whose findings survive enforcement, or a flagged corpus that is mostly genuine breaches, would show this wrong
- pursued: a detector that cannot separate a reading proposing from a reading reporting somebody else proposing should not read prose at all, and the item's own key set is the unambiguous shape it can read instead; a corpus of honest reports refused by the structural rule, or a real decision field that lands because it was written as prose rather than as a field, would show this wrong
