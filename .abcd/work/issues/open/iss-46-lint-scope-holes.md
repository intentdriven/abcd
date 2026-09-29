---
schema_version: 1
id: "iss-46"
slug: "lint-scope-holes"
severity: "minor"
category: "process"
source: "agent-finding"
found_during: "2026-07-08 multi-agent review"
found_at: "Makefile"
deferred_after: "v0.11.1"
deferral_reason: "Re-anchored to v0.11.1 by autonomous run A (lane drainDQ1, 2026-09-29), reason unchanged and re-checked at 1ded82939: four of the five holes are closed (link-lint walks every committed markdown file, docs-lint arms the persona rule, make preflight runs fmt-check and record-lint armed as CI does on CI's toolchain, and CONTRIBUTING.md documents the hooks' activation and the banlist the name guard reads). What remains is a general warn-baseline ratchet in record-lint and the scope matrix beyond links; the ratchets that exist (the prose-citation baseline, the site's unresolved-reference baseline) are per-rule, not the general one. It is a design that needs a ruling: which warn rules freeze their current findings, where the frozen baseline lives, and how it may only shrink."
---

lint scope holes and gate parity: link-lint does not cover all committed markdown; the persona rule is absent from docs-lint; record-lint blocking semantics are inconsistent between local and CI and there is no warn-baseline ratchet; gofmt is missing from make preflight though attributed to it; repo-local hooks activation and its provisioning dependency are undocumented. Detector (per ratchet-not-big-bang): a lint scope matrix (which rule covers which tree, checked into the record) plus baseline-ratchet support in record-lint so new rules arm immediately against frozen violations. Acceptance corpus: the five holes above.