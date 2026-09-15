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
---

A custom domain with `"recall": ["the"]` in `.abcd/rules.json` fires on ordinary English prose (reproduced at v0.7.0: it injected on "update the roadmap"), so a repository can make any domain — and any override text — inject on nearly every prompt. Recall matching (`termHit`, internal/core/rules/rules.go) is word-bounded and stemmed but has no notion of a keyword too common to be a signal. Whether to refuse or warn on stop-word recall terms, and against which list, is a policy question rather than a mechanical fix; captured so the decision has a marker. Sibling of GHSA-22f8-qf5r-gjgq: the provenance marker that fix adds makes such a domain visible as a repo override but does not stop it firing.
