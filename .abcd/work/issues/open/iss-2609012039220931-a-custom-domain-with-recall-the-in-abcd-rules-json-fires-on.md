---
schema_version: 1
id: "iss-2609012039220931"
slug: "a-custom-domain-with-recall-the-in-abcd-rules-json-fires-on"
severity: "minor"
category: "observation"
source: "review-followup"
found_during: "autonomous-run-2026-09-01"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/rules/rules.go"
deferred_after: "v0.11.1"
deferral_reason: "ruling owed to the product thinker (away; run A 2026-09-25; rulings-owed I): Stop-word recall terms: refuse or warn, and against which list (decided with the WRITING domain, iss-2609200618138053)?"
remedy: "Waits on the stop-word recall ruling (decided with iss-2609200618138053): if refused: the rules loader skips a domain any of whose single-token recall terms is a stop word, with the same stderr diagnostic an empty domain gets; if warned: it loads the domain and names the term in that diagnostic and in abcd rules; either way the list is a short embedded English set seeded from the Snowball stop list, multi-word phrases stay legal, and a loader test covers recall ['the'] and passes every bundled domain."
---

A custom domain with `"recall": ["the"]` in `.abcd/rules.json` fires on ordinary English prose (reproduced at v0.7.0: it injected on "update the roadmap"), so a repository can make any domain — and any override text — inject on nearly every prompt. Recall matching (`termHit`, internal/core/rules/rules.go) is word-bounded and stemmed but has no notion of a keyword too common to be a signal. Whether to refuse or warn on stop-word recall terms, and against which list, is a policy question rather than a mechanical fix; captured so the decision has a marker. Sibling of GHSA-22f8-qf5r-gjgq: the provenance marker that fix adds makes such a domain visible as a repo override but does not stop it firing.

## Remedy grounds (2026-09-29)

- The Snowball project publishes an English stop-word list built for the same stemming family recall matching uses (https://snowballstem.org/algorithms/english/stop.txt, checked 2026-09-30).
- The loader already reports a skipped domain on stderr, so either answer reuses that seam. Rejected: frequency-based detection over a corpus, which adds a data file with no gain over a fixed list for a single-token check.
