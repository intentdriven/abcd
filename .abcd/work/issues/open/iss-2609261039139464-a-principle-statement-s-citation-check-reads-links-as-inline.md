---
schema_version: 1
id: "iss-2609261039139464"
slug: "a-principle-statement-s-citation-check-reads-links-as-inline"
severity: "minor"
category: "bug"
source: "review-followup"
found_during: "autonomous run A resumed 2026-09-25: review-principles"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/reading/project.go"
---

A principle statement's citation check reads links as inline [label](target) only: a bare URL, an autolink <https://...> and a reference-style [label][ref] travel into the bundle raw and the lint is silent on all three, while the manifest's exclusion row asserts record handles and links in a principle stay behind.
